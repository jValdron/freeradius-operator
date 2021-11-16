import * as http from 'http';
import * as k8s from '@kubernetes/client-node';

import config from '../Config';
import { logger } from './Logger';

import { Cluster } from '../models/Cluster';

export class CustomResourceDefinition {
  group: string;
  versions: k8s.V1CustomResourceDefinitionVersion[];
  plural: string;
}

export class KubeHelpers {
  /**
   * Upserts a config map to the given namespace.
   * @param client An instance of CoreV1Api.
   * @param namespace The namespace to operate within.
   * @param object The config map.
   * @returns A promise; resolves when upserted.
   */
  static async upsertNamespacedConfigMap(client: k8s.CoreV1Api, namespace: string, object: k8s.V1ConfigMap): Promise<{
    response?: http.IncomingMessage;
    body?: object;
  }> {
    if (config.DryRun)
    {
      logger.warn({
        namespace: namespace,
        object: object
      }, 'Not upserting config map; running in dry mode');

      return Promise.resolve({});
    }
    else
    {
      return new Promise((resolve, reject) => {
        client.readNamespacedConfigMap(
          object.metadata.name,
          namespace
        ).then((existingObject) => {
          object.metadata.resourceVersion = (<k8s.V1ConfigMap>existingObject.body).metadata.resourceVersion;

          client.replaceNamespacedConfigMap(
            object.metadata.name,
            namespace,
            object
          ).then(resolve).catch(reject);
        }).catch(() => {
          client.createNamespacedConfigMap(
            namespace,
            object
          ).then(resolve).catch(reject);
        });
      });
    }
  }

  /**
   * Upserts a deployment to the given namespace.
   * @param client An instance of AppsV1Api.
   * @param namespace The namespace to operate within.
   * @param object The deployment.
   * @returns A promise; resolves when upserted.
   */
  static async upsertNamespacedDeployment(client: k8s.AppsV1Api, namespace: string, object: k8s.V1Deployment): Promise<{
    response?: http.IncomingMessage;
    body?: object;
  }> {
    if (config.DryRun)
    {
      logger.warn({
        namespace: namespace,
        object: object
      }, 'Not upserting deployment; running in dry mode');

      return Promise.resolve({});
    }
    else
    {
      return new Promise((resolve, reject) => {
        client.readNamespacedDeployment(
          object.metadata.name,
          namespace
        ).then((existingObject) => {
          object.metadata.resourceVersion = (<k8s.V1Deployment>existingObject.body).metadata.resourceVersion;

          client.replaceNamespacedDeployment(
            object.metadata.name,
            namespace,
            object
          ).then(resolve).catch(reject);
        }).catch(() => {
          client.createNamespacedDeployment(
            namespace,
            object
          ).then(resolve).catch(reject);
        });
      });
    }
  }

  /**
   * Upserts a secret to the given namespace.
   * @param client An instance of CoreV1Api.
   * @param namespace The namespace to operate within.
   * @param object The secret.
   * @returns A promise; resolves when upserted.
   */
  static async upsertNamespacedSecret(client: k8s.CoreV1Api, namespace: string, object: k8s.V1Secret): Promise<{
    response?: http.IncomingMessage;
    body?: object;
  }> {
    if (config.DryRun)
    {
      logger.warn({
        namespace: namespace,
        object: object
      }, 'Not upserting secret; running in dry mode');

      return Promise.resolve({});
    }
    else
    {
      return new Promise((resolve, reject) => {
        client.readNamespacedSecret(
          object.metadata.name,
          namespace
        ).then((existingObject) => {
          object.metadata.resourceVersion = (<k8s.V1Secret>existingObject.body).metadata.resourceVersion;

          client.replaceNamespacedSecret(
            object.metadata.name,
            namespace,
            object
          ).then(resolve).catch(reject);
        }).catch(() => {
          client.createNamespacedSecret(
            namespace,
            object
          ).then(resolve).catch(reject);
        });
      });
    }
  }

  /**
   * Upserts a service to the given namespace.
   * @param client An instance of CoreV1Api.
   * @param namespace The namespace to operate within.
   * @param object The service.
   * @returns A promise; resolves when upserted.
   */
  static async upsertNamespacedService(client: k8s.CoreV1Api, namespace: string, object: k8s.V1Service): Promise<{
    response?: http.IncomingMessage;
    body?: object;
  }> {
    if (config.DryRun)
    {
      logger.warn({
        namespace: namespace,
        object: object
      }, 'Not upserting service; running in dry mode');

      return Promise.resolve({});
    }
    else
    {
      return new Promise((resolve, reject) => {
        client.readNamespacedService(
          object.metadata.name,
          namespace
        ).then((existingObject) => {
          object.metadata.resourceVersion = (<k8s.V1Service>existingObject.body).metadata.resourceVersion;

          client.replaceNamespacedService(
            object.metadata.name,
            namespace,
            object
          ).then(resolve).catch(reject);
        }).catch(() => {
          client.createNamespacedService(
            namespace,
            object
          ).then(resolve).catch(reject);
        });
      });
    }
  }

  /**
   * Gets the resources for a given cluster instance.
   * @param client An instance of CustomObjectsApi.
   * @param namespace The namespace to pull from.
   * @param cluster The cluster to obtain resources for.
   * @param crd The custom resource definition of the object to pull.
   * @returns A list of resources for the given cluster instance.
   */
  static async getClusterResources<T extends k8s.KubernetesObject>(client: k8s.CustomObjectsApi, namespace: string, cluster: Cluster, crd: {
    group: string,
    versions: k8s.V1CustomResourceDefinitionVersion[],
    plural: string
  }): Promise<T[]> {
    return new Promise((resolve, reject) => {
      Promise.all([
        new Promise((resolve, reject) => {
          client.listNamespacedCustomObject(
            crd.group,
            crd.versions[0].name,
            namespace,
            crd.plural,
            undefined,
            undefined,
            undefined,
            `clusterName=${cluster.metadata.name}`
          ).then((response) => {
            return resolve((<k8s.KubernetesListObject<T>>response.body).items);
          }).catch(reject);
        }),
        new Promise((resolve, reject) => {
          if (cluster.spec.isDefaultCluster)
          {
            client.listNamespacedCustomObject(
              crd.group,
              crd.versions[0].name,
              namespace,
              crd.plural,
              undefined,
              undefined,
              undefined,
              `!clusterName`
            ).then((response) => {
              return resolve((<k8s.KubernetesListObject<T>>response.body).items);
            }).catch(reject);
          }
          else
          {
            return resolve([]);
          }
        })
      ]).then((items) => {
        const flatten = (ary: any) => ary.reduce((a: T[], b: T[]) => a.concat(Array.isArray(b) ? flatten(b) : b), []);
        return resolve(flatten(items));
      }).catch(reject);
    });
  }
}

