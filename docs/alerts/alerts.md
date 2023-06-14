# Alerts
This page contains the steps to create an alert template.

We have alert templates on common Kubernetes issues.

* [Detect pods stuck in pending](templates/pods-stuck-in-pending.json.tmpl)

## Flags

```
Usage of ./create-alert.sh:
    -t  (Required) Wavefront API token
    -c  (Required) Wavefront instance name
    -f  (Required) path to alert file template
    -n  (Required) kubernetes cluster name
    -h  print usage info and exit
```

## Create an alert

### Step 1: Download the alert template file.

1. Replace `<alert_file_output_path>`, (ex: `/tmp/pods-stuck-in-pending.json`).
2. Replace `<alert_template_file.json.tmpl>`, (ex: `pods-stuck-in-pending.json.tmpl`).

```bash
export ALERT_FILE_OUTPUT_PATH=<alert_file_output_path>
export ALERT_TEMPLATE_FILE=<alert_template_file.json.tmpl>
curl -sSL -o "$ALERT_FILE_OUTPUT_PATH" "https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/templates/$ALERT_TEMPLATE_FILE"
```

### Step 2: Create the alert template.

1. Ensure that you have the information for the required fields:
   - **Wavefront API token**. See [Managing API Tokens](https://docs.wavefront.com/wavefront_api.html#managing-api-tokens) page.
   - **Wavefront instance**. For example, the value of `<your_instance>` from your wavefront url (`https://<your_instance>.wavefront.com`).
   - **Cluster name**. For example, a partial regex value (ex: `"prod*"`), or the value of `clusterName` from your Wavefront Custom Resource configuration (ex: [wavefront.yaml](../../deploy/scenarios/wavefront-getting-started.yaml)).
   - **Alert template file**. For example, the download output path of the alert template file from **Step 1**.

```bash
curl -sSL https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/docs/alerts/create-alert.sh | bash -s -- \
  -t <YOUR_API_TOKEN> \
  -c <YOUR_WAVEFRONT_INSTANCE> \
  -n <YOUR_CLUSTER_NAME> \
  -f <PATH_TO_ALERT_FILE>
```

**Note:** You will need to change YOUR_API_TOKEN, YOUR_WAVEFRONT_INSTANCE, YOUR_CLUSTER_NAME, and PATH_TO_ALERT_FILE in the above example.

### Step 3: Customize the alert.

1. Log in to your service instance `https://<your_instance>.wavefront.com` as a user with the Alerts permission. Click **Alerting** > **All Alerts** from the toolbar to display the Alerts Browser.
2. Click the alert name, or click the ellipsis icon next to the alert and select **Edit**.  You can search for the alert by by typing the alert name in the search field.
3. Change the alert properties when you edit the alert.
4. Specify alert recipients to receive notifications when the alert changes state.
5. Click **Save** in the top right to save your changes.

See [Create and Manage Alerts](https://docs.wavefront.com/alerts_manage.html) for an overview on how to create and manage alerts.
