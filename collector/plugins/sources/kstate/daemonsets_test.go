package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-daemonset",
			Labels: nil,
		},
		Spec: appsv1.DaemonSetSpec{},
		Status: appsv1.DaemonSetStatus{
			CurrentNumberScheduled: 1,
			DesiredNumberScheduled: 1,
			NumberMisscheduled:     0,
			NumberReady:            1,
		},
	}
}

func Test_pointsForDaemonSet(t *testing.T) {
	testTransform := setupWorkloadTransform()

	t.Run("test for DaemonSet metrics", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()
		expectedMetricNames := []string{
			"kubernetes.daemonset.current_scheduled",
			"kubernetes.daemonset.desired_scheduled",
			"kubernetes.daemonset.misscheduled",
			"kubernetes.daemonset.ready",
			"kubernetes.workload.status",
		}

		actualWFPoints := pointsForDaemonSet(testDaemonSet, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for DaemonSet with healthy status", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()
		workloadMetricName := "kubernetes.workload.status"

		actualWFPointsMap := getWFPointsMap(pointsForDaemonSet(testDaemonSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadReady, actualWFPoint.Value)
	})

	t.Run("test for DaemonSet with non healthy status", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()
		workloadMetricName := "kubernetes.workload.status"
		testDaemonSet.Status.CurrentNumberScheduled = 0
		testDaemonSet.Status.NumberReady = 0

		actualWFPointsMap := getWFPointsMap(pointsForDaemonSet(testDaemonSet, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
	})
}
