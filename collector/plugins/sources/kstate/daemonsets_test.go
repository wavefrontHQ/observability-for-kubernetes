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
			NumberAvailable:        1,
		},
	}
}

func TestPointsForDaemonSet(t *testing.T) {
	testTransform := setupWorkloadTransform()
	workloadMetricName := testTransform.Prefix + workloadStatusMetric

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
		expectedAvailable := "1"
		expectedDesired := "1"

		actualWFPointsMap := getWFPointsMap(pointsForDaemonSet(testDaemonSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDaemonSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, actualWFPoint.Tags()[workloadDesiredTag])
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("test for DaemonSet with non healthy status", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()
		testDaemonSet.Status.NumberReady = 0
		testDaemonSet.Status.NumberAvailable = 0
		expectedAvailable := "0"

		actualWFPointsMap := getWFPointsMap(pointsForDaemonSet(testDaemonSet, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDaemonSet, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.NotEqual(t, actualWFPoint.Tags()[workloadDesiredTag], actualWFPoint.Tags()[workloadAvailableTag])
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})
}
