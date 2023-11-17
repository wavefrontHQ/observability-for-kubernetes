// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
)

type DummySink struct {
	name        string
	mutex       sync.Mutex
	exportCount int
	stopped     bool
	latency     time.Duration
}

func (dummy *DummySink) Name() string {
	return dummy.name
}
func (dummy *DummySink) Export(*metrics.Batch) {
	dummy.mutex.Lock()
	dummy.exportCount++
	dummy.mutex.Unlock()

	time.Sleep(dummy.latency)
}

func (dummy *DummySink) ExportEvent(*events.Event) {
}

func (dummy *DummySink) Stop() {
	dummy.mutex.Lock()
	dummy.stopped = true
	dummy.mutex.Unlock()

	time.Sleep(dummy.latency)
}

func (dummy *DummySink) IsStopped() bool {
	dummy.mutex.Lock()
	defer dummy.mutex.Unlock()
	return dummy.stopped
}

func (dummy *DummySink) GetExportCount() int {
	dummy.mutex.Lock()
	defer dummy.mutex.Unlock()
	return dummy.exportCount
}

func NewDummySink(name string, latency time.Duration) *DummySink {
	return &DummySink{
		name:        name,
		latency:     latency,
		exportCount: 0,
		stopped:     false,
	}
}

type DummyMetricsSource struct {
	latency          time.Duration
	metricSet        metrics.Set
	name             string
	autoDiscovered   bool
	raiseScrapeError bool
}

func (dummy *DummyMetricsSource) AutoDiscovered() bool {
	return dummy.autoDiscovered
}

func (dummy *DummyMetricsSource) Name() string {
	return dummy.name
}

func (src *DummyMetricsSource) Cleanup() {}

func (dummy *DummyMetricsSource) Scrape() (*metrics.Batch, error) {
	time.Sleep(dummy.latency)

	if dummy.raiseScrapeError {
		return nil, errors.New("scrape error")
	}

	point := wf.NewPoint(
		strings.Replace(dummy.Name(), " ", ".", -1),
		1,
		time.Now().UnixNano()/1000,
		dummy.Name(),
		map[string]string{"tag": "tag"},
	)

	res := &metrics.Batch{
		Timestamp: time.Now(),
	}
	res.Metrics = append(res.Metrics, point)
	return res, nil
}

func newDummyMetricSet(name string) metrics.Set {
	return metrics.Set{
		Values: map[string]metrics.Value{},
		Labels: map[string]string{
			"name": name,
		},
	}
}

func NewDummyMetricsSource(name string, latency time.Duration) *DummyMetricsSource {
	return &DummyMetricsSource{
		latency:          latency,
		metricSet:        newDummyMetricSet(name),
		name:             name,
		autoDiscovered:   false,
		raiseScrapeError: false,
	}
}

func NewDummyMetricsSourceWithError(name string, latency time.Duration, autoDiscovered bool) *DummyMetricsSource {
	return &DummyMetricsSource{
		latency:          latency,
		metricSet:        newDummyMetricSet(name),
		name:             name,
		autoDiscovered:   autoDiscovered,
		raiseScrapeError: true,
	}
}

type DummyMetricsSourceProvider struct {
	sources           []metrics.Source
	collectionIterval time.Duration
	timeout           time.Duration
	name              string
}

func (dummy *DummyMetricsSourceProvider) GetMetricsSources() []metrics.Source {
	return dummy.sources
}

func (dummy *DummyMetricsSourceProvider) Name() string {
	return dummy.name
}

func (dummy *DummyMetricsSourceProvider) CollectionInterval() time.Duration {
	return dummy.collectionIterval
}

func (dummy *DummyMetricsSourceProvider) Timeout() time.Duration {
	return dummy.timeout
}

func NewDummyMetricsSourceProvider(name string, collectionIterval, timeout time.Duration, sources ...metrics.Source) metrics.SourceProvider {
	return &DummyMetricsSourceProvider{
		sources:           sources,
		collectionIterval: collectionIterval,
		timeout:           timeout,
		name:              name,
	}
}

type DummyDataProcessor struct {
	latency time.Duration
}

func (dummy *DummyDataProcessor) Name() string {
	return "dummy"
}

func (dummy *DummyDataProcessor) Process(data *metrics.Batch) (*metrics.Batch, error) {
	time.Sleep(dummy.latency)
	return data, nil
}

func NewDummyDataProcessor(latency time.Duration) *DummyDataProcessor {
	return &DummyDataProcessor{
		latency: latency,
	}
}

func NewDummyProviderHandler(count int) *DummyProviderHandler {
	return &DummyProviderHandler{
		count: count,
	}
}

type DummyProviderHandler struct {
	count int
}

func (d *DummyProviderHandler) AddProvider(provider metrics.SourceProvider) {
	d.count += 1
}

func (d *DummyProviderHandler) DeleteProvider(name string) {
	d.count -= 1
}

const podStuckInTerminating = `
Version: v1
kind: Pod
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{},"finalizers":["kubernetes"],"labels":{"exclude-me":"true","name":"pod-stuck-in-terminating"},"name":"pod-stuck-in-terminating","namespace":"collector-targets"},"spec":{"containers":[{"args":["/bin/sh","-c","i=0; while true; do echo \"$i: $(date)\\n\"; echo \"nextline\"; i=$((i+1)); sleep 1; done"],"image":"projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest","name":"pod-stuck-in-terminating","resources":{"limits":{"cpu":"75m","ephemeral-storage":"512Mi","memory":"75Mi"},"requests":{"cpu":"50m","ephemeral-storage":"256Mi","memory":"50Mi"}}}]}}
  creationTimestamp: "2023-11-09T15:16:48Z"
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-11-09T15:16:59Z"
  finalizers:
  - kubernetes
  labels:
    exclude-me: "true"
    name: pod-stuck-in-terminating
  name: pod-stuck-in-terminating
  namespace: collector-targets
  resourceVersion: "1721"
  uid: 6fe211cf-a823-42fe-b574-48eb94323b9a
spec:
  containers:
  - args:
    - /bin/sh
    - -c
    - 'i=0; while true; do echo "$i: $(date)\n"; echo "nextline"; i=$((i+1)); sleep
      1; done'
    image: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest
    imagePullPolicy: Always
    name: pod-stuck-in-terminating
    resources:
      limits:
        cpu: 75m
        ephemeral-storage: 512Mi
        memory: 75Mi
      requests:
        cpu: 50m
        ephemeral-storage: 256Mi
        memory: 50Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: kube-api-access-bmkfv
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: kube-api-access-bmkfv
    projected:
      defaultMode: 420
      sources:
      - serviceAccountToken:
          expirationSeconds: 3607
          path: token
      - configMap:
          items:
          - key: ca.crt
            path: ca.crt
          name: kube-root-ca.crt
      - downwardAPI:
          items:
          - fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
            path: namespace
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:16:48Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:17:31Z"
    message: 'containers with unready status: [pod-stuck-in-terminating]'
    reason: ContainersNotReady
    status: "False"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:17:31Z"
    message: 'containers with unready status: [pod-stuck-in-terminating]'
    reason: ContainersNotReady
    status: "False"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:16:48Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://65ee5fd0328132157ff97107b5fbc8ca2a9f7257b0b3c4ebf322eb956f44ac11
    image: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest
    imageID: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox@sha256:2376a0c12759aa1214ba83e771ff252c7b1663216b192fbe5e0fb364e952f85c
    lastState: {}
    name: pod-stuck-in-terminating
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://65ee5fd0328132157ff97107b5fbc8ca2a9f7257b0b3c4ebf322eb956f44ac11
        exitCode: 137
        finishedAt: "2023-11-09T15:17:30Z"
        reason: Error
        startedAt: "2023-11-09T15:16:58Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.13
  podIPs:
  - ip: 10.244.0.13
  qosClass: Burstable
  startTime: "2023-11-09T15:16:48Z"
`

func GetPodStuckInTerminating() *v1.Pod {
	pod := &v1.Pod{}
	yaml.Unmarshal([]byte(podStuckInTerminating), pod)
	return pod
}
