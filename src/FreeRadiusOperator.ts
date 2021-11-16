import * as fs from 'fs';
import * as path from 'path';

import * as k8s from '@kubernetes/client-node';
import Operator from '@dot-i/k8s-operator';
import Mustache from 'mustache';

import config from './Config'; 
import { logger } from './utils/Logger';
import { KubeHelpers } from './utils/KubeHelpers';

import { Client, ClientList } from './models/Client';
import { Cluster, ClusterList } from './models/Cluster';
import { Device, DeviceList } from './models/Device';
import { User, UserList } from './models/User';

export default class FreeRadiusOperator extends Operator {
  private apis: {
    apps: k8s.AppsV1Api,
    core: k8s.CoreV1Api,
    customObjects: k8s.CustomObjectsApi
  };

  protected crds: {
    [key: string]: {
      group: string,
      versions: k8s.V1CustomResourceDefinitionVersion[],
      plural: string
    }
  };

  private currentReconciliation: Promise<void> = null;

  private readonly templates: {
    manifests: {
      deployment: string,
      service: string
    },

    authorize: string,
    clients: string,
    radiusd: string,

    mods: { [key: string]: string }
  };

  constructor() {
    super(logger);

    const templatesBasePath = path.resolve(__dirname, '..', 'templates');

    logger.info({
      path: templatesBasePath
    }, 'Reading all templates');

    this.templates = {
      manifests: {
        deployment: fs.readFileSync(path.join(templatesBasePath, 'manifests', 'deployment.yaml.mustache'), 'utf8'),
        service: fs.readFileSync(path.join(templatesBasePath, 'manifests', 'service.yaml.mustache'), 'utf8')
      },

      authorize: fs.readFileSync(path.join(templatesBasePath, 'authorize.mustache'), 'utf8'),
      clients: fs.readFileSync(path.join(templatesBasePath, 'clients.mustache'), 'utf8'),
      radiusd: fs.readFileSync(path.join(templatesBasePath, 'radiusd.conf.mustache'), 'utf8'),
      
      mods: {}
    };

    fs.readdirSync(path.join(templatesBasePath, 'mods-enabled')).forEach((mod) => {
      this.templates.mods[mod] = fs.readFileSync(path.join(templatesBasePath, 'mods-enabled', mod), 'utf8');
    });
  }

  protected async init(): Promise<void> {
    const kc = new k8s.KubeConfig();
    kc.loadFromDefault();

    this.apis = {
      apps: kc.makeApiClient(k8s.AppsV1Api),
      core: kc.makeApiClient(k8s.CoreV1Api),
      customObjects: kc.makeApiClient(k8s.CustomObjectsApi)
    };

    logger.trace({}, 'Setting up watchers');

    const crdsBasePath = path.resolve(__dirname, '..', 'deploy', 'crds');

    this.crds = {
      client: await this.registerCustomResourceDefinition(path.resolve(crdsBasePath, 'client.yaml')),
      cluster: await this.registerCustomResourceDefinition(path.resolve(crdsBasePath, 'cluster.yaml')),
      device: await this.registerCustomResourceDefinition(path.resolve(crdsBasePath, 'device.yaml')),
      user: await this.registerCustomResourceDefinition(path.resolve(crdsBasePath, 'user.yaml'))
    };

    Object.values(this.crds).forEach(async (crd) => {
      await this.watchResource(crd.group, crd.versions[0].name, crd.plural, async (e) => {
        logger.debug({ e }, crd.versions[0].name + 'resource was modified');
        await this.reconcile(e.meta.namespace);
      });
    });
  }

  protected async reconcile(namespace: string): Promise<void> {
    if (this.currentReconciliation)
    {
      logger.trace({
        promise: this.currentReconciliation
      }, 'Reconciliation already in progress; skipping reconciliation');
      return this.currentReconciliation;
    }

    logger.debug({}, 'Reconciliating resources');

    this.currentReconciliation = new Promise((resolve, reject) => {
      this.apis.customObjects.listNamespacedCustomObject(
        this.crds.cluster.group,
        this.crds.cluster.versions[0].name,
        namespace,
        this.crds.cluster.plural
      ).then((response) => {
        const clusters = (<ClusterList>response.body).items;

        Promise.all(clusters.map(cluster => {
          cluster.spec.isDefaultCluster = cluster.spec.isDefaultCluster || clusters.length === 1;
          return this.reconcileCluster(namespace, cluster);
        })).then(() => {
          return resolve();
        }).catch(reject).finally(() => {
          this.currentReconciliation = null;
        });
      });
    });

    return this.currentReconciliation;
  }

  protected async reconcileCluster(namespace: string, cluster: Cluster): Promise<void> {
    logger.info({
      namespace: namespace,
      cluster: cluster
    }, 'Reconciliating cluster');

    return new Promise(async (resolve, reject) => {
      const clients = await KubeHelpers.getClusterResources<Client>(this.apis.customObjects, namespace, cluster, this.crds.client),
            devices = await KubeHelpers.getClusterResources<Device>(this.apis.customObjects, namespace, cluster, this.crds.device),
            users = await KubeHelpers.getClusterResources<User>(this.apis.customObjects, namespace, cluster, this.crds.user);

      await Promise.all(clients.map(client => this.loadClientSecretRef(namespace, client)));
      await Promise.all(users.map(user => this.loadUserSecretRef(namespace, user)));

      logger.debug({
        namespace: namespace,
        cluster: cluster,
        resources: {
          clients: clients,
          devices: devices,
          users: users
        }
      }, 'Sucessfully pulled all cluster resources');

      const values = {
        ...cluster.spec,
        ...{
          clusterName: cluster.metadata.name
        }
      };

      logger.trace({
        namespace: namespace,
        cluster: cluster,
        values: values
      }, 'Rendering manifests and mods using given values');

      await this.loadCertificateSecretRef(namespace, cluster);

      const renderedResources = {
        deployment: <k8s.V1Deployment>k8s.loadYaml(Mustache.render(this.templates.manifests.deployment, values)),
        service: <k8s.V1Service>k8s.loadYaml(Mustache.render(this.templates.manifests.service, values)),
        mods: Object.fromEntries(Object.entries(this.templates.mods).map(
          ([key, mod]) => [key, Mustache.render(mod, values)]
        )),
        authorize: FreeRadiusOperator.renderAuthorize(this.templates.authorize, namespace, cluster, users, devices),
        clients: FreeRadiusOperator.renderClients(this.templates.clients, namespace, cluster, clients),
        radiusd: FreeRadiusOperator.renderRadiusd(this.templates.radiusd, namespace, cluster)
      };

      FreeRadiusOperator.addOwnerRef(renderedResources.deployment.metadata, cluster);
      FreeRadiusOperator.addOwnerRef(renderedResources.service.metadata, cluster);

      logger.debug({
        namespace: namespace,
        cluster: cluster,
        renderedResources: renderedResources
      }, 'Successfully rendered all resources using templates; creating config maps with rendered resources');

      const configResources = await FreeRadiusOperator.generateConfigResources(namespace, cluster, renderedResources);

      Promise.all(configResources.map(res => {
        if (res.kind === 'Secret')
        {
          return KubeHelpers.upsertNamespacedSecret(this.apis.core, namespace, res);
        }
        else
        {
          return KubeHelpers.upsertNamespacedConfigMap(this.apis.core, namespace, res);
        }
      })).then(() => {
        KubeHelpers.upsertNamespacedDeployment(this.apis.apps, namespace, renderedResources.deployment).then(() => {
          KubeHelpers.upsertNamespacedService(this.apis.core, namespace, renderedResources.service).then(() => {
            return resolve();
          }).catch((err: any) => {
            logger.error({
              error: err,
              namespace: namespace,
              cluster: cluster,
              service: renderedResources.service
            }, 'Failed to update the service');
    
            return reject(err);
          });
        }).catch((err: any) => {
          logger.error({
            error: err,
            namespace: namespace,
            cluster: cluster,
            deployment: renderedResources.deployment
          }, 'Failed to update the deployment');
  
          return reject(err);
        });
      }).catch((err: any) => {
        logger.error({
          error: err,
          namespace: namespace,
          cluster: cluster,
          resources: configResources
        }, 'Failed to update all config resources');

        return reject(err);
      });
    });
  }

  private async loadClientSecretRef(namespace: string, client: Client): Promise<Client>
  {
    if (client.spec.fromSecretRef)
    {
      logger.debug({
        client: client,
        namespace: namespace
      }, 'Found secretRef for client; loading in secret');

      return new Promise(async (resolve, reject) => {
        let secretRef: k8s.V1Secret;

        try
        {
          secretRef = <k8s.V1Secret>(await this.apis.core.readNamespacedSecret(
            client.spec.fromSecretRef.name,
            client.spec.fromSecretRef.namespace || namespace)).body;
        }
        catch
        {
          logger.error({
            fromSecretRef: client.spec.fromSecretRef,
          }, 'Invalid client secretRef');

          return reject('Invalid client secretRef');
        }

        try
        {
          client.spec.secret = atob(secretRef.data.secret);
        }
        catch
        {
          logger.error({
            fromSecretRef: client.spec.fromSecretRef,
            secret: secretRef
          }, 'Invalid data keys in client secretRef');

          return reject('Invalid data keys in client secretRef');
        }

        return resolve(client);
      });
    }

    return Promise.resolve(client);
  }

  private async loadCertificateSecretRef(namespace: string, cluster: Cluster): Promise<Cluster>
  {
    if (cluster.spec.certificate.fromSecretRef)
    {
      logger.debug({
        cluster: cluster,
        namespace: namespace
      }, 'Found secretRef for certificate; loading in certificate');

      return new Promise(async (resolve, reject) => {
        let secretRef: k8s.V1Secret;

        try
        {
          secretRef = <k8s.V1Secret>(await this.apis.core.readNamespacedSecret(
            cluster.spec.certificate.fromSecretRef.name,
            cluster.spec.certificate.fromSecretRef.namespace || namespace)).body;
        }
        catch
        {
          logger.error({
            fromSecretRef: cluster.spec.certificate.fromSecretRef,
            cluster: cluster
          }, 'Invalid certificate secretRef');

          return reject('Invalid certificate secretRef');
        }

        try
        {
          cluster.spec.certificate.privateKeyPassword = atob(secretRef.data.privateKeyPassword);
          cluster.spec.certificate.certificates = {
            ca: atob(secretRef.data.ca),
            dh: atob(secretRef.data.dh),
            privateKey: atob(secretRef.data.privateKey),
            publicKey: atob(secretRef.data.publicKey)
          };
        }
        catch
        {
          logger.error({
            fromSecretRef: cluster.spec.certificate.fromSecretRef,
            secret: secretRef
          }, 'Invalid data keys in certificate secretRef');

          return reject('Invalid data keys in certificate secretRef');
        }

        return resolve(cluster);
      });
    }

    return Promise.resolve(cluster);
  }

  private async loadUserSecretRef(namespace: string, user: User): Promise<User>
  {
    if (user.spec.fromSecretRef)
    {
      logger.debug({
        user: user,
        namespace: namespace
      }, 'Found secretRef for user; loading in username/password');

      return new Promise(async (resolve, reject) => {
        let secretRef: k8s.V1Secret;

        try
        {
          secretRef = <k8s.V1Secret>(await this.apis.core.readNamespacedSecret(
            user.spec.fromSecretRef.name,
            user.spec.fromSecretRef.namespace || namespace)).body;
        }
        catch
        {
          logger.error({
            fromSecretRef: user.spec.fromSecretRef,
          }, 'Invalid user secretRef');

          return reject('Invalid user secretRef');
        }

        try
        {
          user.spec.username = atob(secretRef.data.username);
          user.spec.password = atob(secretRef.data.password);
        }
        catch
        {
          logger.error({
            fromSecretRef: user.spec.fromSecretRef,
            secret: secretRef
          }, 'Invalid data keys in user secretRef');

          return reject('Invalid data keys in user secretRef');
        }

        return resolve(user);
      });
    }

    return Promise.resolve(user);
  }

  private static renderAuthorize(template: string, namespace: string, cluster: Cluster, users: User[], devices: Device[]): string
  {
    const values = {
      ...cluster.spec,
      ...{
        clusterName: cluster.metadata.name,
        devices: devices.map(cr => {
          return {
            ...cr.spec,
            ...{
              name: cr.metadata.name,
              macAddresses: cr.spec.macAddresses.map(mac => mac.replace(/\:/g, ''))
            }
          };
        }),
        users: users.map(cr => {
          return {
            ...cr.spec,
            ...{
              name: cr.metadata.name
            }
          };
        })
      }
    };

    logger.trace({
      namespace: namespace,
      cluster: cluster,
      values: values
    }, 'Rendering authorize using given values');

    return Mustache.render(template, values);
  }

  private static renderClients(template: string, namespace: string, cluster: Cluster, clients: Client[]): string
  {
    const values = {
      ...cluster.spec,
      ...{
        clusterName: cluster.metadata.name,
        clients: clients.map(cr => {
          return {
            ...cr.spec,
            ...{
              name: cr.metadata.name
            }
          };
        })
      }
    };

    logger.trace({
      namespace: namespace,
      cluster: cluster,
      values: values
    }, 'Rendering clients.conf using given values');

    return Mustache.render(template, values);
  }

  private static renderRadiusd(template: string, namespace: string, cluster: Cluster): string
  {
    const values = {
      ...cluster.spec,
      ...{
        clusterName: cluster.metadata.name
      }
    };

    logger.trace({
      namespace: namespace,
      cluster: cluster,
      values: values
    }, 'Rendering radiusd.conf file using given values');

    return Mustache.render(template, values);
  }

  private static async generateConfigResources(namespace: string, cluster: Cluster, renderedResources: {
    mods: {
      [key: string]: string
    },
    authorize: string,
    clients: string,
    radiusd: string
  }): Promise<(k8s.V1ConfigMap | k8s.V1Secret)[]>
  {
    const config = new k8s.V1ConfigMap();

    config.kind = 'ConfigMap';
    config.metadata = new k8s.V1ObjectMeta();
    config.metadata.name = `freeradius-${cluster.metadata.name}-config-generated`;
    config.metadata.namespace = namespace;
    config.metadata.labels = {
      app: 'freeradius',
      clusterName: cluster.metadata.name
    };
    FreeRadiusOperator.addOwnerRef(config.metadata, cluster);

    config.data = {
      authorize: renderedResources.authorize,
      'clients.conf': renderedResources.clients,
      'radiusd.conf': renderedResources.radiusd
    }

    const mods = new k8s.V1ConfigMap()

    mods.kind = 'ConfigMap';
    mods.metadata = new k8s.V1ObjectMeta();
    mods.metadata.name = `freeradius-${cluster.metadata.name}-mods-generated`;
    mods.metadata.namespace = namespace;
    mods.metadata.labels = {
      app: 'freeradius',
      clusterName: cluster.metadata.name
    };
    FreeRadiusOperator.addOwnerRef(mods.metadata, cluster);

    mods.data = renderedResources.mods;

    const certificates = new k8s.V1Secret();

    certificates.kind = 'Secret';
    certificates.metadata = new k8s.V1ObjectMeta();
    certificates.metadata.name = `freeradius-${cluster.metadata.name}-certs-generated`;
    certificates.metadata.namespace = namespace;
    certificates.metadata.labels = {
      app: 'freeradius',
      clusterName: cluster.metadata.name
    };
    FreeRadiusOperator.addOwnerRef(certificates.metadata, cluster);

    certificates.data = {
      ca: btoa(cluster.spec.certificate.certificates.ca),
      dh: btoa(cluster.spec.certificate.certificates.dh),
      'server.key': btoa(cluster.spec.certificate.certificates.privateKey),
      'server.pem': btoa(cluster.spec.certificate.certificates.publicKey)
    };
  
    return [config, mods, certificates];
  }

  private static addOwnerRef(meta: k8s.V1ObjectMeta, cluster: Cluster)
  {
    const ownerRef = new k8s.V1OwnerReference();
    ownerRef.apiVersion = cluster.apiVersion;
    ownerRef.controller = true;
    ownerRef.kind = cluster.kind;
    ownerRef.name = cluster.metadata.name;
    ownerRef.uid = cluster.metadata.uid;

    meta.ownerReferences = meta.ownerReferences || [];
    meta.ownerReferences.push(ownerRef);
  }
}
