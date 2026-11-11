# freeradius-operator

Go ([Kubebuilder](https://book.kubebuilder.io/)) operator for [FreeRADIUS](https://freeradius.org/) on Kubernetes. It reconciles **Cluster**, **Client**, **User**, and **Device** custom resources into a FreeRADIUS **Deployment**, **Service**, and generated configuration.

This project was rewritten from the original Node.js operator into Go with AI assistance. It comes as-is, with no support or warranties.

## Quick start

```bash
make install          # CRDs
make deploy IMG=ghcr.io/jvaldron/freeradius-operator:latest
```

Apply samples (adjust namespaces and secrets):

```bash
kubectl apply -k config/samples/cert-manager/   # requires cert-manager
kubectl apply -f config/samples/freeradius_v1_cluster.yaml
kubectl apply -f config/samples/freeradius_v1_client.yaml
kubectl apply -f config/samples/freeradius_v1_user.yaml
kubectl apply -f config/samples/freeradius_v1_device.yaml
```

## Configuration

| Environment variable | Flag | Default | Description |
| -------------------- | ---- | ------- | ----------- |
| `DRY_RUN` | — | `false` | Skip API writes |
| `WATCH_NAMESPACE` | — | *(empty)* | If set, limit watches to this namespace (`NAMESPACE` is an alias) |

The manager uses **leader election** by default (`--leader-elect`).

## Cluster

Each `Cluster` owns generated ConfigMaps, a Service, Deployment, and optionally a certs Secret. Config or credential changes update a **config-hash** on the pod template so workloads roll automatically.

```yaml
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
    tlsSecretRef:
      name: freeradius-primary-tls
  eap:
    defaultEapType: ttls
    tlsMinVersion: "1.2"
```

### TLS

**Preferred (cert-manager):** `tlsSecretRef` only. The Deployment mounts `tls.crt` / `tls.key` from cert-manager. DH for EAP-TLS uses **OpenSSL built-in parameters** (no `dh_file` in generated config). Use `dhSecretRef` only when you need custom DH params from `openssl dhparam`.

**Legacy:** inline `certificates` or `fromSecretRef` with keys `ca`, `dh`, `privateKey`, `publicKey`, optional `privateKeyPassword`.

See [config/samples/cert-manager/](config/samples/cert-manager/) for an Issuer and Certificate example.

### LDAP (optional)

Set `spec.ldap` to generate `mods-enabled/ldap` and patch `sites-enabled/inner-tunnel` for directory-backed authentication alongside `User` CRs (files).

| Field | Description |
| ----- | ----------- |
| `servers` | One or more hostnames or `ldap(s)://` URIs (rendered as FreeRADIUS multi-host `server`) |
| `port` | Optional; only for plain hostnames (not URIs) |
| `baseDN` / `userBaseDN` / `userFilter` | Directory search configuration |
| `fromSecretRef` | Keys `password` (required), `identity` (optional if `spec.ldap.identity` set) |
| `authMode` | `bindAsUser` (default, PAP / EAP-TTLS) or `mschap` (LDAP `NT-Password` attribute) |
| `ntPasswordAttribute` | LDAP attribute for `mschap` (default `ntPassword`) |
| `accessGroupDN` | Wraps user filter with `(memberOf=…)` |

See [config/samples/freeradius_ldap_secret.yaml](config/samples/freeradius_ldap_secret.yaml), [config/samples/freeradius_v1_cluster_authentik.yaml](config/samples/freeradius_v1_cluster_authentik.yaml), and [docs/authentik.md](docs/authentik.md) (one-pager) for Authentik LDAP outpost setup.

### Workload defaults

- Image default: `freeradius/freeradius-server:3.2.10-alpine` (override with `spec.image`)
- Generated `radiusd.conf` uses `log { destination = stdout }` so `kubectl logs` shows auth output (not file-only logging)
- `spec.debug: true` enables `radiusd -X` (off by default; very verbose)
- EAP/TLS: modern minimum versions via `spec.eap`

## Client, User, Device

- **Client:** RADIUS NAS (`ipAddress` + `secret` or `fromSecretRef`)
- **User:** PAP users for inner tunnel (`username`/`password` or `fromSecretRef`)
- **Device:** MAC addresses → VLAN for WPA enterprise (`macAddresses`, `vlan`)

### Multi-cluster

Assign resources with `metadata.labels.clusterName: <cluster.metadata.name>`. Unlabeled resources go to the cluster marked `spec.isDefaultCluster`, or to the sole cluster in a namespace.

## API group

`freeradius.valdron.ca/v1` — compatible with [homelab-operator](https://github.com/jValdron/homelab-operator) Device sync.

## Development

Uses your current **kubeconfig context** (`make install` for CRDs first). `make run` defaults to `--leader-election-namespace=freeradius-operator` and `--metrics-secure=false`. Scale down or pause the in-cluster manager before running locally so two reconcilers do not fight.

```bash
make build
make test
make run
# optional: make run RUN_ARGS="--leader-elect=false"
# optional: WATCH_NAMESPACE=freeradius-operator make run
```

Integration controller tests: `go test -tags=integration ./internal/controller/...` (requires envtest assets).

E2E (Kind): `make test-e2e` — includes LDAP against in-cluster OpenLDAP.

## License

Apache 2.0 — see [LICENSE](LICENSE).
