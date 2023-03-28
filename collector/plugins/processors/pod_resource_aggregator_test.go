package processors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	kube_api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestPodResourceAggregator(t *testing.T) {
	t.Run("it adds request and limit metrics for a pod", func(t *testing.T) {
		// Setup
		pod := &kube_api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "ns1",
			},
			Spec: kube_api.PodSpec{
				Overhead: map[kube_api.ResourceName]resource.Quantity{
					kube_api.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
				},
				InitContainers: []kube_api.Container{
					{
						Name: "init-container-1",
						Resources: kube_api.ResourceRequirements{
							Requests: kube_api.ResourceList{
								kube_api.ResourceCPU:              *resource.NewMilliQuantity(8000, resource.DecimalSI),
								kube_api.ResourceMemory:           *resource.NewQuantity(444, resource.DecimalSI),
								kube_api.ResourceEphemeralStorage: *resource.NewQuantity(100, resource.DecimalSI),
								"some-other-resource":             *resource.NewQuantity(200, resource.DecimalSI),
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
				Containers: []kube_api.Container{
					{
						Name: "container-1",
						Resources: kube_api.ResourceRequirements{
							Requests: kube_api.ResourceList{
								kube_api.ResourceCPU:              *resource.NewMilliQuantity(100, resource.DecimalSI),
								kube_api.ResourceMemory:           *resource.NewQuantity(555, resource.DecimalSI),
								kube_api.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
							},
						},
					},
					{
						Name: "container-2",
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
		}

		batch := &metrics.Batch{
			Timestamp: time.Now(),
			Sets: map[metrics.ResourceKey]*metrics.Set{
				metrics.PodKey("ns1", "pod2"): {
					Labels: map[string]string{
						metrics.LabelMetricSetType.Key: metrics.MetricSetTypePod,
						metrics.LabelPodName.Key:       "pod1",
						metrics.LabelNamespaceName.Key: "ns1",
					},
					Values: map[string]metrics.Value{
						"m1": {
							ValueType: metrics.ValueInt64,
							IntValue:  100,
						},
					},
				},
			},
		}

		store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		err := store.Add(pod)
		assert.NoError(t, err)

		podLister := v1listers.NewPodLister(store)
		aggregator := NewPodResourceAggregator(podLister)

		// Execute
		batch, err = aggregator.Process(batch)

		// Test
		assert.NoError(t, err)

		metricSet := batch.Sets[metrics.PodKey("ns1", "pod2")]
		assert.NotNil(t, metricSet)

		expected := map[string]metrics.Value{
			"cpu/limit":                     {IntValue: 2322},
			"cpu/request":                   {IntValue: 8100},
			"ephemeral_storage/limit":       {IntValue: 5000},
			"ephemeral_storage/request":     {IntValue: 3000},
			"example.com/resource1/limit":   {IntValue: 2},
			"example.com/resource1/request": {IntValue: 2},
			"some-other-resource/request":   {IntValue: 200},
			"m1":                            {IntValue: 100},
			"memory/limit":                  {IntValue: 3333},
			"memory/request":                {IntValue: 1555},
		}

		assert.Equal(t, expected, metricSet.Values)
	})
	t.Run("it skips metric sets that are not for pods", func(t *testing.T) {
		batch := &metrics.Batch{
			Timestamp: time.Now(),
			Sets: map[metrics.ResourceKey]*metrics.Set{
				metrics.PodContainerKey("ns1", "pod1", "container1"): {
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

		aggregator := NewPodResourceAggregator(nil)

		// Execute
		batch, err := aggregator.Process(batch)

		// Test
		assert.NoError(t, err)

		metricSet := batch.Sets[metrics.PodKey("ns1", "pod2")]
		assert.Nil(t, metricSet)

		metricSet = batch.Sets[metrics.PodContainerKey("ns1", "pod1", "container1")]
		assert.NotNil(t, metricSet)

		assert.Empty(t, metricSet.Values)
	})
	t.Run("it skips pods that don't have pod name or namespace name labels", func(t *testing.T) {
		batch := &metrics.Batch{
			Timestamp: time.Now(),
			Sets: map[metrics.ResourceKey]*metrics.Set{
				metrics.PodKey("ns1", "pod2"): {
					Labels: map[string]string{
						metrics.LabelMetricSetType.Key: metrics.MetricSetTypePod,
					},
					Values: map[string]metrics.Value{},
				},
			},
		}

		aggregator := NewPodResourceAggregator(nil)

		// Execute
		batch, err := aggregator.Process(batch)

		// Test
		assert.NoError(t, err)

		metricSet := batch.Sets[metrics.PodKey("ns1", "pod2")]
		assert.NotNil(t, metricSet)

		assert.Empty(t, metricSet.Values)
	})
	t.Run("it deletes metric sets from the batch if the pod cannot be found in the kube api cache", func(t *testing.T) {
		batch := &metrics.Batch{
			Timestamp: time.Now(),
			Sets: map[metrics.ResourceKey]*metrics.Set{
				metrics.PodKey("ns1", "pod2"): {
					Labels: map[string]string{
						metrics.LabelMetricSetType.Key: metrics.MetricSetTypePod,
						metrics.LabelPodName.Key:       "pod1",
						metrics.LabelNamespaceName.Key: "ns1",
					},
					Values: map[string]metrics.Value{},
				},
			},
		}
		store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

		podLister := v1listers.NewPodLister(store)
		aggregator := NewPodResourceAggregator(podLister)

		// Execute
		batch, err := aggregator.Process(batch)

		// Test
		assert.NoError(t, err)

		metricSet := batch.Sets[metrics.PodKey("ns1", "pod2")]
		assert.Nil(t, metricSet)
	})
}
