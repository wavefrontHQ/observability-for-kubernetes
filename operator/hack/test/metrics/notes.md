**Operator / IAAS specific metric tests**

[//]: # (TODO add unwanted metric check for the additional tags)
[//]: # (TODO re-enable prom-example but with light metrics: 1 instead of 10)
[//]: # (TODO these are the remaining true missing:)
[//]: # ({"Name":"kubernetes.collector.target.points.collected","Value":"","Timestamp":"","Tags":{"!service":"","!url":"","cluster":"","discovered":"","leading":"","source":"","type":"prometheus"},"Buckets":null})
[//]: # ({"Name":"kubernetes.controlplane.etcd.db.total.size.in.bytes.gauge","Value":"","Timestamp":"","Tags":{"cluster":"","operation":"","source":"","type":""},"Buckets":null})
[//]: # ({"Name":"kubernetes.pod.cpu.limit","Value":"","Timestamp":"","Tags":{"cluster":"","label.name":"jobs","namespace_name":"collector-targets","nodename":"","pod_name":"","source":"","type":"pod"},"Buckets":null})
[//]: # ({"Name":"kubernetes.pod.memory.limit","Value":"","Timestamp":"","Tags":{"cluster":"","label.name":"jobs","namespace_name":"collector-targets","nodename":"","pod_name":"","source":"","type":"pod"},"Buckets":null})

Kubernetes - same
Control plane api server - same
control plane etcd - overlay
Core dns  - overlay
Kube dns - overlay
autodiscovery - works (prom example)
runtime plugin config - basic (memcached)

**Collector / non specific** on kind VMs
Cadvisor
Telegraf
prometheus
systemd
autodiscovery - all combinations
prom / telegraf plugin config - all combinations
runtime config - all combinations 

**Collector cont'd by jcornish**
- kstate?


## Missing Metric Questions for Mark

Why is this particular one excluded?
```
{"Name":"kubernetes.pod.cpu.limit","Tags":{"cluster":"","label.app.kubernetes.io/instance":"memcached-release","label.app.kubernetes.io/managed-by":"Helm","label.app.kubernetes.io/name":"memcached","label.helm.sh/chart":"","namespace_name": "collector-targets","nodename":"","pod_name":"","source":"","type":"pod"}}
{"Name":"kubernetes.pod.cpu.limit", "Value":"200", "Tags":{"cluster":"","label.k8s-app":"prom-example","label.name":"prom-example","namespace_name": "collector-targets","nodename":"","pod_name":"","source":"","type":"pod"}}
{"Name":"kubernetes.pod.cpu.limit", "Tags":{"cluster":"","label.name":"jobs","namespace_name": "collector-targets","nodename":"","pod_name":"","source":"","type":"pod"}}
~{"Name":"kubernetes.pod.cpu.limit","Tags":{"cluster":"","label.app.kubernetes.io/name":"wavefront","namespace_name":"observability-system","nodename":"","pod_name":"","source":"","type":"pod"}}
{"Name":"kubernetes.pod.cpu.limit","Tags":{"cluster":"","namespace_name":"","nodename":"","pod_name":"","source":"","type":"pod","workload_name":"","workload_kind":""}}
```
also these
```
kubernetes.pod.cpu.request
kubernetes.pod.memory.request
```

TODO how do we get test-proxy not to install and use actual proxy?

### Timing Issue (shows up in Nimba)


### Actually Missing (missing in test and nimba)


## jcornish notes

Prefer wider testing, i.e. with tags in collector, just default operator config in operator

Collector integration test = what all is possible?
Operator integration test = what did we choose specifically to configure by default?

I SHALL TALK ABOUT THIS AT ARCH RETR

