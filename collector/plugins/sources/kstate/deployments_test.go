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

func setupBasicDeployment() *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploymentset1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
		},
	}
	return deployment
}

func setupTestDeploymentTransform() configuration.Transforms {
	return configuration.Transforms{
		Source:  "testSource",
		Prefix:  "kubernetes.",
		Tags:    nil,
		Filters: filter.Config{},
	}
}

func Test_pointsForDeployment(t *testing.T) {
	testTransform := setupTestDeploymentTransform()

	t.Run("test for basic Deployment", func(t *testing.T) {
		testDeployment := setupBasicDeployment()

		actualWFPoints := pointsForDeployment(testDeployment, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))
		expectedMetricNames := []string{
			"kubernetes.deployment.desired_replicas",
			"kubernetes.deployment.available_replicas",
			"kubernetes.deployment.ready_replicas",
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

	t.Run("test for Deployment healthy", func(t *testing.T) {
		testDeployment := setupBasicDeployment()

		testDeployment.Spec.Replicas = new(int32)
		*testDeployment.Spec.Replicas = 1
		testDeployment.Status.ReadyReplicas = 1
		testDeployment.Status.AvailableReplicas = 1

		actualWFPoints := pointsForDeployment(testDeployment, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))

		expectedPointValue := float64(1)
		for _, point := range actualWFPoints {
			wfpoint := point.(*wf.Point)
			assert.Equal(t, expectedPointValue, wfpoint.Value)
		}
	})

	t.Run("test for Deployment not healthy", func(t *testing.T) {
		testDeployment := setupBasicDeployment()

		testDeployment.Spec.Replicas = new(int32)
		*testDeployment.Spec.Replicas = 1
		testDeployment.Status.ReadyReplicas = 0
		testDeployment.Status.AvailableReplicas = 0

		actualWFPoints := pointsForDeployment(testDeployment, testTransform)
		assert.Equal(t, 4, len(actualWFPoints))

		point := actualWFPoints[3].(*wf.Point)
		assert.Zero(t, point.Value)
	})
}
