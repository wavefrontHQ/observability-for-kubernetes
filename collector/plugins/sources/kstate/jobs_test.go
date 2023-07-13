package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicJob() *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "basic-job",
			Labels: nil,
		},
		Spec: batchv1.JobSpec{},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
}

func setupJobWithOwner() *batchv1.Job {
	job := setupBasicJob()
	job.OwnerReferences = []metav1.OwnerReference{{
		Kind: "Cronjob",
		Name: "someOwner",
	}}
	return job
}

func TestPointsForJob(t *testing.T) {
	testTransform := setupWorkloadTransform()

	t.Run("test for Successful Job metrics without OwnerReferences", func(t *testing.T) {
		testJob := setupBasicJob()

		expectedMetricNames := []string{
			"kubernetes.job.active",
			"kubernetes.job.failed",
			"kubernetes.job.succeeded",
			"kubernetes.job.completions",
			"kubernetes.job.parallelism",
			"kubernetes.workload.status",
		}

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
		assert.Equal(t, workloadReady, actualWFPoints[5].(*wf.Point).Value)
	})

	t.Run("test for Failed Job metrics without OwnerReferences", func(t *testing.T) {
		testJob := setupBasicJob()
		testJob.Status.Failed = 1

		actualWFPoints := pointsForJob(testJob, testTransform)
		assert.Equal(t, workloadNotReady, actualWFPoints[5].(*wf.Point).Value)

	})

	t.Run("test for Successful Job metrics without OwnerReferences", func(t *testing.T) {
		testJob := setupJobWithOwner()

		expectedMetricNames := []string{
			"kubernetes.job.active",
			"kubernetes.job.failed",
			"kubernetes.job.succeeded",
			"kubernetes.job.completions",
			"kubernetes.job.parallelism",
		}

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualMetricNames := getTestWFMetricNames(actualWFPoints)

		assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)

		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})
}
