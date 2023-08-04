package kstate

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicStatefulSet() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-statefulset",
			Labels: nil,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: genericPointer(int32(1)),
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   1,
			CurrentReplicas: 1,
			UpdatedReplicas: 1,
		},
	}
}

func TestPointsForStatefulSet(t *testing.T) {
	testTransform := setupWorkloadTransform()
	workloadMetricName := testTransform.Prefix + workloadStatusMetric

	t.Run("test for StatefulSet metrics", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()
		expectedMetricNames := []string{
			"kubernetes.statefulset.current_replicas",
			"kubernetes.statefulset.desired_replicas",
			"kubernetes.statefulset.ready_replicas",
			"kubernetes.statefulset.updated_replicas",
			"kubernetes.workload.status",
		}

		actualWFPoints := pointsForStatefulSet(testStatefulSet, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for StatefulSet with healthy status", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()
		expectedAvailable := fmt.Sprint(testStatefulSet.Status.ReadyReplicas)
		expectedDesired := fmt.Sprint(*testStatefulSet.Spec.Replicas)

		actualWFPointsMap := getWFPointsMap(pointsForStatefulSet(testStatefulSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindStatefulSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, actualWFPoint.Tags()[workloadDesiredTag])
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("test for StatefulSet with non healthy status", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()
		testStatefulSet.Status.ReadyReplicas = 0
		testStatefulSet.Status.CurrentReplicas = 0
		expectedAvailable := fmt.Sprint(testStatefulSet.Status.ReadyReplicas)

		actualWFPointsMap := getWFPointsMap(pointsForStatefulSet(testStatefulSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindStatefulSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.NotEqual(t, actualWFPoint.Tags()[workloadDesiredTag], actualWFPoint.Tags()[workloadAvailableTag])
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

}
