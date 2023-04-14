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

package processors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"

	kube_api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const otherResource = "example.com/resource1"

type enricherTestContext struct {
	pod                *kube_api.Pod
	batch              *metrics.Batch
	collectionInterval time.Duration
}

func TestPodEnricherHandlesContainerBatches(t *testing.T) {
	tc := setup()
	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(createContainerBatch())
	assert.NoError(t, err)

	// Updates the pod container metric set with information from the kube_api
	containerMs, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.True(t, found)

	expectedContainerLabels := map[string]string{
		"container_base_image": "k8s.gcr.io/pause:2.0",
		"container_name":       "c1",
		"labels":               "",
		"namespace_name":       "ns1",
		"pod_id":               "",
		"pod_name":             "pod1",
		"type":                 "pod_container",
	}

	assert.Equal(t, expectedContainerLabels, containerMs.Labels)

	expectedContainerValues := map[string]metrics.Value{
		"cpu/limit":                 {IntValue: 0},
		"cpu/request":               {IntValue: 100},
		"ephemeral_storage/limit":   {IntValue: 0},
		"ephemeral_storage/request": {IntValue: 1000},
		"memory/limit":              {IntValue: 0},
		"memory/request":            {IntValue: 555},
	}
	assert.Equal(t, expectedContainerValues, containerMs.Values)

	assert.Empty(t, containerMs.LabeledValues)

	// Creates a second pod container metric set after finding the second container from the kube_api
	containerMs, found = batch.Sets[metrics.PodContainerKey("ns1", "pod1", "nginx")]
	assert.True(t, found)

	expectedContainerLabels = map[string]string{
		"container_base_image": "k8s.gcr.io/pause:2.0",
		"container_name":       "nginx",
		"host_id":              "",
		"hostname":             "",
		"labels":               "",
		"namespace_name":       "ns1",
		"nodename":             "",
		"pod_id":               "",
		"pod_name":             "pod1",
		"type":                 "pod_container",
	}

	assert.Equal(t, expectedContainerLabels, containerMs.Labels)

	expectedContainerValues = map[string]metrics.Value{
		"cpu/limit":                     {IntValue: 2222},
		"cpu/request":                   {IntValue: 333},
		"ephemeral_storage/limit":       {IntValue: 5000},
		"ephemeral_storage/request":     {IntValue: 2000},
		"example.com/resource1/request": {IntValue: 2},
		"memory/limit":                  {IntValue: 3333},
		"memory/request":                {IntValue: 1000},
	}
	assert.Equal(t, expectedContainerValues, containerMs.Values)

	assert.Empty(t, containerMs.LabeledValues)

	// Creates a pod metric set placeholder with some information
	podMs, found := batch.Sets[metrics.PodKey("ns1", "pod1")]
	assert.True(t, found)

	expectedPodLabels := map[string]string{
		"host_id":        "",
		"hostname":       "",
		"labels":         "",
		"namespace_name": "ns1",
		"nodename":       "",
		"pod_id":         "",
		"pod_name":       "pod1",
		"type":           "pod",
	}

	assert.Equal(t, expectedPodLabels, podMs.Labels)
	assert.Empty(t, podMs.Values)

	expectedPodLabelValue := metrics.LabeledValue{
		Name: "status/phase",
		Labels: map[string]string{
			"phase": "Running",
		},
		Value: metrics.Value{
			IntValue: 2,
		},
	}
	assert.Len(t, podMs.LabeledValues, 1)
	assert.Equal(t, expectedPodLabelValue, podMs.LabeledValues[0])
}

func TestPodEnricherHandlesPodBatches(t *testing.T) {
	tc := setup()
	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(createPodBatch())
	assert.NoError(t, err)

	// Updates the pod metric set placeholder with some information
	podMs, found := batch.Sets[metrics.PodKey("ns1", "pod1")]
	assert.True(t, found)

	expectedPodLabels := map[string]string{
		"labels":         "",
		"namespace_name": "ns1",
		"pod_id":         "",
		"pod_name":       "pod1",
		"type":           "pod",
	}

	assert.Equal(t, expectedPodLabels, podMs.Labels)
	assert.Empty(t, podMs.Values)

	expectedPodLabelValue := metrics.LabeledValue{
		Name: "status/phase",
		Labels: map[string]string{
			"phase": "Running",
		},
		Value: metrics.Value{
			IntValue: 2,
		},
	}
	assert.Len(t, podMs.LabeledValues, 1)
	assert.Equal(t, expectedPodLabelValue, podMs.LabeledValues[0])

	// Creates a pod container metric set with information from the kube_api
	containerMs, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.True(t, found)

	expectedContainerLabels := map[string]string{
		"container_base_image": "k8s.gcr.io/pause:2.0",
		"container_name":       "c1",
		"host_id":              "",
		"hostname":             "",
		"labels":               "",
		"namespace_name":       "ns1",
		"nodename":             "",
		"pod_id":               "",
		"pod_name":             "pod1",
		"type":                 "pod_container",
	}

	assert.Equal(t, expectedContainerLabels, containerMs.Labels)

	expectedContainerValues := map[string]metrics.Value{
		"cpu/limit":                 {IntValue: 0},
		"cpu/request":               {IntValue: 100},
		"ephemeral_storage/limit":   {IntValue: 0},
		"ephemeral_storage/request": {IntValue: 1000},
		"memory/limit":              {IntValue: 0},
		"memory/request":            {IntValue: 555},
	}
	assert.Equal(t, expectedContainerValues, containerMs.Values)

	assert.Empty(t, containerMs.LabeledValues)

	// Creates a second pod container metric set after finding the second container from the kube_api
	containerMs, found = batch.Sets[metrics.PodContainerKey("ns1", "pod1", "nginx")]
	assert.True(t, found)

	expectedContainerLabels = map[string]string{
		"container_base_image": "k8s.gcr.io/pause:2.0",
		"container_name":       "nginx",
		"host_id":              "",
		"hostname":             "",
		"labels":               "",
		"namespace_name":       "ns1",
		"nodename":             "",
		"pod_id":               "",
		"pod_name":             "pod1",
		"type":                 "pod_container",
	}

	assert.Equal(t, expectedContainerLabels, containerMs.Labels)

	expectedContainerValues = map[string]metrics.Value{
		"cpu/limit":                     {IntValue: 2222},
		"cpu/request":                   {IntValue: 333},
		"ephemeral_storage/limit":       {IntValue: 5000},
		"ephemeral_storage/request":     {IntValue: 2000},
		"example.com/resource1/request": {IntValue: 2},
		"memory/limit":                  {IntValue: 3333},
		"memory/request":                {IntValue: 1000},
	}
	assert.Equal(t, expectedContainerValues, containerMs.Values)

	assert.Empty(t, containerMs.LabeledValues)
}

func TestDropsContainerMetricWhenPodMissing(t *testing.T) {
	tc := setup()
	tc.pod = nil

	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(batch.Sets))
	_, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.False(t, found)
}

func TestDropsPodMetricWhenPodMissing(t *testing.T) {
	tc := setup()
	tc.pod = nil
	//tc.batch = createPodBatch()

	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(batch.Sets))
	_, found := batch.Sets[metrics.PodKey("ns1", "pod1")]
	assert.False(t, found)
}

func TestPodNotRunning(t *testing.T) {
	tc := setup()
	tc.batch = createPodBatch()

	t.Run("test for succeeded pod", func(t *testing.T) {
		tc.pod.Status = kube_api.PodStatus{
			Phase: kube_api.PodSucceeded,
		}
		runAndCheckNoEnrichedMetrics(t, tc)
	})

	t.Run("test for pending pod", func(t *testing.T) {
		tc.pod.Status = kube_api.PodStatus{
			Phase: kube_api.PodPending,
		}
		runAndCheckNoEnrichedMetrics(t, tc)
	})

	t.Run("test for failed pod", func(t *testing.T) {
		tc.pod.Status = kube_api.PodStatus{
			Phase: kube_api.PodFailed,
		}
		runAndCheckNoEnrichedMetrics(t, tc)
	})
}

func runAndCheckNoEnrichedMetrics(t *testing.T, tc *enricherTestContext) {
	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(batch.Sets))
	podMetric, _ := batch.Sets[metrics.PodKey("ns1", "pod1")]
	assert.Nil(t, podMetric.LabeledValues)
}

func TestMultiplePodsWithOneMissing(t *testing.T) {
	tc := setup() // only returns pod1
	tc.batch = createPodBatch("pod1", "MissingPod")
	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(batch.Sets))

	_, found := batch.Sets[metrics.PodKey("ns1", "MissingPod")]
	assert.False(t, found)

	_, found = batch.Sets[metrics.PodKey("ns1", "pod1")]
	assert.True(t, found)
}

func TestStatusRunning(t *testing.T) {
	tc := setup()
	tc.pod.Status = kube_api.PodStatus{
		ContainerStatuses: []kube_api.ContainerStatus{
			{
				Name:  "c1",
				State: createGoodState(time.Now().Add(-5 * time.Second)),
			},
		},
	}

	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	containerMs, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.True(t, found)

	expectedStatus := metrics.LabeledValue{
		Name: "status",
		Labels: map[string]string{
			"status": "running",
		},
		Value: metrics.Value{
			IntValue: 1,
		},
	}
	assert.Equal(t, expectedStatus, containerMs.LabeledValues[0])
}

func TestStatusTerminated(t *testing.T) {
	tc := setup()
	tc.pod.Status = kube_api.PodStatus{
		ContainerStatuses: []kube_api.ContainerStatus{
			{
				Name:  "c1",
				State: createCrashState(time.Now().Add(-10*time.Minute), time.Now().Add(-5*time.Minute)),
			},
		},
	}

	podBasedEnricher := createEnricher(t, tc)

	batch, err := podBasedEnricher.Process(tc.batch)
	assert.NoError(t, err)

	containerMs, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.True(t, found)

	expectedStatus := metrics.LabeledValue{
		Name: "status",
		Labels: map[string]string{
			"status":    "terminated",
			"reason":    "bad juju",
			"exit_code": "137",
		},
		Value: metrics.Value{
			IntValue: 3,
		},
	}
	assert.Equal(t, expectedStatus, containerMs.LabeledValues[0])
}

func TestStatusMissedTermination(t *testing.T) {
	tc := setup()

	now := time.Now()
	firstStart := now.Add(-10 * time.Minute)
	crashTime := now.Add(-30 * time.Second)
	latestStart := now.Add(-5 * time.Second)

	missedCollectionTime := now

	tc.pod.Status = kube_api.PodStatus{
		ContainerStatuses: []kube_api.ContainerStatus{
			{
				Name:                 "c1",
				State:                createGoodState(latestStart),
				LastTerminationState: createCrashState(firstStart, crashTime),
			},
		},
	}

	podBasedEnricher := createEnricher(t, tc)

	tc.batch.Timestamp = missedCollectionTime
	expectedStatus := metrics.LabeledValue{
		Name: "status",
		Labels: map[string]string{
			"status":    "terminated",
			"reason":    "bad juju",
			"exit_code": "137",
		},
		Value: metrics.Value{
			IntValue: 3,
		},
	}
	assert.Equal(t, expectedStatus, processBatch(t, podBasedEnricher, tc.batch))
}

func TestStatusPassedTermination(t *testing.T) {
	tc := setup()

	now := time.Now()
	firstStart := now.Add(-10 * time.Minute)
	crashTime := now.Add(-30 * time.Second)
	latestStart := now.Add(-5 * time.Second)

	followingCollectionTime := now.Add(tc.collectionInterval)

	tc.pod.Status = kube_api.PodStatus{
		ContainerStatuses: []kube_api.ContainerStatus{
			{
				Name:                 "c1",
				State:                createGoodState(latestStart),
				LastTerminationState: createCrashState(firstStart, crashTime),
			},
		},
	}

	podBasedEnricher := createEnricher(t, tc)

	expectedStatus := metrics.LabeledValue{
		Name: "status",
		Labels: map[string]string{
			"status": "running",
		},
		Value: metrics.Value{
			IntValue: 1,
		},
	}
	batch2 := createContainerBatch()
	batch2.Timestamp = followingCollectionTime
	assert.Equal(t, expectedStatus, processBatch(t, podBasedEnricher, batch2))
}

func processBatch(t assert.TestingT, podBasedEnricher *PodBasedEnricher, batch *metrics.Batch) metrics.LabeledValue {
	var err error
	batch, err = podBasedEnricher.Process(batch)
	assert.NoError(t, err)

	containerMs, found := batch.Sets[metrics.PodContainerKey("ns1", "pod1", "c1")]
	assert.True(t, found)
	return containerMs.LabeledValues[0]
}

func createEnricher(t *testing.T, tc *enricherTestContext) *PodBasedEnricher {
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	podLister := v1listers.NewPodLister(store)
	if tc.pod != nil {
		err := store.Add(tc.pod)
		assert.NoError(t, err)
	}

	labelCopier, err := util.NewLabelCopier(",", []string{}, []string{})
	assert.NoError(t, err)

	return NewPodBasedEnricher(podLister, labelCopier, tc.collectionInterval)
}

func setup() *enricherTestContext {
	return &enricherTestContext{
		collectionInterval: time.Minute,
		batch:              createContainerBatch(),
		pod: &kube_api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "ns1",
			},
			Status: kube_api.PodStatus{
				Phase: kube_api.PodRunning,
			},
			Spec: kube_api.PodSpec{
				NodeName: "node1",
				Containers: []kube_api.Container{
					{
						Name:  "c1",
						Image: "k8s.gcr.io/pause:2.0",
						Resources: kube_api.ResourceRequirements{
							Requests: kube_api.ResourceList{
								kube_api.ResourceCPU:              *resource.NewMilliQuantity(100, resource.DecimalSI),
								kube_api.ResourceMemory:           *resource.NewQuantity(555, resource.DecimalSI),
								kube_api.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
							},
						},
					},
					{
						Name:  "nginx",
						Image: "k8s.gcr.io/pause:2.0",
						Resources: kube_api.ResourceRequirements{
							Requests: kube_api.ResourceList{
								kube_api.ResourceCPU:              *resource.NewMilliQuantity(333, resource.DecimalSI),
								kube_api.ResourceMemory:           *resource.NewQuantity(1000, resource.DecimalSI),
								kube_api.ResourceEphemeralStorage: *resource.NewQuantity(2000, resource.DecimalSI),
								otherResource:                     *resource.NewQuantity(2, resource.DecimalSI),
							},
							Limits: kube_api.ResourceList{
								kube_api.ResourceCPU:              *resource.NewMilliQuantity(2222, resource.DecimalSI),
								kube_api.ResourceMemory:           *resource.NewQuantity(3333, resource.DecimalSI),
								kube_api.ResourceEphemeralStorage: *resource.NewQuantity(5000, resource.DecimalSI),
								otherResource:                     *resource.NewQuantity(2, resource.DecimalSI),
							},
						},
					},
				},
			},
		},
	}
}

func createContainerBatch() *metrics.Batch {
	return &metrics.Batch{
		Timestamp: time.Now(),
		Sets: map[metrics.ResourceKey]*metrics.Set{
			metrics.PodContainerKey("ns1", "pod1", "c1"): {
				Labels: map[string]string{
					metrics.LabelMetricSetType.Key: metrics.MetricSetTypePodContainer,
					metrics.LabelPodName.Key:       "pod1",
					metrics.LabelNamespaceName.Key: "ns1",
					metrics.LabelContainerName.Key: "c1",
				},
				Values: map[string]metrics.Value{},
			},
		},
	}
}

func createPodBatch(podNames ...string) *metrics.Batch {
	if len(podNames) == 0 {
		podNames = append(podNames, "pod1")
	}
	dataBatch := metrics.Batch{
		Timestamp: time.Now(),
		Sets:      map[metrics.ResourceKey]*metrics.Set{},
	}
	for _, podName := range podNames {
		dataBatch.Sets[metrics.PodKey("ns1", podName)] = &metrics.Set{
			Labels: map[string]string{
				metrics.LabelMetricSetType.Key: metrics.MetricSetTypePod,
				metrics.LabelPodName.Key:       podName,
				metrics.LabelNamespaceName.Key: "ns1",
			},
			Values: map[string]metrics.Value{},
		}
	}
	return &dataBatch
}

func createCrashState(startTime time.Time, crashTime time.Time) kube_api.ContainerState {
	return kube_api.ContainerState{
		Terminated: &kube_api.ContainerStateTerminated{
			Reason:  "bad juju",
			Message: "broken",
			StartedAt: metav1.Time{
				Time: startTime,
			},
			FinishedAt: metav1.Time{
				Time: crashTime,
			},
			ContainerID: "",
			ExitCode:    137,
		},
	}
}

func createGoodState(timestamp time.Time) kube_api.ContainerState {
	return kube_api.ContainerState{
		Running: &kube_api.ContainerStateRunning{
			StartedAt: metav1.Time{
				Time: timestamp,
			},
		},
	}
}
