package kstate

import (
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
		Spec: appsv1.StatefulSetSpec{},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   1,
			CurrentReplicas: 1,
			UpdatedReplicas: 0,
		},
	}
}

func Test_pointsForStatefulSet(t *testing.T) {
	testTransform := setupWorkloadTransform()

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
		workloadMetricName := "kubernetes.workload.status"

		actualWFPointsMap := getWFPointsMap(pointsForStatefulSet(testStatefulSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindStatefulSet, actualWFPoint.Tags()["workload_kind"])
	})

	t.Run("test for StatefulSet with non healthy status", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()
		workloadMetricName := "kubernetes.workload.status"
		testStatefulSet.Status.ReadyReplicas = 0
		testStatefulSet.Status.CurrentReplicas = 0

		actualWFPointsMap := getWFPointsMap(pointsForStatefulSet(testStatefulSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindStatefulSet, actualWFPoint.Tags()["workload_kind"])
	})

}
