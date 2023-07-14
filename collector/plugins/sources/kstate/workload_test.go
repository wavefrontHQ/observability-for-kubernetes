package kstate

import (
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
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
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

func Test_buildWorkloadStatusMetric(t *testing.T) {
	testTransform := setupWorkloadTransform()
	timestamp := time.Now().Unix()

	t.Run("test for deployment workload status ready", func(t *testing.T) {
		testDeployment := setupBasicDeploymentWorkload()
		numberDesired := 1.0
		numberReady := 1.0

		testTags := buildWorkloadTags(testDeployment.Kind, testDeployment.Name, "", testTransform.Tags)

		assert.Equal(t, "basic-deployment-workload", testTags[workloadNameTag])
		assert.Equal(t, "Deployment", testTags[workloadKindTag])

		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, numberDesired, numberReady, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, "kubernetes.workload.status", point.Name())
		assert.Equal(t, workloadReady, point.Value)
	})

	t.Run("test for deployment workload status not ready", func(t *testing.T) {
		testDeployment := setupBasicDeploymentWorkload()
		numberDesired := 1.0
		numberReady := 0.0

		testTags := buildWorkloadTags(testDeployment.Kind, testDeployment.Name, "", testTransform.Tags)

		assert.Equal(t, "basic-deployment-workload", testTags[workloadNameTag])
		assert.Equal(t, "Deployment", testTags[workloadKindTag])

		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, numberDesired, numberReady, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, "kubernetes.workload.status", point.Name())
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
