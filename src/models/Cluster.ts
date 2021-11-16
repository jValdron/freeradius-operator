import * as k8s from '@kubernetes/client-node';

export class ClusterSpec {
  isDefaultCluster?: boolean;
  replicas: number = 2;
  service: Service;
  defaultVlan?: number;
  certificate: Certificate;
}

export class ClusterStatus {
}

export class Cluster implements k8s.KubernetesObject {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: ClusterSpec;
  status?: ClusterStatus;
}

export class ClusterList implements k8s.KubernetesListObject<Cluster> {
  apiVersion: string;
  items: Cluster[];
  kind: string;
  metadata: k8s.V1ObjectMeta;
}

export class Service {
  type: string;
  loadBalancerIP?: string;
}

export class Certificate {
  fromSecretRef?: k8s.V1SecretReference;
  privateKeyPassword?: string;
  certificates?: {
    ca?: string,
    dh?: string,
    privateKey?: string,
    publicKey?: string
  };
}