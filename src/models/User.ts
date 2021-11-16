import * as k8s from '@kubernetes/client-node';

export class UserSpec {
  fromSecretRef?: k8s.V1SecretReference;
  username?: string;
  password?: string;
}

export class UserStatus {
}

export class User implements k8s.KubernetesObject {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: UserSpec;
  status?: UserStatus;
}

export class UserList implements k8s.KubernetesListObject<User> {
  apiVersion: string;
  items: User[];
  kind: string;
  metadata: k8s.V1ObjectMeta;
}
