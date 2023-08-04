package kstate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func setupBasicDeploymentWorkload() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-deployment-workload",
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

func setupWorkloadTransform() configuration.Transforms {
	return configuration.Transforms{Prefix: "kubernetes.", Source: "test-source-workload"}
}

func TestBuildWorkloadStatusMetric(t *testing.T) {
	testTransform := setupWorkloadTransform()
	timestamp := time.Now().Unix()
	workloadStatusMetricName := testTransform.Prefix + workloadStatusMetric

	t.Run("test for deployment workload status ready", func(t *testing.T) {
		testDeployment := setupBasicDeploymentWorkload()
		expectedWorkloadName := testDeployment.Name
		numberDesired := *testDeployment.Spec.Replicas
		numberAvailable := testDeployment.Status.AvailableReplicas

		testTags := buildWorkloadTags(workloadKindDeployment, testDeployment.Name, "", numberDesired, numberAvailable, "", "", testTransform.Tags)

		assert.Equal(t, expectedWorkloadName, testTags[workloadNameTag])
		assert.Equal(t, workloadKindDeployment, testTags[workloadKindTag])
		assert.Equal(t, fmt.Sprint(numberDesired), testTags[workloadDesiredTag])
		assert.Equal(t, fmt.Sprint(numberAvailable), testTags[workloadAvailableTag])
		assert.NotContains(t, testTags, workloadFailedReasonTag)
		assert.NotContains(t, testTags, workloadFailedMessageTag)

		workloadStatus := getWorkloadStatus(numberDesired, numberAvailable)
		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, workloadStatus, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, workloadStatusMetricName, point.Metric)
		assert.Equal(t, workloadReady, point.Value)
	})

	t.Run("test for deployment workload status not ready", func(t *testing.T) {
		testDeployment := setupBasicDeploymentWorkload()
		expectedWorkloadName := testDeployment.Name
		numberDesired := *testDeployment.Spec.Replicas
		testDeployment.Status.AvailableReplicas = 0
		numberAvailable := testDeployment.Status.AvailableReplicas
		testDeployment.Status.Conditions = []appsv1.DeploymentCondition{{
			Type:    appsv1.DeploymentAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "MinimumReplicasUnavailable",
			Message: "Deployment does not have minimum availability.",
		}}
		failureReason := testDeployment.Status.Conditions[0].Reason
		failureMessage := testDeployment.Status.Conditions[0].Message

		testTags := buildWorkloadTags(workloadKindDeployment, testDeployment.Name, "", numberDesired, numberAvailable, failureReason, failureMessage, testTransform.Tags)

		assert.Equal(t, expectedWorkloadName, testTags[workloadNameTag])
		assert.Equal(t, workloadKindDeployment, testTags[workloadKindTag])
		assert.Equal(t, fmt.Sprint(numberDesired), testTags[workloadDesiredTag])
		assert.Equal(t, fmt.Sprint(numberAvailable), testTags[workloadAvailableTag])
		assert.Equal(t, failureReason, testTags[workloadFailedReasonTag])
		assert.Equal(t, failureMessage, testTags[workloadFailedMessageTag])

		workloadStatus := getWorkloadStatus(numberDesired, numberAvailable)
		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, workloadStatus, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, workloadStatusMetricName, point.Metric)
		assert.Equal(t, workloadNotReady, point.Value)
	})

	t.Run("sanitizes message tags with double quotes", func(t *testing.T) {
		failureMessage := "Back-off pulling image \"busybox123\"."
		failureMessageRawString := `Back-off pulling image "busybox123".`
		expectedMessage := "Back-off pulling image busybox123."

		testTags := buildWorkloadTags(workloadKindDeployment, "some-name", "", 1, 0, "SomeReason", failureMessage, map[string]string{})
		assert.Equal(t, expectedMessage, testTags[workloadFailedMessageTag])

		testTagsRawString := buildWorkloadTags(workloadKindDeployment, "some-name", "", 1, 0, "SomeReason", failureMessageRawString, map[string]string{})
		assert.Equal(t, expectedMessage, testTagsRawString[workloadFailedMessageTag])
	})
}

func getTestWFMetricNames(points []wf.Metric) []string {
	var metricNames []string
	for _, point := range points {
		metricNames = append(metricNames, point.Name())
	}
	return metricNames
}

func getWFPointsMap(points []wf.Metric) map[string]*wf.Point {
	metricsToPoints := make(map[string]*wf.Point, len(points))
	for _, point := range points {
		metricsToPoints[point.Name()] = point.(*wf.Point)
	}
	return metricsToPoints
}
