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

func setupBasicDaemonSet() *appsv1.DaemonSet {
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daemonsetset1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
		},
	}
	return daemonset
}

func setupTestDaemonsetTransform() configuration.Transforms {
	return configuration.Transforms{
		Source:  "testSource",
		Prefix:  "kubernetes.",
		Tags:    nil,
		Filters: filter.Config{},
	}
}

func Test_pointsForDaemonSet(t *testing.T) {
	testTransform := setupTestDaemonsetTransform()

	t.Run("test for basic DaemonSet", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()
		actualWFPoints := pointsForDaemonSet(testDaemonSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))
		expectedMetricNames := []string{
			"kubernetes.daemonset.current_scheduled",
			"kubernetes.daemonset.desired_scheduled",
			"kubernetes.daemonset.misscheduled",
			"kubernetes.daemonset.ready",
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

	t.Run("test for DaemonSet healthy", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()

		testDaemonSet.Status.CurrentNumberScheduled = 1
		testDaemonSet.Status.DesiredNumberScheduled = 1
		testDaemonSet.Status.NumberMisscheduled = 1
		testDaemonSet.Status.NumberReady = 1

		actualWFPoints := pointsForDaemonSet(testDaemonSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))

		expectedPointValue := float64(1)
		for _, point := range actualWFPoints {
			wfpoint := point.(*wf.Point)
			assert.Equal(t, expectedPointValue, wfpoint.Value)
		}
	})

	t.Run("test for DaemonSet not healthy", func(t *testing.T) {
		testDaemonSet := setupBasicDaemonSet()

		testDaemonSet.Status.CurrentNumberScheduled = 0
		testDaemonSet.Status.DesiredNumberScheduled = 1
		testDaemonSet.Status.NumberMisscheduled = 1
		testDaemonSet.Status.NumberReady = 0

		actualWFPoints := pointsForDaemonSet(testDaemonSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))

		point := actualWFPoints[4].(*wf.Point)
		assert.Zero(t, point.Value)
	})
}
