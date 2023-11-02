ETCD_HAS_LEADER_TYPE="cluster"
exit_on_fail \
  wait_for_query_match_exact \
  "ts(kubernetes.controlplane.etcd.server.has.leader.gauge%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22%20AND%20type%3D%22${ETCD_HAS_LEADER_TYPE}%22)" \
  "1"

num_nodes=$(kubectl get nodes --no-headers | wc -l | tr -d '[:space:]')
if [ $num_nodes -gt 1 ]; then
  exit_on_fail \
    wait_for_query_non_zero \
    "ts(kubernetes.controlplane.etcd.network.client.grpc.received.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"

  exit_on_fail \
    wait_for_query_non_zero \
    "ts(kubernetes.controlplane.etcd.network.client.grpc.sent.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"
fi

if [ $num_nodes -gt 2 ]; then
  exit_on_fail \
    wait_for_query_non_zero \
    "ts(kubernetes.controlplane.etcd.network.peer.received.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"

  exit_on_fail \
    wait_for_query_non_zero \
    "ts(kubernetes.controlplane.etcd.network.peer.sent.bytes.total.counter%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"

  exit_on_fail \
    wait_for_query_non_zero \
    "hs(kubernetes.controlplane.etcd.network.peer.round.trip.time.seconds.m%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"
fi
