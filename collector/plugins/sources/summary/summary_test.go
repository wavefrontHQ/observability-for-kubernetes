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

package summary

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testhelper"
	util "k8s.io/client-go/util/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/summary/kubelet"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

const (
	// Offsets from seed value in generated container stats.
	offsetCPUUsageCores = iota
	offsetCPUUsageCoreSeconds
	offsetMemPageFaults
	offsetMemMajorPageFaults
	offsetMemUsageBytes
	offsetMemRSSBytes
	offsetMemWorkingSetBytes
	offsetNetRxBytes
	offsetNetRxErrors
	offsetNetTxBytes
	offsetNetTxErrors
	offsetFsUsed
	offsetFsCapacity
	offsetFsAvailable
	offsetAcceleratorMemoryTotal
	offsetAcceleratorMemoryUsed
	offsetAcceleratorDutyCycle
)

const (
	seedNode           = 0
	seedRuntime        = 100
	seedKubelet        = 200
	seedMisc           = 300
	seedPod0           = 1000
	seedPod0Container0 = 2000
	seedPod0Container1 = 2001
	seedPod1           = 3000
	seedPod1Container  = 4000
	seedPod2           = 5000
	seedPod2Container0 = 6000
	seedPod2Container1 = 7000
	seedPod3Container0 = 9000
	seedPod4           = 10000
	seedPod4Container0 = 11000
	seedPod5           = 12000
	seedPod5Container0 = 13000
)

const (
	namespace0 = "test0"
	namespace1 = "test1"

	pName0   = "pod0"
	pName1   = "pod1"
	pName2   = "pod0" // ensure pName2 conflicts with pName0, but is in a different namespace
	pName3   = "pod2"
	pName4   = "pod4" // Regression test for #1838
	pName5   = "pod5"
	pWithPvc = "pvc-pod"

	cName00 = "c0"
	cName01 = "c1"
	cName10 = "c0"      // ensure cName10 conflicts with cName02, but is in a different pod
	cName20 = "c1"      // ensure cName20 conflicts with cName01, but is in a different pod + namespace
	cName21 = "runtime" // ensure that runtime containers are not renamed
	cName30 = "c3"
	cName40 = "c4" // Running, with cpu / memory stats
	cName41 = "c4" // Terminated, has no CPU / Memory stats
	cName42 = "c4" // Terminated, has blank CPU / Memory stats
	cName50 = "c5"

	pvcName = "pvc-claim"
)

var (
	availableFsBytes = uint64(1130)
	usedFsBytes      = uint64(13340)
	totalFsBytes     = uint64(2153)
	freeInode        = uint64(10440)
	usedInode        = uint64(103520)
	totalInode       = uint64(103620)
	scrapeTime       = time.Now()
	startTime        = time.Now().Add(-time.Minute)
)

var nodeInfo = NodeInfo{
	NodeName:       "test",
	HostName:       "test-hostname",
	HostID:         "1234567890",
	KubeletVersion: "1.2",
}

func TestScrapeSummaryMetrics(t *testing.T) {
	summary := stats.Summary{
		Node: stats.NodeStats{
			NodeName:  nodeInfo.NodeName,
			StartTime: metav1.NewTime(startTime),
		},
	}
	data, err := json.Marshal(&summary)
	require.NoError(t, err)

	server := httptest.NewServer(&util.FakeHandler{
		StatusCode:   200,
		ResponseBody: string(data),
		T:            t,
	})
	defer server.Close()

	uri, err := url.Parse(server.URL)
	require.NoError(t, err)

	port, err := strconv.Atoi(uri.Port())
	require.NoError(t, err)

	require.NoError(t, err)
	ms := testingSummaryMetricsSource(uint(port))
	ms.node.IP = net.ParseIP(uri.Hostname())

	res, err := ms.Scrape()
	assert.Nil(t, err, "scrape error")
	assert.Equal(t, res.Sets["node:test"].Labels[core.LabelMetricSetType.Key], core.MetricSetTypeNode)
}

func TestAddSummaryMetrics(t *testing.T) {

	ms := testingSummaryMetricsSource(1234)
	summary := getTestStatsSummary()

	expectations := getTestSummaryExpectations()
	dataBatch := &core.Batch{
		Timestamp: time.Now(),
		Sets:      map[core.ResourceKey]*core.Set{},
	}

	ms.addSummaryMetricSets(dataBatch, &summary)
	metrics := dataBatch.Sets
	for _, e := range expectations {
		m, ok := metrics[e.key]
		if !assert.True(t, ok, "missing metric %q", e.key) {
			continue
		}
		assert.Equal(t, m.Labels[core.LabelMetricSetType.Key], e.setType, e.key)
		assert.Equal(t, m.CollectionStartTime, startTime, e.key)
		assert.Equal(t, m.ScrapeTime, scrapeTime, e.key)
		if e.cpu {
			checkIntMetric(t, m, e.key, core.MetricCpuUsage, e.seed+offsetCPUUsageCoreSeconds)
		}
		if e.memory {
			checkIntMetric(t, m, e.key, core.MetricMemoryUsage, e.seed+offsetMemUsageBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryWorkingSet, e.seed+offsetMemWorkingSetBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryRSS, e.seed+offsetMemRSSBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryPageFaults, e.seed+offsetMemPageFaults)
			checkIntMetric(t, m, e.key, core.MetricMemoryMajorPageFaults, e.seed+offsetMemMajorPageFaults)
		}
		if e.network {
			checkIntMetric(t, m, e.key, core.MetricNetworkRx, e.seed+offsetNetRxBytes)
			checkIntMetric(t, m, e.key, core.MetricNetworkRxErrors, e.seed+offsetNetRxErrors)
			checkIntMetric(t, m, e.key, core.MetricNetworkTx, e.seed+offsetNetTxBytes)
			checkIntMetric(t, m, e.key, core.MetricNetworkTxErrors, e.seed+offsetNetTxErrors)
		}
		if e.accelerators {
			checkAcceleratorMetric(t, m, e.key, core.MetricAcceleratorMemoryTotal, e.seed+offsetAcceleratorMemoryTotal)
			checkAcceleratorMetric(t, m, e.key, core.MetricAcceleratorMemoryUsed, e.seed+offsetAcceleratorMemoryUsed)
			checkAcceleratorMetric(t, m, e.key, core.MetricAcceleratorDutyCycle, e.seed+offsetAcceleratorDutyCycle)
		}
		if e.ephemeralstorage {
			checkIntMetric(t, m, e.key, core.MetricEphemeralStorageUsage, e.seed+offsetFsUsed)
		}
		if e.containerEphemeralstorage {
			checkIntMetric(t, m, e.key, core.MetricEphemeralStorageUsage, 2*(e.seed+offsetFsUsed))
		}
		for _, label := range e.fs {
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemAvailable, e.seed+offsetFsAvailable)
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemLimit, e.seed+offsetFsCapacity)
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemUsage, e.seed+offsetFsUsed)
		}
		delete(metrics, e.key)
	}

	// Verify volume information labeled metrics
	var volumeInformationMetricsKey = core.PodKey(namespace0, pName3)
	var mappedVolumeStats = map[string]int64{}
	for _, labeledMetric := range metrics[volumeInformationMetricsKey].LabeledValues {
		assert.True(t, strings.HasPrefix("Volume:C", labeledMetric.Labels["resource_id"]))
		mappedVolumeStats[labeledMetric.Name] = labeledMetric.IntValue
	}

	assert.True(t, mappedVolumeStats["filesystem/available"] == int64(availableFsBytes))
	assert.True(t, mappedVolumeStats["filesystem/usage"] == int64(usedFsBytes))
	assert.True(t, mappedVolumeStats["filesystem/limit"] == int64(totalFsBytes))

	delete(metrics, volumeInformationMetricsKey)

	for k, v := range metrics {
		assert.Fail(t, "unexpected metric", "%q: %+v", k, v)
	}
}

func TestAddSummaryMetricsWithPvc(t *testing.T) {

	ms := testingSummaryMetricsSource(1234)
	summary := getTestStatsSummaryWithPvc()

	dataBatch := &core.Batch{
		Timestamp: time.Now(),
		Sets:      map[core.ResourceKey]*core.Set{},
	}

	ms.addSummaryMetricSets(dataBatch, &summary)
	metrics := dataBatch.Sets

	// Verify volume information labeled metrics
	var volumeInformationMetricsKey = core.PodKey(namespace0, pWithPvc)
	for _, labeledMetric := range metrics[volumeInformationMetricsKey].LabeledValues {
		assert.Equal(t, pvcName, labeledMetric.Labels["pvc_name"])
	}
}

func TestAddMissingRunningPodMetricSets(t *testing.T) {
	statusStartTime := metav1.NewTime(startTime)
	podList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-pod",
					Namespace: namespace0,
				},
				Status: v1.PodStatus{
					Phase:     v1.PodRunning,
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-pod-in-stats-summary",
					Namespace: namespace0,
					UID:       "podList UID",
				},
				Status: v1.PodStatus{
					Phase:     v1.PodRunning,
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-pod",
					Namespace: namespace0,
				},
				Status: v1.PodStatus{
					Phase:     v1.PodPending,
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "succeeded-pod",
					Namespace: namespace1,
				},
				Status: v1.PodStatus{
					Phase:     v1.PodSucceeded,
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-pod",
					Namespace: namespace1,
				},
				Status: v1.PodStatus{
					Phase:     v1.PodFailed,
					StartTime: &statusStartTime,
				},
			},
		},
	}

	ms := testingSummaryMetricsSource(1234)
	dataBatch := &core.Batch{
		Timestamp: time.Now(),
		Sets:      map[core.ResourceKey]*core.Set{},
	}

	t.Run("handles empty pod list", func(t *testing.T) {
		emptyPodList := v1.PodList{
			Items: []v1.Pod{},
		}
		ms.addMissingRunningPodMetricSets(dataBatch, &emptyPodList)
		assert.Equal(t, 0, len(dataBatch.Sets))
	})

	t.Run("only adds missing running pods", func(t *testing.T) {
		modifiedExistingPod := podList.Items[1]
		modifiedExistingPod.UID = "stats/summary uid"

		addFakePodMetric(dataBatch, ms, modifiedExistingPod)
		assert.Equal(t, 1, len(dataBatch.Sets))

		ms.addMissingRunningPodMetricSets(dataBatch, &podList)
		assert.Equal(t, 2, len(dataBatch.Sets))

		// Make sure the new running pod exists
		podMetrics := dataBatch.Sets[core.PodKey(namespace0, "running-pod")]
		assert.Equal(t, "running-pod", podMetrics.Labels[core.LabelPodName.Key])

		// Make sure we didn't overwrite an existing pod from the stats/summary output
		podMetrics = dataBatch.Sets[core.PodKey(namespace0, "running-pod-in-stats-summary")]
		assert.Equal(t, "stats/summary uid", podMetrics.Labels[core.LabelPodId.Key])
	})

	t.Run("pod metrics values", func(t *testing.T) {
		ms.addMissingRunningPodMetricSets(dataBatch, &podList)
		podMetrics := dataBatch.Sets[core.PodKey(namespace0, "running-pod")]
		pod := podList.Items[0]

		assert.Equal(t, core.MetricSetTypePod, podMetrics.Labels[core.LabelMetricSetType.Key])
		assert.Equal(t, pod.Status.StartTime.Time, podMetrics.CollectionStartTime)
		assert.Equal(t, dataBatch.Timestamp, podMetrics.ScrapeTime)
		assert.Equal(t, pod.Name, podMetrics.Labels[core.LabelPodName.Key])
		assert.Equal(t, pod.Namespace, podMetrics.Labels[core.LabelNamespaceName.Key])
		assert.Equal(t, ms.node.NodeName, podMetrics.Labels[core.LabelNodename.Key])
		assert.Equal(t, ms.node.HostName, podMetrics.Labels[core.LabelHostname.Key])
		assert.Equal(t, ms.node.HostID, podMetrics.Labels[core.LabelHostID.Key])
	})
}

func TestAddMissingPendingScheduledPodMetricSets(t *testing.T) {
	statusStartTime := metav1.NewTime(startTime)

	podList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-pod",
					Namespace: namespace0,
					UID:       "running-pod UID",
				},
				Status: v1.PodStatus{
					Phase:      v1.PodRunning,
					Conditions: []v1.PodCondition{},
					StartTime:  &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-and-scheduled-pod",
					Namespace: namespace0,
					UID:       "running-pod UID",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: "True",
						},
					},
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-scheduled-pod",
					Namespace: namespace0,
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: "True",
						},
					},
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-nonscheduled-pod",
					Namespace: namespace0,
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: "False",
						},
					},
					StartTime: &statusStartTime,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-unscheduled-pod",
					Namespace: namespace0,
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: "False",
						},
					},
					StartTime: &statusStartTime,
				},
			},
		},
	}

	ms := testingSummaryMetricsSource(1234)

	dataBatch := &core.Batch{
		Timestamp: time.Now(),
		Sets:      map[core.ResourceKey]*core.Set{},
	}

	t.Run("handles empty pod list", func(t *testing.T) {
		dataBatch.Sets = map[core.ResourceKey]*core.Set{}
		emptyPodList := v1.PodList{
			Items: []v1.Pod{},
		}
		ms.addMissingPendingScheduledPodMetricSets(dataBatch, &emptyPodList)
		assert.Equal(t, 0, len(dataBatch.Sets))
	})

	t.Run("only adds pending pods", func(t *testing.T) {
		dataBatch.Sets = map[core.ResourceKey]*core.Set{}
		ms.addMissingPendingScheduledPodMetricSets(dataBatch, &podList)

		// the pending pod is added
		_, present := dataBatch.Sets[core.PodKey(namespace0, "pending-scheduled-pod")]
		assert.True(t, present)

		// the running pods are not added
		assert.Equal(t, 1, len(dataBatch.Sets))
	})

	t.Run("skips if not condition scheduled/True", func(t *testing.T) {
		dataBatch.Sets = map[core.ResourceKey]*core.Set{}
		ms.addMissingPendingScheduledPodMetricSets(dataBatch, &podList)

		// the pending-nonscheduled-pod pod is not added
		_, present := dataBatch.Sets[core.PodKey(namespace0, "pending-nonscheduled-pod")]
		assert.False(t, present)
	})

	t.Run("skips if already in data batch", func(t *testing.T) {
		dataBatch.Sets = map[core.ResourceKey]*core.Set{}

		preExistingPod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pending-scheduled-pod",
				Namespace: namespace0,
				UID:       "uid-databatch",
			},
			Status: v1.PodStatus{
				Phase: v1.PodPending,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodScheduled,
						Status: "True",
					},
				},
				StartTime: &statusStartTime,
			},
		}

		addFakePodMetric(dataBatch, ms, preExistingPod)
		assert.Equal(t, 1, len(dataBatch.Sets))

		preExistingPod.UID = "uid-podlist"

		pods := v1.PodList{
			Items: []v1.Pod{preExistingPod},
		}

		ms.addMissingRunningPodMetricSets(dataBatch, &pods)
		assert.Equal(t, 1, len(dataBatch.Sets))

		// Make sure the UID has not been changed
		podMetrics := dataBatch.Sets[core.PodKey(namespace0, "pending-scheduled-pod")]
		assert.Equal(t, "uid-databatch", podMetrics.Labels[core.LabelPodId.Key])
	})

	t.Run("pod metrics values", func(t *testing.T) {
		dataBatch.Sets = map[core.ResourceKey]*core.Set{}

		ms.addMissingPendingScheduledPodMetricSets(dataBatch, &podList)
		podMetrics := dataBatch.Sets[core.PodKey(namespace0, "pending-scheduled-pod")]
		indexOfPendingScheduledPod := 2
		pod := podList.Items[indexOfPendingScheduledPod]

		assert.Equal(t, core.MetricSetTypePod, podMetrics.Labels[core.LabelMetricSetType.Key])
		assert.Equal(t, pod.Status.StartTime.Time, podMetrics.CollectionStartTime)
		assert.Equal(t, dataBatch.Timestamp, podMetrics.ScrapeTime)
		assert.Equal(t, pod.Name, podMetrics.Labels[core.LabelPodName.Key])
		assert.Equal(t, pod.Namespace, podMetrics.Labels[core.LabelNamespaceName.Key])
		assert.Equal(t, ms.node.NodeName, podMetrics.Labels[core.LabelNodename.Key])
		assert.Equal(t, ms.node.HostName, podMetrics.Labels[core.LabelHostname.Key])
		assert.Equal(t, ms.node.HostID, podMetrics.Labels[core.LabelHostID.Key])
	})
}

func TestDecodeEphemeralStorageStatsForContainer(t *testing.T) {
	ms := testingSummaryMetricsSource(1234)
	rootFs := &stats.FsStats{}
	logs := &stats.FsStats{}
	assert.NotPanics(t, func() { ms.decodeEphemeralStorageStatsForContainer(nil, rootFs, logs) })
}

func testingSummaryMetricsSource(port uint) *summaryMetricsSource {
	client, _ := kubelet.NewKubeletClient(&kubelet.KubeletClientConfig{Port: port})
	return &summaryMetricsSource{
		node:          nodeInfo,
		kubeletClient: client,
	}
}

func getTestSummaryExpectations() []struct {
	key                       core.ResourceKey
	setType                   string
	seed                      int64
	cpu                       bool
	memory                    bool
	network                   bool
	accelerators              bool
	ephemeralstorage          bool
	containerEphemeralstorage bool
	fs                        []string
} {
	containerFs := []string{"/", "logs"}
	expectations := []struct {
		key                       core.ResourceKey
		setType                   string
		seed                      int64
		cpu                       bool
		memory                    bool
		network                   bool
		accelerators              bool
		ephemeralstorage          bool
		containerEphemeralstorage bool
		fs                        []string
	}{{
		key:              core.NodeKey(nodeInfo.NodeName),
		setType:          core.MetricSetTypeNode,
		seed:             seedNode,
		cpu:              true,
		memory:           true,
		network:          true,
		ephemeralstorage: true,
		fs:               []string{"/"},
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "kubelet"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedKubelet,
		cpu:     true,
		memory:  true,
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "docker-daemon"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedRuntime,
		cpu:     true,
		memory:  true,
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "system"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedMisc,
		cpu:     true,
		memory:  true,
	}, {
		key:              core.PodKey(namespace0, pName0),
		setType:          core.MetricSetTypePod,
		seed:             seedPod0,
		network:          true,
		cpu:              true,
		memory:           true,
		ephemeralstorage: true,
	}, {
		key:     core.PodKey(namespace0, pName1),
		setType: core.MetricSetTypePod,
		seed:    seedPod1,
		network: true,
		fs:      []string{"Volume:A", "Volume:B"},
	}, {
		key:     core.PodKey(namespace1, pName2),
		setType: core.MetricSetTypePod,
		seed:    seedPod2,
		network: true,
	}, {
		key:     core.PodKey(namespace0, pName4),
		setType: core.MetricSetTypePod,
		seed:    seedPod4,
		network: true,
	}, {
		key:     core.PodKey(namespace0, pName5),
		setType: core.MetricSetTypePod,
		seed:    seedPod5,
		network: true,
	}, {
		key:                       core.PodContainerKey(namespace0, pName0, cName00),
		setType:                   core.MetricSetTypePodContainer,
		seed:                      seedPod0Container0,
		cpu:                       true,
		memory:                    true,
		containerEphemeralstorage: true,
		fs:                        containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName0, cName01),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod0Container1,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName1, cName10),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod1Container,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace1, pName2, cName20),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod2Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace1, pName2, cName21),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod2Container1,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName3, cName30),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod3Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName4, cName40),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod4Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:          core.PodContainerKey(namespace0, pName5, cName50),
		setType:      core.MetricSetTypePodContainer,
		seed:         seedPod5Container0,
		cpu:          true,
		accelerators: true,
	}}
	return expectations
}

func getTestStatsSummary() stats.Summary {
	summary := stats.Summary{
		Node: stats.NodeStats{
			NodeName:  nodeInfo.NodeName,
			StartTime: metav1.NewTime(startTime),
			CPU:       genTestSummaryCPU(seedNode),
			Memory:    genTestSummaryMemory(seedNode),
			Network:   genTestSummaryNetwork(seedNode),
			SystemContainers: []stats.ContainerStats{
				genTestSummaryContainer(stats.SystemContainerKubelet, seedKubelet),
				genTestSummaryContainer(stats.SystemContainerRuntime, seedRuntime),
				genTestSummaryContainer(stats.SystemContainerMisc, seedMisc),
			},
			Fs: genTestSummaryFsStats(seedNode),
		},
		Pods: []stats.PodStats{{
			PodRef: stats.PodReference{
				Name:      pName0,
				Namespace: namespace0,
			},
			StartTime:        metav1.NewTime(startTime),
			Network:          genTestSummaryNetwork(seedPod0),
			EphemeralStorage: genTestSummaryFsStats(seedPod0),
			CPU:              genTestSummaryCPU(seedPod0),
			Memory:           genTestSummaryMemory(seedPod0),
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName00, seedPod0Container0),
				genTestSummaryContainer(cName01, seedPod0Container1),
				genTestSummaryTerminatedContainer(cName00, seedPod0Container0),
			},
		}, {
			PodRef: stats.PodReference{
				Name:      pName1,
				Namespace: namespace0,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod1),
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName10, seedPod1Container),
			},
			VolumeStats: []stats.VolumeStats{{
				Name:    "A",
				FsStats: *genTestSummaryFsStats(seedPod1),
			}, {
				Name:    "B",
				FsStats: *genTestSummaryFsStats(seedPod1),
			}},
		}, {
			PodRef: stats.PodReference{
				Name:      pName2,
				Namespace: namespace1,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod2),
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName20, seedPod2Container0),
				genTestSummaryContainer(cName21, seedPod2Container1),
			},
		}, {
			PodRef: stats.PodReference{
				Name:      pName3,
				Namespace: namespace0,
			},
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName30, seedPod3Container0),
			},
			VolumeStats: []stats.VolumeStats{{
				Name: "C",
				FsStats: stats.FsStats{
					AvailableBytes: &availableFsBytes,
					UsedBytes:      &usedFsBytes,
					CapacityBytes:  &totalFsBytes,
					InodesFree:     &freeInode,
					InodesUsed:     &usedInode,
					Inodes:         &totalInode,
				},
			},
			},
		}, {
			PodRef: stats.PodReference{
				Name:      pName4,
				Namespace: namespace0,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod4),
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName40, seedPod4Container0),
				genTestSummaryTerminatedContainerNoStats(cName41),
				genTestSummaryTerminatedContainerBlankStats(cName42),
			},
		}, {
			PodRef: stats.PodReference{
				Name:      pName5,
				Namespace: namespace0,
			},
			Network:   genTestSummaryNetwork(seedPod5),
			StartTime: metav1.NewTime(startTime),
			Containers: []stats.ContainerStats{
				genTestSummaryContainerWithAccelerator(cName50, seedPod5Container0),
			},
		}},
	}
	return summary
}

func getTestStatsSummaryWithPvc() stats.Summary {
	summary := stats.Summary{
		Node: stats.NodeStats{
			NodeName:  nodeInfo.NodeName,
			StartTime: metav1.NewTime(startTime),
			CPU:       genTestSummaryCPU(seedNode),
			Memory:    genTestSummaryMemory(seedNode),
			Network:   genTestSummaryNetwork(seedNode),
			SystemContainers: []stats.ContainerStats{
				genTestSummaryContainer(stats.SystemContainerKubelet, seedKubelet),
				genTestSummaryContainer(stats.SystemContainerRuntime, seedRuntime),
				genTestSummaryContainer(stats.SystemContainerMisc, seedMisc),
			},
			Fs: genTestSummaryFsStats(seedNode),
		},
		Pods: []stats.PodStats{{
			PodRef: stats.PodReference{
				Name:      pWithPvc,
				Namespace: namespace0,
			},
			Containers: []stats.ContainerStats{
				genTestSummaryContainer(cName30, seedPod3Container0),
			},
			VolumeStats: []stats.VolumeStats{{
				Name: "pvc-claim",
				PVCRef: &stats.PVCReference{
					Name:      pvcName,
					Namespace: namespace0,
				},
				FsStats: stats.FsStats{
					AvailableBytes: &availableFsBytes,
					UsedBytes:      &usedFsBytes,
					CapacityBytes:  &totalFsBytes,
					InodesFree:     &freeInode,
					InodesUsed:     &usedInode,
					Inodes:         &totalInode,
				},
			}},
		}},
	}
	return summary
}

func genTestSummaryTerminatedContainer(name string, seed int) stats.ContainerStats {
	return stats.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
		CPU:       genTestSummaryZeroCPU(seed),
		Memory:    genTestSummaryZeroMemory(seed),
		Rootfs:    genTestSummaryFsStats(seed),
		Logs:      genTestSummaryFsStats(seed),
	}
}

func genTestSummaryTerminatedContainerNoStats(name string) stats.ContainerStats {
	return stats.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
	}
}

func genTestSummaryTerminatedContainerBlankStats(name string) stats.ContainerStats {
	return stats.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
		CPU:       genTestSummaryBlankCPU(),
		Memory:    genTestSummaryBlankMemory(),
	}
}

func genTestSummaryContainer(name string, seed int) stats.ContainerStats {
	return stats.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime),
		CPU:       genTestSummaryCPU(seed),
		Memory:    genTestSummaryMemory(seed),
		Rootfs:    genTestSummaryFsStats(seed),
		Logs:      genTestSummaryFsStats(seed),
	}
}

func genTestSummaryContainerWithAccelerator(name string, seed int) stats.ContainerStats {
	return stats.ContainerStats{
		Name:         name,
		StartTime:    metav1.NewTime(startTime),
		CPU:          genTestSummaryCPU(seed),
		Accelerators: genTestSummaryAccelerator(seed),
	}
}

func genTestSummaryAccelerator(seed int) []stats.AcceleratorStats {
	return []stats.AcceleratorStats{
		{
			Make:        "nvidia",
			Model:       "Tesla P100",
			ID:          "GPU-deadbeef-1234-5678-90ab-feedfacecafe",
			MemoryTotal: *testhelper.Uint64Val(seed, offsetAcceleratorMemoryTotal),
			MemoryUsed:  *testhelper.Uint64Val(seed, offsetAcceleratorMemoryUsed),
			DutyCycle:   *testhelper.Uint64Val(seed, offsetAcceleratorDutyCycle),
		},
	}
}

func genTestSummaryZeroCPU(seed int) *stats.CPUStats {
	cpu := stats.CPUStats{
		Time:                 metav1.NewTime(scrapeTime),
		UsageNanoCores:       testhelper.Uint64Val(seed, -seed),
		UsageCoreNanoSeconds: testhelper.Uint64Val(seed, offsetCPUUsageCoreSeconds),
	}
	*cpu.UsageCoreNanoSeconds *= uint64(time.Millisecond.Nanoseconds())
	return &cpu
}

func genTestSummaryCPU(seed int) *stats.CPUStats {
	cpu := stats.CPUStats{
		Time:                 metav1.NewTime(scrapeTime),
		UsageNanoCores:       testhelper.Uint64Val(seed, offsetCPUUsageCores),
		UsageCoreNanoSeconds: testhelper.Uint64Val(seed, offsetCPUUsageCoreSeconds),
	}
	*cpu.UsageNanoCores *= uint64(time.Millisecond.Nanoseconds())
	return &cpu
}

func genTestSummaryBlankCPU() *stats.CPUStats {
	return &stats.CPUStats{
		Time: metav1.NewTime(scrapeTime),
	}
}

func genTestSummaryZeroMemory(seed int) *stats.MemoryStats {
	return &stats.MemoryStats{
		Time:            metav1.NewTime(scrapeTime),
		UsageBytes:      testhelper.Uint64Val(seed, offsetMemUsageBytes),
		WorkingSetBytes: testhelper.Uint64Val(seed, offsetMemWorkingSetBytes),
		RSSBytes:        testhelper.Uint64Val(seed, -seed),
		PageFaults:      testhelper.Uint64Val(seed, offsetMemPageFaults),
		MajorPageFaults: testhelper.Uint64Val(seed, offsetMemMajorPageFaults),
	}
}

func genTestSummaryMemory(seed int) *stats.MemoryStats {
	return &stats.MemoryStats{
		Time:            metav1.NewTime(scrapeTime),
		UsageBytes:      testhelper.Uint64Val(seed, offsetMemUsageBytes),
		WorkingSetBytes: testhelper.Uint64Val(seed, offsetMemWorkingSetBytes),
		RSSBytes:        testhelper.Uint64Val(seed, offsetMemRSSBytes),
		PageFaults:      testhelper.Uint64Val(seed, offsetMemPageFaults),
		MajorPageFaults: testhelper.Uint64Val(seed, offsetMemMajorPageFaults),
	}
}

func genTestSummaryBlankMemory() *stats.MemoryStats {
	return &stats.MemoryStats{
		Time: metav1.NewTime(scrapeTime),
	}
}

func genTestSummaryNetwork(seed int) *stats.NetworkStats {
	return &stats.NetworkStats{
		Time: metav1.NewTime(scrapeTime),
		InterfaceStats: stats.InterfaceStats{
			RxBytes:  testhelper.Uint64Val(seed, offsetNetRxBytes),
			RxErrors: testhelper.Uint64Val(seed, offsetNetRxErrors),
			TxBytes:  testhelper.Uint64Val(seed, offsetNetTxBytes),
			TxErrors: testhelper.Uint64Val(seed, offsetNetTxErrors),
		},
	}
}

func genTestSummaryFsStats(seed int) *stats.FsStats {
	return &stats.FsStats{
		AvailableBytes: testhelper.Uint64Val(seed, offsetFsAvailable),
		CapacityBytes:  testhelper.Uint64Val(seed, offsetFsCapacity),
		UsedBytes:      testhelper.Uint64Val(seed, offsetFsUsed),
	}
}

func checkIntMetric(t *testing.T, metrics *core.Set, key core.ResourceKey, metric core.Metric, value int64) {
	m, ok := metrics.Values[metric.Name]
	if !assert.True(t, ok, "missing %q:%q", key, metric.Name) {
		return
	}
	assert.Equal(t, value, m.IntValue, "%q:%q", key, metric.Name)
}

func checkFsMetric(t *testing.T, metrics *core.Set, key core.ResourceKey, label string, metric core.Metric, value int64) {
	for _, m := range metrics.LabeledValues {
		if m.Name == metric.Name && m.Labels[core.LabelResourceID.Key] == label {
			assert.Equal(t, value, m.IntValue, "%q:%q[%s]", key, metric.Name, label)
			return
		}
	}
	assert.Fail(t, "missing filesystem metric", "%q:[%q]:%q", key, metric.Name, label)
}

func checkAcceleratorMetric(t *testing.T, metrics *core.Set, key core.ResourceKey, metric core.Metric, value int64) {
	for _, m := range metrics.LabeledValues {
		if m.Name == metric.Name {
			assert.Equal(t, value, m.IntValue, "%q:%q", key, metric.Name)
			return
		}
	}
	assert.Fail(t, "missing accelerator metric", "%q:[%q]", key, metric.Name)
}

func addFakePodMetric(dataBatch *core.Batch, ms *summaryMetricsSource, pod v1.Pod) {
	podMetrics := &core.Set{
		Labels:              map[string]string{},
		Values:              map[string]core.Value{},
		LabeledValues:       []core.LabeledValue{},
		CollectionStartTime: pod.Status.StartTime.Time,
		ScrapeTime:          dataBatch.Timestamp,
	}

	podMetrics.Labels[core.LabelMetricSetType.Key] = core.MetricSetTypePod
	podMetrics.Labels[core.LabelPodId.Key] = string(pod.UID)
	podMetrics.Labels[core.LabelPodName.Key] = pod.Name
	podMetrics.Labels[core.LabelNamespaceName.Key] = pod.Namespace

	dataBatch.Sets[core.PodKey(pod.Namespace, pod.Name)] = podMetrics
	uptime := uint64(time.Since(startTime).Nanoseconds() / time.Millisecond.Nanoseconds())
	cpu := uint64(1000)
	ms.addIntMetric(podMetrics, &core.MetricUptime, &uptime)
	ms.addIntMetric(podMetrics, &core.MetricCpuUsage, &cpu)
}
