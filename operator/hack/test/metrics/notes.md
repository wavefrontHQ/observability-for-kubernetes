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


**Operator / IAAS specific metric tests**
TODO OVERLAY Core dns  - overlay on everything except for GKE
TODO OVERLAY Kube dns - overlay for GKE specifically
E2E control plane etcd - currently tested with e2e - should we have an overlay?

COVERED Kubernetes - same
COVERED Control plane api server - same
COVERED autodiscovery - works (prom example)
COVERED runtime plugin config - basic (memcached)

**Collector / non specific** on kind VMs
Including k8s versions tests
Cadvisor
Telegraf
prometheus
systemd
autodiscovery - all combinations
prom / telegraf plugin config - all combinations
runtime config - all combinations 

**Collector cont'd by jcornish**
- kstate?


TODO how do we get test-proxy not to install and use actual proxy?

### Timing Issue (shows up in Nimba)


### Actually Missing (missing in test and nimba)


## jcornish notes

Prefer wider testing, i.e. with tags in collector, just default operator config in operator

Collector integration test = what all is possible?
Operator integration test = what did we choose specifically to configure by default?

I SHALL TALK ABOUT THIS AT ARCH RETR

