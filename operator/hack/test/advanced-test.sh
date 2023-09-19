OPERATOR_NAMESPACE="observability-system"
PROXY_POD_NAME="wavefront-proxy*"
PROXY_CONTAINER_NAME="wavefront-proxy"

exit_on_fail wait_for_query_match_exact "at(\"end\", 2m, ts(kubernetes.collector.version, cluster=\"${CONFIG_CLUSTER_NAME}\" and installation_method=\"operator\" and processed=\"true\"))" "${COLLECTOR_VERSION_IN_DECIMAL}"

exit_on_fail \
  wait_for_query_match_exact \
  "ts(kubernetes.controlplane.etcd.server.has.leader.gauge, cluster=\"${CONFIG_CLUSTER_NAME}\")" \
  "1"

exit_on_fail \
  wait_for_query_non_zero \
  "ts(kubernetes.controlplane.etcd.network.client.grpc.received.bytes.total.counter, cluster=\"${CONFIG_CLUSTER_NAME}\")"

exit_on_fail \
  wait_for_query_non_zero \
  "at(\"end\", 2m, ts(kubernetes.cadvisor.container.cpu.cfs.throttled.seconds.total.counter, cluster=\"${K8S_CLUSTER_NAME}\" and namespace_name=\"${OPERATOR_NAMESPACE}\" and pod_name=\"${PROXY_POD_NAME}\" and container_name=\"${PROXY_CONTAINER_NAME}\"))"
