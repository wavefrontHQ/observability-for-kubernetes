exit_on_fail \
  wait_for_query_match_exact \
  "ts(kubernetes.collector.version%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22%20AND%20installation_method%3D%22operator%22%20AND%20processed%3D%22true%22)" \
  "${COLLECTOR_VERSION_IN_DECIMAL}"