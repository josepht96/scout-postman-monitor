# Scout Helm Chart

A Helm chart for deploying Scout Postman Monitor on Kubernetes and OpenShift. Scout is an automated Postman collection runner with a PostgreSQL backend for storing results and metrics.

## Prerequisites

- Kubernetes 1.19+ or OpenShift 4.x+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure (for PostgreSQL persistence)

## Installing the Chart

### Basic Installation (OpenShift)

```bash
helm install scout deployments/helm/scout
```

This will deploy Scout with:
- Internal PostgreSQL database
- OpenShift Route enabled (with TLS)
- Auto-generated hostname

### Installation with Custom URL (OpenShift)

```bash
helm install scout deployments/helm/scout \
  --set route.host=scout.example.com
```

### Installation on Standard Kubernetes

For standard Kubernetes clusters (non-OpenShift), use Ingress instead of Route:

```bash
helm install scout deployments/helm/scout \
  --set route.enabled=false \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=scout.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

### Installation with External Database

```bash
helm install scout deployments/helm/scout \
  --set postgresql.enabled=false \
  --set database.external.enabled=true \
  --set database.external.url="postgres://user:pass@host:5432/scout?sslmode=require"
```

## Configuration

The following table lists the configurable parameters of the Scout chart and their default values.

### Application Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of Scout replicas | `1` |
| `image.repository` | Scout image repository | `scout` |
| `image.tag` | Scout image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `service.targetPort` | Container port | `8080` |

### OpenShift Route Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `route.enabled` | Enable OpenShift Route | `true` |
| `route.host` | Custom hostname for the route (leave empty for auto-generated) | `""` |
| `route.path` | Route path | `""` |
| `route.tls.enabled` | Enable TLS for the route | `true` |
| `route.tls.termination` | TLS termination type | `edge` |
| `route.tls.insecureEdgeTerminationPolicy` | Policy for insecure traffic | `Redirect` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable Ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Ingress hosts configuration | See values.yaml |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Scout Application Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `scout.interval` | Collection execution interval | `30s` |
| `scout.port` | Application port | `8080` |
| `scout.collectionsDir` | Collections directory path | `/app/collections` |
| `scout.newmanScriptPath` | Newman executor script path | `/app/newman/executor.js` |
| `scout.collections` | Postman collections as configmap data | `{}` |

### Database Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `database.external.enabled` | Use external database | `false` |
| `database.external.url` | Full database connection URL | `""` |
| `database.host` | Database host | `postgres` |
| `database.port` | Database port | `5432` |
| `database.name` | Database name | `scout` |
| `database.username` | Database username | `postgres` |
| `database.password` | Database password | `postgres` |
| `database.sslmode` | Database SSL mode | `disable` |

### PostgreSQL Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `postgresql.image.repository` | PostgreSQL image repository | `postgres` |
| `postgresql.image.tag` | PostgreSQL image tag | `15-alpine` |
| `postgresql.persistence.enabled` | Enable persistence | `true` |
| `postgresql.persistence.size` | PVC size | `5Gi` |
| `postgresql.persistence.storageClass` | Storage class | `""` (default) |
| `postgresql.resources.limits.cpu` | CPU limit | `1000m` |
| `postgresql.resources.limits.memory` | Memory limit | `1Gi` |

### Resource Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Autoscaling Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `1` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |

## Usage Examples

### Example 1: Deploy with Custom Collections

Create a `custom-values.yaml`:

```yaml
scout:
  collections:
    api-health-check.json: |
      {
        "info": {
          "name": "API Health Check",
          "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": [
          {
            "name": "Health Check",
            "request": {
              "method": "GET",
              "url": "https://api.example.com/health"
            }
          }
        ]
      }
```

Install with custom values:

```bash
helm install scout deployments/helm/scout -f custom-values.yaml
```

### Example 2: Production Deployment with External Database

```bash
helm install scout deployments/helm/scout \
  --set postgresql.enabled=false \
  --set database.external.enabled=true \
  --set database.external.url="postgres://scout_user:secure_pass@db.example.com:5432/scout_prod?sslmode=require" \
  --set route.host=scout.production.example.com \
  --set resources.limits.cpu=2000m \
  --set resources.limits.memory=2Gi \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=512Mi \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10
```

### Example 3: Deploy with Specific Storage Class

```bash
helm install scout deployments/helm/scout \
  --set postgresql.persistence.storageClass=fast-ssd \
  --set postgresql.persistence.size=10Gi
```

## Upgrading

### Upgrade the Release

```bash
helm upgrade scout deployments/helm/scout -f custom-values.yaml
```

### Upgrade with New Collections

```bash
helm upgrade scout deployments/helm/scout \
  --set scout.collections."new-collection\.json"='{"collection": "data"}'
```

## Uninstalling

```bash
helm uninstall scout
```

This will delete all resources associated with the chart, including the PersistentVolumeClaim for PostgreSQL.

To keep the PVC (and preserve data):

```bash
# Delete the release but keep PVC
helm uninstall scout --keep-history
kubectl delete deployment,service,route,configmap,secret -l app.kubernetes.io/instance=scout
```

## Accessing Scout

### On OpenShift

After installation, get the Route URL:

```bash
oc get route scout -o jsonpath='{.spec.host}'
```

Or use the full URL:

```bash
echo "https://$(oc get route scout -o jsonpath='{.spec.host}')"
```

### On Standard Kubernetes with Ingress

Access via the configured Ingress hostname:

```bash
curl https://scout.example.com/health
```

### Port-Forward (Development)

For local development/testing:

```bash
kubectl port-forward svc/scout 8080:80
curl http://localhost:8080/health
```

## Monitoring

Scout exposes Prometheus metrics at `/metrics` endpoint. The pods are annotated for Prometheus scraping:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -l app.kubernetes.io/name=scout
```

### View Logs

```bash
kubectl logs -l app.kubernetes.io/name=scout -f
```

### Check Database Connectivity

```bash
kubectl exec -it deployment/scout -- env | grep DATABASE_URL
```

### Verify Route/Ingress

```bash
# OpenShift
oc get route scout

# Kubernetes
kubectl get ingress scout
```

### Debug Configuration

```bash
# View rendered templates
helm template scout deployments/helm/scout

# View actual deployed manifests
helm get manifest scout
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourorg/scout-postman-monitor/issues
- Documentation: https://github.com/yourorg/scout-postman-monitor

## License

See the main repository for license information.
