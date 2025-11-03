# Mailman Helm Chart

A Helm chart for deploying the Mailman email management and OAuth2 authentication system on Kubernetes.

## Introduction

This chart bootstraps a [Mailman](https://github.com/your-org/mailman) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

Mailman provides:
- Email account management and synchronization
- OAuth2 authentication for Gmail and Outlook
- Email labeling and organization features
- AI-powered email processing with OpenAI integration
- Web-based email management interface

## Prerequisites

- Kubernetes 1.23+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure

## Installing the Chart

To install the chart with the release name `mailman`:

```bash
# Clone the repository
git clone https://github.com/seongminhwan/mailman.git
cd mailman/helm/mailman

# Using the deployment script (recommended)
./deploy.sh -e development -t standard

# Or using Helm directly
helm install mailman . -f values-development.yaml

# All-in-One deployment (simplified)
./deploy.sh -e development -t allinone

# Production deployment
./deploy.sh -e production -t standard -n mailman-prod -c
```

### Docker Images

The chart uses the following Docker images from GitHub Container Registry:

- **Backend**: `ghcr.io/seongminhwan/mailman-backend:latest`
- **Frontend**: `ghcr.io/seongminhwan/mailman-frontend:latest`
- **All-in-One**: `ghcr.io/seongminhwan/mailman-all:latest` (contains both frontend and backend)

## Uninstalling the Chart

To uninstall/delete the `mailman` deployment:

```bash
helm uninstall mailman
```

## Configuration

The following table lists the configurable parameters of the mailman chart and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Global image registry | `""` |
| `global.imagePullSecrets` | Global image pull secrets | `[]` |
| `global.storageClass` | Global storage class | `""` |

### Mailman Backend Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mailman.backend.replicaCount` | Number of backend replicas | `1` |
| `mailman.backend.image.repository` | Backend image repository | `mailman-backend` |
| `mailman.backend.image.tag` | Backend image tag | `latest` |
| `mailman.backend.image.pullPolicy` | Backend image pull policy | `IfNotPresent` |
| `mailman.backend.resources` | Backend resource requests/limits | `{}` |
| `mailman.backend.autoscaling.enabled` | Enable backend autoscaling | `false` |
| `mailman.backend.env` | Backend environment variables | `{}` |

### Mailman Frontend Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mailman.frontend.replicaCount` | Number of frontend replicas | `1` |
| `mailman.frontend.image.repository` | Frontend image repository | `mailman-frontend` |
| `mailman.frontend.image.tag` | Frontend image tag | `latest` |
| `mailman.frontend.image.pullPolicy` | Frontend image pull policy | `IfNotPresent` |
| `mailman.frontend.resources` | Frontend resource requests/limits | `{}` |
| `mailman.frontend.autoscaling.enabled` | Enable frontend autoscaling | `false` |

### MySQL Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mysql.enabled` | Enable MySQL dependency | `true` |
| `mysql.auth.rootPassword` | MySQL root password | `""` |
| `mysql.auth.database` | MySQL database name | `mailman` |
| `mysql.auth.username` | MySQL username | `mailman` |
| `mysql.auth.password` | MySQL password | `""` |
| `mysql.primary.persistence.size` | MySQL persistence size | `20Gi` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Ingress host configuration | `[]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Secret Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `secrets.database.create` | Create database secret | `true` |
| `secrets.database.name` | Database secret name | `mailman-database` |
| `secrets.oauth2.create` | Create OAuth2 secret | `true` |
| `secrets.oauth2.name` | OAuth2 secret name | `mailman-oauth2` |
| `secrets.app.create` | Create app secret | `true` |
| `secrets.app.name` | App secret name | `mailman-app` |

### Additional Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.data.enabled` | Enable data persistence | `true` |
| `persistence.data.size` | Data persistence size | `10Gi` |
| `networkPolicy.enabled` | Enable network policies | `false` |
| `podDisruptionBudget.enabled` | Enable PDB | `false` |
| `monitoring.prometheus.enabled` | Enable Prometheus monitoring | `false` |
| `backup.enabled` | Enable backup | `false` |

## Deployment Examples

### Standard Development Deployment

```bash
./deploy.sh -e development -t standard -n mailman-dev -c
```

### All-in-One Development Deployment (Simplified)

```bash
./deploy.sh -e development -t allinone -n mailman-allinone -c
```

### Production Standard Deployment

```bash
./deploy.sh -e production -t standard -n mailman-prod -c \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=mailman.example.com \
  --set secrets.oauth2.gmailClientId=your-gmail-client-id \
  --set secrets.oauth2.gmailClientSecret=your-gmail-client-secret
```

### Production All-in-One Deployment

```bash
./deploy.sh -e production -t allinone -n mailman-prod-allinone -c \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=mailman.example.com \
  --set secrets.oauth2.gmailClientId=your-gmail-client-id \
  --set secrets.oauth2.gmailClientSecret=your-gmail-client-secret
```

### Custom Configuration

Create a custom values file `custom-values.yaml`:

```yaml
mailman:
  backend:
    replicaCount: 2
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: 1000m
        memory: 2Gi

ingress:
  enabled: true
  hosts:
    - host: mailman.company.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mailman-tls
      hosts:
        - mailman.company.com

secrets:
  oauth2:
    gmailClientId: "your-gmail-client-id"
    gmailClientSecret: "your-gmail-client-secret"
```

Then deploy with:

```bash
helm install mailman mailman/mailman -f custom-values.yaml
```

## OAuth2 Configuration

To configure OAuth2 providers, you need to:

1. **Gmail OAuth2 Setup**:
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable Gmail API
   - Create OAuth 2.0 credentials
   - Add redirect URI: `https://your-domain.com/api/oauth2/callback/gmail`

2. **Outlook OAuth2 Setup**:
   - Go to [Azure Portal](https://portal.azure.com/)
   - Register a new application
   - Add API permissions for Mail.Read, Mail.Send, offline_access
   - Add redirect URI: `https://your-domain.com/api/oauth2/callback/outlook`

3. **Configure in Helm**:
   ```bash
   helm upgrade mailman mailman/mailman \
     --set secrets.oauth2.gmailClientId=your-gmail-client-id \
     --set secrets.oauth2.gmailClientSecret=your-gmail-client-secret \
     --set secrets.oauth2.outlookClientId=your-outlook-client-id \
     --set secrets.oauth2.outlookClientSecret=your-outlook-client-secret
   ```

## Database Management

### Creating Database Backup

```bash
kubectl exec -it deployment/mailman-backend -- \
  mysqldump -h mysql -u mailman -p mailman > backup.sql
```

### Restoring Database

```bash
kubectl exec -i deployment/mailman-backend -- \
  mysql -h mysql -u mailman -p mailman < backup.sql
```

## Monitoring

### Prometheus Metrics

The application exposes metrics at `/metrics` endpoint when monitoring is enabled:

```bash
helm install mailman mailman/mailman \
  --set monitoring.prometheus.enabled=true \
  --set monitoring.prometheus.serviceMonitor.enabled=true
```

### Grafana Dashboard

Import the provided Grafana dashboard for comprehensive monitoring:

```bash
helm install mailman mailman/mailman \
  --set monitoring.grafana.enabled=true
```

## Troubleshooting

### Common Issues

1. **Pods not starting**:
   ```bash
   kubectl get pods -l app.kubernetes.io/name=mailman
   kubectl describe pod <pod-name>
   ```

2. **Database connection issues**:
   ```bash
   kubectl logs deployment/mailman-backend
   kubectl exec -it deployment/mailman-backend -- nslookup mysql
   ```

3. **OAuth2 redirect errors**:
   - Verify redirect URIs in OAuth2 provider console
   - Check ingress configuration
   - Ensure TLS certificates are valid

### Logs

```bash
# Backend logs
kubectl logs -f deployment/mailman-backend

# Frontend logs
kubectl logs -f deployment/mailman-frontend

# MySQL logs
kubectl logs -f deployment/mysql
```

### Port Forwarding for Testing

```bash
# Forward backend API
kubectl port-forward deployment/mailman-backend 8080:8080

# Forward frontend
kubectl port-forward deployment/mailman-frontend 3000:80
```

## Upgrading

To upgrade the deployment:

```bash
helm upgrade mailman mailman/mailman
```

To upgrade with custom values:

```bash
helm upgrade mailman mailman/mailman -f custom-values.yaml
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This chart is licensed under the MIT License.

## Support

For support and questions:
- Create an issue in the GitHub repository
- Check the [documentation](https://github.com/your-org/mailman/docs)
- Join our [Slack community](https://slack.example.com)