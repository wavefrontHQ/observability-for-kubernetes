# TODO
- add operator overlay metrics test for core dns + kube dns
- add anything that's different on different IaaSs to operator test
- run collector tests on kind VM; make it fast and catch variations.
  Not e2e, just test proxy checking "integration test" internal to cluster
- add combined test to CI
- run collector variation tests only on kind VMs
- start with anything that's not covered by e2e,
  simple metrics overlay test or log,
  then highest feedback loop
- proxy/ preprocessing rules will require e2e
- goal: really want to know if we break
  `/operator/deploy/internal/...`
- add versioning to kind metrics tests for collector to get
  fine-grained picture of metrics per k8s version

**Operator / IAAS specific metric tests** = what did we choose specifically to configure by default?
TODO OVERLAY Core dns  - overlay on everything except for GKE
TODO OVERLAY Kube dns - overlay for GKE specifically
E2E control plane etcd - currently tested with e2e - should we have an overlay?

TODO Kubernetes - same
TODO Control plane api server - same
TODO autodiscovery - works (prom example)
TODO runtime plugin config - basic (memcached)

**Collector / non specific** on kind VMs = what all is possible?
TODO Including k8s versions tests
TODO Cadvisor
TODO Telegraf
TODO prometheus
TODO systemd
TODO autodiscovery - all combinations
TODO prom / telegraf plugin config - all combinations
TODO runtime config - all combinations 
