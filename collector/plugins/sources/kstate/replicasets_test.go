package kstate

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicReplicaSet() *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-replicaset",
			Labels: nil,
		},
		Spec: appsv1.ReplicaSetSpec{},
		Status: appsv1.ReplicaSetStatus{
			AvailableReplicas: 1,
			ReadyReplicas:     1,
		},
	}
}

func setupReplicaSetWithOwner() *appsv1.ReplicaSet {
	replicaset := setupBasicReplicaSet()
	replicaset.OwnerReferences = []metav1.OwnerReference{{
		Kind: "Deployment",
		Name: "someOwner",
	}}
	return replicaset
}

func TestPointsForReplicaSet(t *testing.T) {
	testTransform := setupWorkloadTransform()
	workloadMetricName := testTransform.Prefix + workloadStatusMetric

	t.Run("test for ReplicaSet metrics with OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()
		expectedMetricNames := []string{
			"kubernetes.replicaset.desired_replicas",
			"kubernetes.replicaset.available_replicas",
			"kubernetes.replicaset.ready_replicas",
			"kubernetes.workload.status",
		}

		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for ReplicaSet metrics without OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupReplicaSetWithOwner()
		expectedMetricNames := []string{
			"kubernetes.replicaset.desired_replicas",
			"kubernetes.replicaset.available_replicas",
			"kubernetes.replicaset.ready_replicas",
		}

		actualWFPoints := pointsForReplicaSet(testReplicaSet, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for ReplicaSet with healthy status and no OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()

		actualWFPointsMap := getWFPointsMap(pointsForReplicaSet(testReplicaSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindReplicaSet, actualWFPoint.Tags()[workloadKindTag])
	})

	t.Run("test for ReplicaSet with non healthy status and no OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupBasicReplicaSet()
		testReplicaSet.Status.ReadyReplicas = 0
		testReplicaSet.Status.AvailableReplicas = 0

		actualWFPointsMap := getWFPointsMap(pointsForReplicaSet(testReplicaSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindReplicaSet, actualWFPoint.Tags()[workloadKindTag])
	})

}
