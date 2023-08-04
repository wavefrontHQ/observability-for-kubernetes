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

func setupBasicDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-deployment",
			Labels: nil,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: genericPointer(int32(1)),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}
}

func setupFailedDeployment() *appsv1.Deployment {
	failedDeployment := setupBasicDeployment()
	failedDeployment.Status.ReadyReplicas = 0
	failedDeployment.Status.AvailableReplicas = 0
	failedDeployment.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:    appsv1.DeploymentAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  "MinimumReplicasUnavailable",
		Message: "Deployment does not have minimum availability.",
	}}
	return failedDeployment
}

func TestPointsForDeployment(t *testing.T) {
	testTransform := setupWorkloadTransform()
	workloadMetricName := testTransform.Prefix + workloadStatusMetric

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
		expectedAvailable := fmt.Sprint(testDeployment.Status.AvailableReplicas)
		expectedDesired := fmt.Sprint(*testDeployment.Spec.Replicas)

		actualWFPointsMap := getWFPointsMap(pointsForDeployment(testDeployment, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDeployment, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, actualWFPoint.Tags()[workloadDesiredTag])
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, actualWFPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("test for Deployment workload with non healthy status", func(t *testing.T) {
		testDeployment := setupFailedDeployment()
		expectedAvailable := fmt.Sprint(testDeployment.Status.AvailableReplicas)
		expectedReason := testDeployment.Status.Conditions[0].Reason
		expectedMessage := testDeployment.Status.Conditions[0].Message

		actualWFPointsMap := getWFPointsMap(pointsForDeployment(testDeployment, testTransform))
		actualWFPoint, found := actualWFPointsMap[workloadMetricName]
		assert.True(t, found)

		assert.Equal(t, workloadNotReady, actualWFPoint.Value)
		assert.Equal(t, workloadKindDeployment, actualWFPoint.Tags()[workloadKindTag])

		assert.Equal(t, expectedAvailable, actualWFPoint.Tags()[workloadAvailableTag])
		assert.NotEqual(t, actualWFPoint.Tags()[workloadDesiredTag], actualWFPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedReason, actualWFPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, actualWFPoint.Tags()[workloadFailedMessageTag])
	})
}
