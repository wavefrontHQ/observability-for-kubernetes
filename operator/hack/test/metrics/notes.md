# 8/21/23 fix CI notes from conversation with Mark the other day
Fix CI
- TODO Limit what’s run on TKGm
~~- Fail fast: try to connect and read something from the server and fail on that instead of test timeout
    - Try 3 retries as-is
    - If that doesn't work consistently, then incorporate getting new environment~~

- Set up a separate pipeline for TKGm testing that triggers on collector and operator publish in separate pipeline
    - Get new lease
    - Run light tests
    - Fail fast
    - Try 3 times
- Light test for TKGm: smoke-test
    - Metrics test (common-metrics)
    - Logging test
    - Some e2e test
    - Nothing else
- Last resort: intermittent / release / manual testing if we can’t rely on this for CI per-commit

- If we stop running collector integration tests on TKGm,
  is there any coverage we're losing that we're concerned about?
- What happened to the bin shim on CI for jq?


## Today's hope / ambition:
- Merge
  - TKGM tests + setup for integration test
  - Worker leasing pipeline
- Trigger ^ right after test and publish stage and run in parallel to
  Collector + operator integration tests
- Nesting it all in one pipeline appears to be worse than trying to figure out a whole separate pipeline; reverting



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

DONE Kubernetes - same
DONE Control plane api server - same
DONE autodiscovery - works (prom example)
DONE runtime plugin config - basic (memcached)

**Collector / non specific** on kind VMs = what all is possible?
TODO Including k8s versions tests
TODO Cadvisor
TODO Telegraf
TODO prometheus
TODO systemd
TODO autodiscovery - all combinations
TODO prom / telegraf plugin config - all combinations
TODO runtime config - all combinations 
