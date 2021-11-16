import * as k8s from '@kubernetes/client-node';

export class ClientSpec {
  ipAddress: string;
  fromSecretRef?: k8s.V1SecretReference;
  secret?: string;
}

export class ClientStatus {
}

export class Client implements k8s.KubernetesObject {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: ClientSpec;
  status?: ClientStatus;
}

export class ClientList implements k8s.KubernetesListObject<Client> {
  apiVersion: string;
  items: Client[];
  kind: string;
  metadata: k8s.V1ObjectMeta;
}
