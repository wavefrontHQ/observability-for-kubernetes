OPERATOR_NAMESPACE="observability-system"
PROXY_POD_NAME="wavefront-proxy*"
PROXY_CONTAINER_NAME="wavefront-proxy"
exit_on_fail wait_for_query_match_exact "at(%22end%22%2C%202m%2C%20ts(kubernetes.collector.version%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22%20AND%20installation_method%3D%22operator%22%20AND%20processed%3D%22true%22))" "${COLLECTOR_VERSION_IN_DECIMAL}"

exit_on_fail \
  wait_for_query_non_zero \
  "at(%22end%22%2C%202m%2C%20ts(kubernetes.cadvisor.container.cpu.cfs.throttled.seconds.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22%20AND%20namespace_name%3D%22${OPERATOR_NAMESPACE}%22%20AND%20pod_name%3D%22${PROXY_POD_NAME}%22%20AND%20container_name%3D%22${PROXY_CONTAINER_NAME}%22))"
