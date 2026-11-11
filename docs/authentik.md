# Authentik + freeradius-operator (LDAP outpost)

Use Authentik’s **LDAP provider and outpost** as the user directory for FreeRADIUS **inner-tunnel** auth. This operator generates the `ldap` module and patches `inner-tunnel`; you run FreeRADIUS from the `Cluster` CR and point NAS/Wi‑Fi at it with **EAP-TTLS** (recommended with `bindAsUser`).

Authentik’s built-in **RADIUS outpost** is a different path; this guide is for **LDAP outpost → FreeRADIUS**.

## 1. Authentik

1. Create an **LDAP provider** (Applications → Providers). Note the **Base DN** (default is often `DC=ldap,DC=goauthentik,DC=io`; change only if you customized it).
2. Create an **Application** bound to that provider and include the users/groups who may use RADIUS.
3. Deploy an **LDAP outpost** (embedded or standalone) so the directory is reachable from the cluster where FreeRADIUS runs.
4. Create a **service account** (or dedicated user) for FreeRADIUS **search binds**:
   - Must be allowed to use the LDAP application.
   - For bind-as-user, the module still needs to **find** the user DN before the user password bind; grant **“Search full LDAP directory”** on a role if searches are restricted (see [Authentik LDAP docs](https://docs.goauthentik.io/add-secure-apps/providers/ldap/)).
5. Expose the outpost in Kubernetes (Service DNS the operator will use), e.g. `authentik-ldap-outpost.my-ns.svc.cluster.local:389` (or `ldaps://` if you terminate TLS on the outpost).

Optional: restrict who can get RADIUS by setting `spec.ldap.accessGroupDN` to a group DN under `ou=groups,<baseDN>`.

## 2. Kubernetes

**Bind secret** (namespace must match the `Cluster`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: freeradius-ldap-bind
type: Opaque
stringData:
  identity: uid=freeradius-service,ou=users,DC=ldap,DC=goauthentik,DC=io
  password: <service-account-password>
```

**Cluster** — full example: [config/samples/freeradius_v1_cluster_authentik.yaml](../config/samples/freeradius_v1_cluster_authentik.yaml). Important fields:

| Setting | Typical Authentik value |
|--------|-------------------------|
| `spec.ldap.servers` | `[<ldap-outpost-service>.<ns>.svc.cluster.local]` (add more hosts for HA) |
| `spec.ldap.baseDN` | Provider Base DN from step 1 |
| `spec.ldap.userBaseDN` | `ou=users,<baseDN>` |
| `spec.ldap.userFilter` | `(&(objectClass=goauthentik.io/ldap/user)(uid=%{%{Stripped-User-Name}:-%{User-Name}}))` |
| `spec.ldap.authMode` | `bindAsUser` (default) — Authentik does not expose `userPassword` for hash compare |
| `spec.eap.defaultEapType` | `ttls` (inner PAP over TLS) |

Apply cert-manager/TLS for EAP as in the other samples, then `Client` CRs for your NAS.

## 3. Auth modes

**bindAsUser (default)** — Matches Authentik without extra policies. Clients use **EAP-TTLS**; inner tunnel uses PAP; FreeRADIUS LDAP-binds as the user. See [Authentik discussion #8825](https://github.com/goauthentik/authentik/discussions/8825).

**mschap** — Requires an NT hash attribute on users (e.g. `ipaNTHash`) via Authentik expression policies on password change; set `authMode: mschap`, `ntPasswordAttribute: ipaNTHash`, and usually `spec.eap.defaultEapType: peap`. Only choose this if you intentionally need PEAP-MSCHAPv2.

## 4. Verify

- `kubectl wait cluster/<name> --for=condition=Ready`
- Pod logs: LDAP lookups and inner-tunnel accepts (enable `spec.debug: true` only when troubleshooting).
- From inside the FreeRADIUS pod, stock inner-tunnel listens on `127.0.0.1:18120` with client `localhost` / secret `testing123` (added when LDAP is enabled).

## 5. Troubleshooting

| Symptom | Check |
|--------|--------|
| User not found | `baseDN`, `userBaseDN`, `userFilter`, service account search permissions |
| Bind OK in Authentik UI but RADIUS fails | Outpost Service DNS/port from the FreeRADIUS pod (`ldapsearch` or `nc`) |
| `unknown client 127.0.0.1` on inner-tunnel | Regenerate config with LDAP enabled (localhost client is included) |
| MS-CHAP / `No NT-Password` | Use `bindAsUser` + TTLS, or configure `ipaNTHash` and `authMode: mschap` |

Break-glass **User** CRs still work: they are evaluated via `files` before LDAP in inner-tunnel.
