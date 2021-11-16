import * as k8s from '@kubernetes/client-node';

export class DeviceSpec {
  macAddresses: string[];
  vlan: number;
}

export class DeviceStatus {
}

export class Device implements k8s.KubernetesObject {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: DeviceSpec;
  status?: DeviceStatus;
}

export class DeviceList implements k8s.KubernetesListObject<Device> {
  apiVersion: string;
  items: Device[];
  kind: string;
  metadata: k8s.V1ObjectMeta;
}
