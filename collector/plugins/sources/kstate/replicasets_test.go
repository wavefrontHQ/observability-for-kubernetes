package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicReplicaSet() *appsv1.ReplicaSet {
	replicaset := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "replicasetset1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
		},
	}
	return replicaset
}

func setupReplicaSetWithOwner() *appsv1.ReplicaSet {
	replicaset := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "replicasetset1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment",
				Name: "someOwner",
			}},
		},
	}
	return replicaset
}

func setupTestReplicaSetTransform() configuration.Transforms {
	return configuration.Transforms{
		Source:  "testSource",
		Prefix:  "kubernetes.",
		Tags:    nil,
		Filters: filter.Config{},
	}
}

func Test_pointsForReplicaSet(t *testing.T) {
	testTransform := setupTestReplicaSetTransform()

	t.Run("test for basic ReplicaSet", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()
		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))
		expectedMetricNames := []string{
			"kubernetes.replicaset.desired_replicas",
			"kubernetes.replicaset.available_replicas",
			"kubernetes.replicaset.ready_replicas",
			"kubernetes.workload.status",
		}

		var actualMetricNames []string

		for _, point := range actualWFPoints {
			actualMetricNames = append(actualMetricNames, point.Name())
		}

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)
		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for ReplicaSet with OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupReplicaSetWithOwner()
		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		assert.Equal(t, 3, len(actualWFPoints))
		expectedMetricNames := []string{
			"kubernetes.replicaset.desired_replicas",
			"kubernetes.replicaset.available_replicas",
			"kubernetes.replicaset.ready_replicas",
		}

		var actualMetricNames []string

		for _, point := range actualWFPoints {
			actualMetricNames = append(actualMetricNames, point.Name())
		}

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)
		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for ReplicaSet healthy with no OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()

		testReplicaSet.Spec.Replicas = new(int32)
		*testReplicaSet.Spec.Replicas = 1
		testReplicaSet.Status.ReadyReplicas = 1
		testReplicaSet.Status.AvailableReplicas = 1

		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))

		expectedPointValue := float64(1)
		for _, point := range actualWFPoints {
			wfpoint := point.(*wf.Point)
			assert.Equal(t, expectedPointValue, wfpoint.Value)
		}
	})

	t.Run("test for ReplicaSet not healthy with no OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()

		testReplicaSet.Spec.Replicas = new(int32)
		*testReplicaSet.Spec.Replicas = 1
		testReplicaSet.Status.ReadyReplicas = 0
		testReplicaSet.Status.AvailableReplicas = 0

		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))

		point := actualWFPoints[3].(*wf.Point)
		assert.Zero(t, point.Value)
	})

}
