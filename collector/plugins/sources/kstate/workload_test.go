package kstate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func setupWorkloadTransform() configuration.Transforms {
	return configuration.Transforms{Prefix: "kubernetes.", Source: "test-source-workload"}
}

func Test_buildWorkloadStatusMetric(t *testing.T) {
	testTransform := setupWorkloadTransform()
	timestamp := time.Now().Unix()

	t.Run("test for workload status ready", func(t *testing.T) {
		testDeployment := setupBasicDeployment()
		numberDesired := 1.0
		numberReady := 1.0

		testTags := buildWorkloadTags("deployment", testDeployment.Name, "", testTransform.Tags)

		assert.Equal(t, "basic-deployment", testTags[workloadNameTag])
		assert.Equal(t, "deployment", testTags[workloadTypeTag])

		actualWFPoint := buildWorkloadStatusMetric(testTransform.Prefix, numberDesired, numberReady, timestamp, testTransform.Source, testTags)
		point := actualWFPoint.(*wf.Point)

		assert.Equal(t, "kubernetes.workload.status", point.Name())
		assert.Equal(t, workloadReady, point.Value)
	})

	t.Run("test for workload status not ready", func(t *testing.T) {
		testDeployment := setupBasicDeployment()
		numberDesired := 1.0
		numberReady := 0.0

		testTags := buildWorkloadTags("deployment", testDeployment.Name, "", testTransform.Tags)

		assert.Equal(t, "basic-deployment", testTags[workloadNameTag])
		assert.Equal(t, "deployment", testTags[workloadTypeTag])

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
