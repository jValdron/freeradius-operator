# freeradius-operator
An operator that creates instances of FreeRADIUS, and allows setting up clients and users.

## Notice
This comes as-is, with no support or no warranties.

I have very little experience with FreeRADIUS (and RADIUS in general). This was created just to allow me to connect to my APs using WPA enterprise and that's about it.

The scope of the project won't be expanded, at least not by myself, seeing as I have a very limited use-case for FreeRADIUS.

Nevertheless, I've tried to built it as generic as possible, and there is a generous amount of templating available, so that it can be expanded if required.

## Options
Available options via environment variables (or dotenv):
* `LOG_LEVEL`: Optional, overrides the log level: ['trace', 'debug', 'info', 'warn', 'error']
* `DRY_RUN`: Whether or not to enable dry run mode, won't actually create any resources, used for development/debugging.

## Resources
### Cluster
```
---
apiVersion: freeradius.valdron.ca/v1
kind: Cluster
metadata:
  name: primary
spec:
  replicas: 1
  service:
    type: LoadBalancer
    loadBalancerIP: 10.1.21.33
  certificate:
    privateKeyPassword: changeme
    certificates:
      ca: <ca file>
      dh: <dh file>
      privateKey: <server private key>
      publicKey: <server public key>
```

Using `fromSecretRef`, generate secret and reference to the secret with `certificate.fromSecretRef.name` instead of a literal passed as `secret`:
```
kubectl create secret generic \
    --from-file=ca=certs/ca.pem \
    --from-file=dh=certs/dh \
    --from-file=privateKey=certs/server.key \
    --from-file=publicKey=certs/server.pem \
    --from-literal=privateKeyPassword=changeme \
    freeradius-primary-certs
```

#### Multi-clusters
You can label resources with a special label `clusterName` which will create the resources under the cluster matching the given name.

This allows you to have multiple clusters setup. If you only have one cluster, you do not need to specify the `clusterName` label, as it will be treated as the default.

### Client
Using plain text secret:
```
apiVersion: freeradius.valdron.ca/v1
kind: Client
metadata:
  name: node0
spec:
  ipAddress: 10.1.20.1/32
  secret: changeme
```

Using `fromSecretRef`, generate secret and reference to the secret with `fromSecretRef.name` instead of a literal passed as `secret`:
```
kubectl create secret generic \
    --from-literal=secret=changeme \
    freeradius-primary-clients
```

### Device
```
apiVersion: freeradius.valdron.ca/v1
kind: Device
metadata:
  name: phone0
spec:
  macAddresses:
    - aa:bb:cc:dd:ee:ff
  vlan: 1111
```

### User
Using plain text secret:
```---
apiVersion: freeradius.valdron.ca/v1
kind: User
metadata:
  name: john
spec:
  username: john
  password: supersecret
```

Using `fromSecretRef`, generate secret and reference to the secret with `fromSecretRef.name` instead of a literal passed as `secret`:
```
kubectl create secret generic \
    --from-literal=username=john \
    --from-literal=password=supersecret \
    freeradius-primary-users-john
```

## TODO
* Bug: No config reload (add suffix/generator?)
* Add `ownerRef` reconciliation when owned resources are modified
* Cleanup on delete
* Add LDAP support
* Add leader election