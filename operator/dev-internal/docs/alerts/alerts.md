# Alerts

This page contains the steps to create alerts for the Observability for Kubernetes Operator.

## Table of Content

- [Alert Templates](#alert-templates)
- [Creating Alerts](#creating-alerts)
- [Example: Creating All the Alerts](#example-creating-all-the-alerts)
- [Example: Creating a Single Alert](#example-creating-a-single-alert)
- [Customizing Alerts](#customizing-alerts)

## Alert Templates

We have alert templates on common Kubernetes issues.

| Alert | Template |
|---|---|
| [Detect pod stuck in pending](templates/pod-stuck-in-pending.json.tmpl) | `pod-stuck-in-pending.json.tmpl` |
| [Detect pod stuck in terminating](templates/pod-stuck-in-terminating.json.tmpl) | `pod-stuck-in-terminating.json.tmpl` |
| [Detect pod backoff event](templates/pod-backoff-event.json.tmpl) | `pod-backoff-event.json.tmpl` |
| [Detect workload with non-ready pods](templates/workload-not-ready.json.tmpl) | `workload-not-ready.json.tmpl` |
| [Detect pod out-of-memory kills](templates/pod-out-of-memory-kills.json.tmpl) | `pod-out-of-memory-kills.json.tmpl` |
| [Detect container cpu throttling](templates/container-cpu-throttling.json.tmpl) | `container-cpu-throttling.json.tmpl` |
| [Detect container cpu overutilization](templates/container-cpu-overutilization.json.tmpl) | `container-cpu-overutilization.json.tmpl` |
| [Detect persistent volumes with no claims](templates/persistent-volumes-no-claim.json.tmpl) | `persistent-volumes-no-claim.json.tmpl` |
| [Detect persistent volumes with error](templates/persistent-volumes-error.json.tmpl) | `persistent-volumes-error.json.tmpl` |
| [Detect persistent volumes filling up](templates/persistent-volume-claim-overutilization.json.tmpl) | `persistent-volume-claim-overutilization.json.tmpl` |
| [Detect node memory overutilization](templates/node-memory-overutilization.json.tmpl) | `node-memory-overutilization.json.tmpl` |
| [Detect node cpu overutilization](templates/node-cpu-overutilization.json.tmpl) | `node-cpu-overutilization.json.tmpl` |
| [Detect node filesystem overutilization](templates/node-filesystem-overutilization.json.tmpl) | `node-filesystem-overutilization.json.tmpl` |
| [Detect node cpu-request saturation](templates/node-cpu-request-saturation.json.tmpl) | `node-cpu-request-saturation.json.tmpl` |
| [Detect node memory-request saturation](templates/node-memory-request-saturation.json.tmpl) | `node-memory-request-saturation.json.tmpl` |
| [Detect node disk pressure condition](templates/node-disk-pressure.json.tmpl) | `node-disk-pressure.json.tmpl` |
| [Detect node memory pressure condition](templates/node-memory-pressure.json.tmpl) | `node-memory-pressure.json.tmpl` |

## Creating Alerts

1. Ensure that you have the information for the required fields:
    - **Wavefront API token**. See [Managing API Tokens](https://docs.wavefront.com/wavefront_api.html#managing-api-tokens) page.
    - **Wavefront instance**. For example, the value of `<YOUR_WAVEFRONT_INSTANCE>` from your wavefront url (`https://<YOUR_WAVEFRONT_INSTANCE>.wavefront.com`).
    - **Cluster name**. For example, the value of `clusterName` from your Wavefront Custom Resource configuration (ex: `mycluster-us-west-1`).
    - **(Optional) Alert template**. For example, the value of `<alert_template_file.json.tmpl>` from the list of alert templates (ex: `pod-backoff-event.json.tmpl`).

### Example: Creating All the Alerts

```bash
curl -sSL https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/create-all-alerts.sh | bash -s -- \
  -t <YOUR_API_TOKEN> \
  -c <YOUR_WAVEFRONT_INSTANCE> \
  -n <YOUR_CLUSTER_NAME>
```

>**Note:** You will need to change <YOUR_API_TOKEN>, <YOUR_WAVEFRONT_INSTANCE>, and <YOUR_CLUSTER_NAME> in the above example.

### Example: Creating a Single Alert

```bash
curl -sSL https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/create-alert.sh | bash -s -- \
  -t <YOUR_API_TOKEN> \
  -c <YOUR_WAVEFRONT_INSTANCE> \
  -n <YOUR_CLUSTER_NAME> \
  -f <ALERT_TEMPLATE>
```

>**Note:** You will need to change <YOUR_API_TOKEN>, <YOUR_WAVEFRONT_INSTANCE>, <YOUR_CLUSTER_NAME>, and <ALERT_TEMPLATE> in the above example.

## Customizing Alerts

1. Log in to your service instance `https://<YOUR_WAVEFRONT_INSTANCE>.wavefront.com` as a user with the Alerts permission. Click **Alerting** > **All Alerts** from the toolbar to display the Alerts Browser.
2. Click the alert name, or click the ellipsis icon next to the alert and select **Edit**.  You can search for the alert by typing the alert name in the search field.
3. Change the alert properties when you edit the alert.
4. Specify alert recipients to receive notifications when the alert changes state.
5. Click **Save** in the top right to save your changes.

>**Note:** See [Create and Manage Alerts](https://docs.wavefront.com/alerts_manage.html) for an overview on how to create and manage alerts.
