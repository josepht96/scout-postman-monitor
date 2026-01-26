# Scout Helm Chart - Quick Start Guide

This guide will help you quickly deploy Scout Postman Monitor on OpenShift or Kubernetes using Helm.

## Prerequisites

- Helm 3.x installed
- Access to an OpenShift or Kubernetes cluster
- `kubectl` or `oc` CLI configured

## Installation Methods

### Method 1: Basic OpenShift Deployment (Auto-generated URL)

The simplest way to deploy on OpenShift with an auto-generated route:

```bash
helm install scout deployments/helm/scout
```

Get your route URL:
```bash
oc get route scout -o jsonpath='{.spec.host}'
# Or visit the full URL:
echo "https://$(oc get route scout -o jsonpath='{.spec.host}')"
```

### Method 2: OpenShift with Custom URL

Deploy with a specific hostname:

```bash
helm install scout deployments/helm/scout \
  --set route.host=scout.apps.openshift.example.com
```

Access your application at: `https://scout.apps.openshift.example.com`

### Method 3: Standard Kubernetes with Ingress

For non-OpenShift Kubernetes clusters:

```bash
helm install scout deployments/helm/scout \
  --set route.enabled=false \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=scout.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

Or use the example file:
```bash
helm install scout deployments/helm/scout -f deployments/helm/scout/examples/kubernetes-ingress.yaml
```

### Method 4: Production Deployment with External Database

For production environments with an external PostgreSQL database:

```bash
helm install scout deployments/helm/scout \
  --set postgresql.enabled=false \
  --set database.external.enabled=true \
  --set database.external.url="postgres://user:pass@db.example.com:5432/scout?sslmode=require" \
  --set route.host=scout.production.example.com \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2
```

Or use the example file:
```bash
# Edit the database URL in the example file first
helm install scout deployments/helm/scout -f deployments/helm/scout/examples/external-database.yaml
```

## Verifying the Deployment

Check if pods are running:
```bash
kubectl get pods -l app.kubernetes.io/name=scout
```

View logs:
```bash
kubectl logs -l app.kubernetes.io/name=scout -f
```

Check the route/ingress:
```bash
# OpenShift
oc get route scout

# Kubernetes
kubectl get ingress scout
```

## Testing the API

Once deployed, test the health endpoint:

```bash
# OpenShift (auto-generated route)
curl https://$(oc get route scout -o jsonpath='{.spec.host}')/health

# Custom URL
curl https://scout.example.com/health
```

Check Prometheus metrics:
```bash
curl https://$(oc get route scout -o jsonpath='{.spec.host}')/metrics
```

## Adding Postman Collections

You can add collections in two ways:

### Option 1: Using Helm values file

Create a `my-values.yaml`:
```yaml
scout:
  collections:
    my-collection.json: |
      {
        "info": {
          "name": "My Collection",
          "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": [
          {
            "name": "Test Request",
            "request": {
              "method": "GET",
              "url": "https://api.example.com/test"
            }
          }
        ]
      }
```

Install with collections:
```bash
helm install scout deployments/helm/scout -f my-values.yaml
```

### Option 2: Using the example file

```bash
helm install scout deployments/helm/scout -f deployments/helm/scout/examples/with-collections.yaml
```

## Upgrading

To update your deployment (e.g., add new collections or change configuration):

```bash
helm upgrade scout deployments/helm/scout -f my-values.yaml
```

## Uninstalling

Remove the deployment:
```bash
helm uninstall scout
```

To also delete the database PVC:
```bash
kubectl delete pvc scout-postgresql-pvc
```

## Common Configurations

### Change Collection Execution Interval

```bash
helm install scout deployments/helm/scout --set scout.interval="120s"
```

### Adjust Resource Limits

```bash
helm install scout deployments/helm/scout \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=1Gi \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=512Mi
```

### Configure PostgreSQL Storage

```bash
helm install scout deployments/helm/scout \
  --set postgresql.persistence.size=10Gi \
  --set postgresql.persistence.storageClass=fast-ssd
```

### Enable Autoscaling

```bash
helm install scout deployments/helm/scout \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10 \
  --set autoscaling.targetCPUUtilizationPercentage=75
```

## Troubleshooting

### Pods not starting

Check pod events:
```bash
kubectl describe pod -l app.kubernetes.io/name=scout
```

### Database connection issues

Verify the database URL:
```bash
kubectl exec deployment/scout -- env | grep DATABASE_URL
```

Test PostgreSQL connectivity:
```bash
kubectl exec deployment/scout-postgresql -- pg_isready -U postgres
```

### Route/Ingress not working

Check the route status:
```bash
# OpenShift
oc describe route scout

# Kubernetes
kubectl describe ingress scout
```

### View all resources

```bash
kubectl get all,configmap,secret,pvc,route -l app.kubernetes.io/instance=scout
```

## Next Steps

- See the full [README](scout/README.md) for detailed configuration options
- Check the [examples](scout/examples/) directory for more deployment scenarios
- Review the [values.yaml](scout/values.yaml) for all available configuration options

## Support

For issues or questions, please open an issue in the GitHub repository.
