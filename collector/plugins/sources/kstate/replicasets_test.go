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
		Spec: appsv1.ReplicaSetSpec{
			Replicas: genericPointer(int32(1)),
		},
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

func setupFailedReplicaSet() *appsv1.ReplicaSet {
	replicaset := setupBasicReplicaSet()
	replicaset.Status.Conditions = []appsv1.ReplicaSetCondition{{
		Type:    appsv1.ReplicaSetReplicaFailure,
		Status:  corev1.ConditionTrue,
		Reason:  "SomeFailureReason",
		Message: "Some failure message.",
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
		expectedAvailable := fmt.Sprint(testReplicaSet.Status.AvailableReplicas)
		expectedDesired := fmt.Sprint(*testReplicaSet.Spec.Replicas)

		actualWFPointsMap := getWFPointsMap(pointsForReplicaSet(testReplicaSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindReplicaSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, actualWFPoint.Tags()[workloadDesiredTag])
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("test for ReplicaSet with non healthy status and no OwnerReferences", func(t *testing.T) {
		testReplicaSet := setupFailedReplicaSet()
		testReplicaSet.Status.ReadyReplicas = 0
		testReplicaSet.Status.AvailableReplicas = 0
		expectedAvailable := fmt.Sprint(testReplicaSet.Status.AvailableReplicas)

		actualWFPointsMap := getWFPointsMap(pointsForReplicaSet(testReplicaSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindReplicaSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.NotEqual(t, actualWFPoint.Tags()[workloadDesiredTag], actualWFPoint.Tags()[workloadAvailableTag])
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

}
