exit_on_fail \
  wait_for_query_match_exact \
  "ts(kubernetes.controlplane.etcd.server.has.leader.gauge%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)" \
  "1"

exit_on_fail \
  wait_for_query_non_zero \
  "ts(kubernetes.controlplane.etcd.network.client.grpc.received.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"