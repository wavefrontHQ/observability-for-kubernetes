## Installation
1. Go to Tanzu hub and configure a new self-managed cluster
2. Copy script and copy over values to hub.yaml
3. `VSS_COLLECTOR_CLIENT_SECRET=<VSS_SECRET> K8S_EVENTS_ENDPOINT_TOKEN=<TOKEN> K8S_EVENTS_ENDPOINT_URL=https://data.be.symphony-dev.com/le-mans/v1/streams/ccc-insights-k8s-observations-stream make -C operator deploy-hub`
4. `kubectl apply -f hub.yaml`
5. Go to Tanzu Hub and see data!
