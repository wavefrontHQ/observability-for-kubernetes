package kstate

import (
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func setupBasicDeploymentWorkload() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-deployment-workload",
			Labels: nil,
		},
		Spec: appsv1.DeploymentSpec{},
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
		numberDesired := int32(1)
		numberAvailable := int32(1)

		testTags := buildWorkloadTags(workloadKindDeployment, testDeployment.Name, "", numberDesired, numberAvailable, "", testTransform.Tags)

		assert.Equal(t, expectedWorkloadName, testTags[workloadNameTag])
		assert.Equal(t, workloadKindDeployment, testTags[workloadKindTag])
		assert.Equal(t, fmt.Sprintf("%d", numberDesired), testTags[workloadDesiredTag])
		assert.Equal(t, fmt.Sprintf("%d", numberAvailable), testTags[workloadAvailableTag])

		workloadStatus := getWorkloadStatus(numberDesired, numberAvailable)
		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, workloadStatus, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, workloadStatusMetricName, point.Name())
		assert.Equal(t, workloadReady, point.Value)
	})

	t.Run("test for deployment workload status not ready", func(t *testing.T) {
		testDeployment := setupBasicDeploymentWorkload()
		expectedWorkloadName := testDeployment.Name
		numberDesired := int32(1)
		numberAvailable := int32(0)

		testTags := buildWorkloadTags(workloadKindDeployment, testDeployment.Name, "", numberDesired, numberAvailable, "", testTransform.Tags)

		assert.Equal(t, expectedWorkloadName, testTags[workloadNameTag])
		assert.Equal(t, workloadKindDeployment, testTags[workloadKindTag])
		assert.Equal(t, fmt.Sprintf("%d", numberDesired), testTags[workloadDesiredTag])
		assert.Equal(t, fmt.Sprintf("%d", numberAvailable), testTags[workloadAvailableTag])

		workloadStatus := getWorkloadStatus(numberDesired, numberAvailable)
		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, workloadStatus, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, workloadStatusMetricName, point.Name())
		assert.Equal(t, workloadNotReady, point.Value)
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
