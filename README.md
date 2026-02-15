# SFTPGO Kubernetes Operator

A Kubernetes operator for [SFTPGO](https://github.com/drakkan/sftpgo) built with [Operator SDK](https://sdk.operatorframework.io/).

## Features

- **SftpGoServer CRD**: Deploy and configure SFTPGO server instances
- **SftpGoUser CRD**: Manage SFTPGO users declaratively (create, update, enable/disable)
- Full reconciliation loop with status updates
- Support for SFTP, Web Admin, and REST API
- Configurable storage (SQLite, MySQL, PostgreSQL)

## Quick Start

### Prerequisites

- Kubernetes cluster
- Operator SDK CLI
- kubectl

### Install CRDs and Operator

```bash
# Install CRDs
make install

# Deploy the operator (from config/default)
make deploy

# Or run locally (for development)
make run
```

### Create an SFTPGO Server

```yaml
apiVersion: sftpgo.sftpgo.io/v1alpha1
kind: SftpGoServer
metadata:
  name: my-sftpgo
spec:
  replicas: 1
  sftpPort: 2022
  webPort: 8080
  storageBackend: sqlite
  dataVolume:
    size: 10Gi
  adminSecretRef:  # Required for SftpGoUser management
    name: sftpgo-admin-secret
```

Create the admin secret (SFTPGO creates admin on first run - check logs for credentials, or set in config):

```bash
kubectl create secret generic sftpgo-admin-secret \
  --from-literal=username=admin \
  --from-literal=password=your-admin-password
```

### Manage Users

```yaml
apiVersion: sftpgo.sftpgo.io/v1alpha1
kind: SftpGoUser
metadata:
  name: alice
spec:
  username: alice
  status: enabled        # or "disabled" to disable user
  homeDir: /srv/sftpgo/data/alice
  serverRef:
    name: my-sftpgo
  passwordSecretRef:
    name: alice-password
    key: password
  permissions:
    - "*"
  quota:
    size: 1073741824     # 1GB
    files: 10000
  bandwidthLimits:
    upload: 1048576      # 1MB/s
    download: 10485760   # 10MB/s
  maxSessions: 5
```

### Enable/Disable Users

Set `spec.status` to `enabled` or `disabled`:

```bash
kubectl patch sftpgouser alice --type=merge -p '{"spec":{"status":"disabled"}}'
```

## CRD Reference

### SftpGoServer

| Field | Type | Description |
|-------|------|-------------|
| spec.image | string | Container image (default: docker.io/drakkan/sftpgo:latest) |
| spec.replicas | int32 | Number of replicas |
| spec.sftpPort | int32 | SFTP port (default: 2022) |
| spec.webPort | int32 | Web/API port (default: 8080) |
| spec.storageBackend | string | memory, sqlite, mysql, postgres |
| spec.dataVolume | object | PVC configuration |
| spec.database | object | Database config for mysql/postgres |
| spec.adminSecretRef | object | Secret with username/password for API |
| spec.resources | object | Container resource limits |
| spec.nodeSelector | map | Pod node selector |
| spec.tolerations | [] | Pod tolerations |
| spec.affinity | object | Pod affinity |

### SftpGoUser

| Field | Type | Description |
|-------|------|-------------|
| spec.username | string | SFTPGO username (required) |
| spec.status | string | `enabled` or `disabled` |
| spec.homeDir | string | Home directory (required) |
| spec.password | string | Plain password (avoid in production) |
| spec.passwordSecretRef | object | Secret reference for password |
| spec.publicKeys | []string | SSH public keys |
| spec.publicKeysSecretRef | object | Secret with public keys |
| spec.email | string | User email |
| spec.permissions | []string | Permissions (e.g. `["*"]` for all) |
| spec.quota | object | Storage quota (size, files) |
| spec.bandwidthLimits | object | Upload/download limits |
| spec.maxSessions | int | Max concurrent sessions |
| spec.allowedIP | []string | Allowed IPs (CIDR) |
| spec.deniedIP | []string | Denied IPs (CIDR) |
| spec.virtualFolders | [] | Virtual folder mappings |
| spec.serverRef | object | Reference to SftpGoServer (required) |

## Development

```bash
# Run tests
make test

# Generate code
make generate manifests

# Build
make build

# Run locally
make run
```

## License

Apache-2.0
