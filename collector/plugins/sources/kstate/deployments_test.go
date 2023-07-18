package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-deployment",
			Labels: nil,
		},
		Spec: appsv1.DeploymentSpec{},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}
}

func Test_pointsForDeployment(t *testing.T) {
	testTransform := setupWorkloadTransform()

	t.Run("test for Deployment metrics", func(t *testing.T) {
		testDeployment := setupBasicDeployment()
		expectedMetricNames := []string{
			"kubernetes.deployment.desired_replicas",
			"kubernetes.deployment.available_replicas",
			"kubernetes.deployment.ready_replicas",
			"kubernetes.workload.status",
		}

		actualWFPoints := pointsForDeployment(testDeployment, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("test for Deployment workload with healthy status", func(t *testing.T) {
		testDeployment := setupBasicDeployment()
		workloadMetricName := "kubernetes.workload.status"

		actualWFPointsMap := getWFPointsMap(pointsForDeployment(testDeployment, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDeployment, actualWFPoint.Tags()["workload_kind"])
	})

	t.Run("test for Deployment workload with non healthy status", func(t *testing.T) {
		testDeployment := setupBasicDeployment()
		workloadMetricName := "kubernetes.workload.status"
		testDeployment.Status.AvailableReplicas = 0
		testDeployment.Status.ReadyReplicas = 0

		actualWFPointsMap := getWFPointsMap(pointsForDeployment(testDeployment, testTransform))
		actualWFPoint := actualWFPointsMap[workloadMetricName]

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDeployment, actualWFPoint.Tags()["workload_kind"])
	})
}
