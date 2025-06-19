# Pod Restart Slack Operator

A Kubernetes operator that monitors pod restart events and sends notifications to Slack channels.

## Features

- **Real-time Monitoring**: Watches Kubernetes events for pod restarts
- **Slack Integration**: Sends rich notifications to Slack channels via webhooks
- **Flexible Filtering**: Configure namespace and pod selectors to monitor specific resources
- **Custom Templates**: Customize notification messages
- **Multiple Alerts**: Support multiple SlackAlert configurations per cluster
- **RBAC Ready**: Includes proper RBAC permissions for secure deployment

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Slack webhook URL

### Installation

1. **Install CRDs**:
```bash
make install
```

2. **Deploy the operator**:
```bash
make docker-build
make deploy
```

3. **Create a SlackAlert resource**:
```yaml
apiVersion: alerts.vibe-coding.com/v1
kind: SlackAlert
metadata:
  name: production-alerts
  namespace: default
spec:
  webhookUrl: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
  channel: "#alerts"
  username: "K8s Pod Monitor"
  enabled: true
  # Optional: Monitor only production namespaces
  namespaceSelector:
    matchLabels:
      environment: "production"
```

```bash
kubectl apply -f config/samples/slack_alert_sample.yaml
```

## Configuration

### SlackAlert Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `webhookUrl` | string | Yes | Slack webhook URL |
| `channel` | string | No | Target Slack channel (overrides webhook default) |
| `username` | string | No | Bot username for messages |
| `enabled` | boolean | Yes | Enable/disable this alert configuration |
| `messageTemplate` | string | No | Custom message template (uses default if not set) |
| `namespaceSelector` | LabelSelector | No | Select namespaces to monitor |
| `podSelector` | LabelSelector | No | Select pods to monitor |

### Message Template

Use Go-style format strings with these parameters:
1. Pod name
2. Namespace
3. Event reason
4. Event message

Example: `"🚨 Pod %s in namespace %s restarted due to %s: %s"`

### Examples

**Monitor all pods in production namespaces:**
```yaml
apiVersion: alerts.vibe-coding.com/v1
kind: SlackAlert
metadata:
  name: production-monitor
spec:
  webhookUrl: "https://hooks.slack.com/services/..."
  channel: "#production-alerts"
  enabled: true
  namespaceSelector:
    matchLabels:
      environment: "production"
```

**Monitor specific application pods:**
```yaml
apiVersion: alerts.vibe-coding.com/v1
kind: SlackAlert
metadata:
  name: backend-monitor
spec:
  webhookUrl: "https://hooks.slack.com/services/..."
  enabled: true
  podSelector:
    matchLabels:
      app: "backend"
      tier: "api"
```

## Development

### Building

```bash
# Build binary
make build

# Build and run locally
make run

# Build Docker image
make docker-build IMG=your-registry/pod-restart-slack-operator:tag
```

### Testing

```bash
# Run tests
make test

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Deployment

### Local Development

```bash
# Run against your current kubeconfig
make run
```

### Production Deployment

1. **Build and push image**:
```bash
make docker-build IMG=your-registry/pod-restart-slack-operator:v1.0.0
make docker-push IMG=your-registry/pod-restart-slack-operator:v1.0.0
```

2. **Update deployment image**:
```bash
# Edit config/manager/deployment.yaml
# Change image: pod-restart-slack-operator:latest
# to image: your-registry/pod-restart-slack-operator:v1.0.0
```

3. **Deploy**:
```bash
make deploy
```

### Cleanup

```bash
# Remove operator
make undeploy

# Remove CRDs
make uninstall
```

## Monitoring Events

The operator watches for these pod restart events:
- `Killing` - Pod being terminated
- `BackOff` - Container restart backoff
- `FailedMount` - Volume mount failures
- `Failed` - Container failures
- `Unhealthy` - Health check failures
- Any event containing "restart", "restarting", or "killed"

## RBAC Permissions

The operator requires these permissions:
- `slackalerts` (alerts.vibe-coding.com): Full CRUD
- `events`: Read-only access
- `pods`: Read-only access
- `namespaces`: Read-only access

## Troubleshooting

### Operator not receiving events
- Check RBAC permissions
- Verify operator is running: `kubectl get pods -n pod-restart-slack-operator-system`
- Check logs: `kubectl logs -n pod-restart-slack-operator-system deployment/pod-restart-slack-operator`

### Slack messages not sending
- Verify webhook URL is correct
- Check operator logs for HTTP errors
- Test webhook URL manually with curl

### Missing pod restart events
- Check if events match the filtering criteria
- Verify namespace/pod selectors
- Check if SlackAlert is enabled

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0.# augment-code
# claude-code
# claude-code
