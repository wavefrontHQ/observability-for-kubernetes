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

func setupBasicStatefulSet() *appsv1.StatefulSet {
	specReplicahelper := int32(1)
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &specReplicahelper,
		},
	}
	return statefulset
}

func setupTestStatefulSetTransform() configuration.Transforms {
	return configuration.Transforms{
		Source:  "testSource",
		Prefix:  "kubernetes.",
		Tags:    nil,
		Filters: filter.Config{},
	}
}

func Test_pointsForStatefulSet(t *testing.T) {
	testTransform := setupTestStatefulSetTransform()

	t.Run("test for basic StatefulSet", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()
		actualWFPoints := pointsForStatefulSet(testStatefulSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))
		expectedMetricNames := []string{
			"kubernetes.statefulset.current_replicas",
			"kubernetes.statefulset.desired_replicas",
			"kubernetes.statefulset.ready_replicas",
			"kubernetes.statefulset.updated_replicas",
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

	t.Run("test for StatefulSet healthy", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()

		testStatefulSet.Spec.Replicas = new(int32)
		*testStatefulSet.Spec.Replicas = 1
		testStatefulSet.Status.ReadyReplicas = 1
		testStatefulSet.Status.CurrentReplicas = 1
		testStatefulSet.Status.UpdatedReplicas = 1

		actualWFPoints := pointsForStatefulSet(testStatefulSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))

		expectedPointValue := float64(1)
		for _, point := range actualWFPoints {
			wfpoint := point.(*wf.Point)
			assert.Equal(t, expectedPointValue, wfpoint.Value)
		}
	})

	t.Run("test for StatefulSet not healthy", func(t *testing.T) {
		testStatefulSet := setupBasicStatefulSet()

		testStatefulSet.Spec.Replicas = new(int32)
		*testStatefulSet.Spec.Replicas = 1
		testStatefulSet.Status.ReadyReplicas = 0
		testStatefulSet.Status.CurrentReplicas = 0
		testStatefulSet.Status.UpdatedReplicas = 0

		actualWFPoints := pointsForStatefulSet(testStatefulSet, testTransform)
		assert.Equal(t, 5, len(actualWFPoints))

		point := actualWFPoints[4].(*wf.Point)
		assert.Zero(t, point.Value)
	})

}
