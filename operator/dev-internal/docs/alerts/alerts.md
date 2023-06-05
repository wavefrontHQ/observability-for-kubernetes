# Alerts

We have templates for common scenarios.

* [Pods Stuck in Pending phase](templates/pods-stuck-in-pending.json.tmpl)

## How to create an alert

1. Download the alert template file:

```bash
curl -L -o "PATH_TO_ALERT_FILE" \
  https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/templates/<PATH_TO_ALERT_FILE>
```

2. Create the alert by running the following script:

```bash
curl https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/create-alert.sh \
  | bash -s -- \
  -f "PATH_TO_ALERT_FILE" \
  -n "YOUR_K8S_CLUSTER_NAME" \
  -c "YOUR_WAVEFRONT_CLUSTER" \
  -t "YOUR_WAVEFRONT_TOKEN"
```

3. Customize the alert in wavefront.
