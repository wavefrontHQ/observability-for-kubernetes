
OPERATOR_NAMESPACE="observability-system"
PROXY_POD_NAME="wavefront-proxy*"
PROXY_CONTAINER_NAME="wavefront-proxy"
exit_on_fail \
  wait_for_query_match_exact \
  "ts(kubernetes.controlplane.etcd.server.has.leader.gauge%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)" \
  "1"

exit_on_fail \
  wait_for_query_non_zero \
  "ts(kubernetes.controlplane.etcd.network.client.grpc.received.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"

exit_on_fail \
  wait_for_query_non_zero \
  "at(\"end\", 2m, ts(kubernetes.cadvisor.container.cpu.cfs.throttled.seconds.total.counter, cluster=\"${K8S_CLUSTER_NAME}\" and namespace_name=\"${OPERATOR_NAMESPACE}\" and pod_name=\"${PROXY_POD_NAME}\" and container_name=\"${PROXY_CONTAINER_NAME}\"))"
