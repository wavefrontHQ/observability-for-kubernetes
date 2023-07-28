package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
			Failed:    0,
			Active:    0,
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}

func setupFailedJob() *batchv1.Job {
	job := setupBasicJob()
	job.Status.Conditions = []batchv1.JobCondition{{
		Type:    batchv1.JobFailed,
		Status:  corev1.ConditionTrue,
		Reason:  "BackoffLimitExceeded",
		Message: "Job has reached the specified backoff limit",
	}}
	job.Status.Failed = 1
	job.Status.Succeeded = 0
	return job
}

func setupJobWithOwner() *batchv1.Job {
	job := setupBasicJob()
	job.OwnerReferences = []metav1.OwnerReference{{
		Kind: "CronJob",
		Name: "someOwner",
	}}
	return job
}

func TestPointsForJob(t *testing.T) {
	testTransform := setupWorkloadTransform()
	workloadStatusMetricName := testTransform.Prefix + workloadStatusMetric

	t.Run("test for Job metrics", func(t *testing.T) {
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
	})

	t.Run("workload status metrics should have available and desired tags", func(t *testing.T) {
		testJob := setupBasicJob()

		actualWFPointsMap := getWFPointsMap(pointsForJob(testJob, testTransform))
		actualWorkloadStatusPoint, found := actualWFPointsMap[workloadStatusMetricName]
		assert.True(t, found)

		expectedAvailable := "1"
		expectedDesired := "1"
		assert.Equal(t, expectedAvailable, actualWorkloadStatusPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, actualWorkloadStatusPoint.Tags()[workloadDesiredTag])
	})

	t.Run("healthy workload status metrics should not have reason and message tags", func(t *testing.T) {
		testJob := setupBasicJob()

		actualWFPointsMap := getWFPointsMap(pointsForJob(testJob, testTransform))
		actualWorkloadStatusPoint, found := actualWFPointsMap[workloadStatusMetricName]
		assert.True(t, found)

		assert.NotContains(t, actualWorkloadStatusPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, actualWorkloadStatusPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("unhealthy workload status metrics should have reason and message tags", func(t *testing.T) {
		testJob := setupFailedJob()
		expectedReason := testJob.Status.Conditions[0].Reason
		expectedMessage := testJob.Status.Conditions[0].Message

		actualWFPointsMap := getWFPointsMap(pointsForJob(testJob, testTransform))
		actualWorkloadStatusPoint, found := actualWFPointsMap[workloadStatusMetricName]
		assert.True(t, found)

		assert.Contains(t, actualWorkloadStatusPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, actualWorkloadStatusPoint.Tags(), workloadFailedMessageTag)
		assert.Equal(t, expectedReason, actualWorkloadStatusPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, actualWorkloadStatusPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("Non-parallel Job without OwnerReferences has a ready workload status", func(t *testing.T) {
		// For a non-parallel Job, you can leave both .spec.completions and .spec.parallelism unset.
		// When both are unset, both are defaulted to 1.
		testJob := setupBasicJob()

		actualWFPoints := pointsForJob(testJob, testTransform)
		assert.NotNil(t, actualWFPoints)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)
		assert.Greater(t, len(actualWFPointsMap), 0)

		actualWorkloadStatusPoint, found := actualWFPointsMap[workloadStatusMetricName]
		assert.True(t, found)
		assert.Equal(t, workloadReady, actualWorkloadStatusPoint.Value)
		assert.Equal(t, testJob.Name, actualWorkloadStatusPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindJob, actualWorkloadStatusPoint.Tags()[workloadKindTag])
	})

	t.Run("Non-parallel Job without OwnerReferences does not have a ready workload status", func(t *testing.T) {
		testJob := setupFailedJob()

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWorkloadStatusPoint := getWFPointsMap(actualWFPoints)[workloadStatusMetricName]

		assert.Equal(t, workloadNotReady, actualWorkloadStatusPoint.Value)
		assert.Equal(t, workloadKindJob, actualWorkloadStatusPoint.Tags()[workloadKindTag])
	})

	t.Run("Non-parallel Job with OwnerReferences has a ready workload status", func(t *testing.T) {
		testJob := setupJobWithOwner()
		expectedOwnerName := "someOwner"

		actualWFPoints := pointsForJob(testJob, testTransform)
		assert.NotNil(t, actualWFPoints)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)
		assert.Greater(t, len(actualWFPointsMap), 0)

		actualWorkloadStatusPoint, found := actualWFPointsMap[workloadStatusMetricName]
		assert.True(t, found)
		assert.Equal(t, workloadReady, actualWorkloadStatusPoint.Value)
		assert.Equal(t, expectedOwnerName, actualWorkloadStatusPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindCronJob, actualWorkloadStatusPoint.Tags()[workloadKindTag])
	})

	t.Run("Non-parallel Job with OwnerReferences does not have a ready workload status", func(t *testing.T) {
		testJob := setupJobWithOwner()
		testJob.Status.Conditions[0].Type = batchv1.JobFailed
		testJob.Status.Conditions[0].Status = corev1.ConditionTrue
		testJob.Status.Failed = 2

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWorkloadStatusPoint := getWFPointsMap(actualWFPoints)[workloadStatusMetricName]

		assert.Equal(t, workloadNotReady, actualWorkloadStatusPoint.Value)
		assert.Equal(t, workloadKindCronJob, actualWorkloadStatusPoint.Tags()[workloadKindTag])
	})

	t.Run("Parallel Job with a fixed completion count has a ready workload status", func(t *testing.T) {
		// For a fixed completion count Job, you should set .spec.completions to the number of completions needed.
		// You can set .spec.parallelism, or leave it unset, and it will default to 1.
		testJob := setupBasicJob()
		testJob.Status.Succeeded = 2
		completionsCount := 2
		parallelismCount := 2
		testJob.Spec.Completions = genericPointer(int32(completionsCount))
		testJob.Spec.Parallelism = genericPointer(int32(parallelismCount))

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)

		actualJobSucceededPoint := actualWFPointsMap[testTransform.Prefix+"job.succeeded"]
		actualJobCompletionsPoint := actualWFPointsMap[testTransform.Prefix+"job.completions"]
		actualJobParallelismPoint := actualWFPointsMap[testTransform.Prefix+"job.parallelism"]
		assert.Equal(t, float64(testJob.Status.Succeeded), actualJobSucceededPoint.Value)
		assert.Equal(t, float64(completionsCount), actualJobCompletionsPoint.Value)
		assert.Equal(t, float64(parallelismCount), actualJobParallelismPoint.Value)

		actualWorkloadStatusPoint := actualWFPointsMap[workloadStatusMetricName]
		assert.Equal(t, workloadReady, actualWorkloadStatusPoint.Value)
	})

	t.Run("Parallel Job with a fixed completion count does not have a ready workload status", func(t *testing.T) {
		testJob := setupFailedJob()
		testJob.Status.Failed = 5
		testJob.Status.Succeeded = 3
		completionsCount := 12
		parallelismCount := 3
		testJob.Spec.Completions = genericPointer(int32(completionsCount))
		testJob.Spec.Parallelism = genericPointer(int32(parallelismCount))

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)

		actualJobSucceededPoint := actualWFPointsMap[testTransform.Prefix+"job.succeeded"]
		actualJobCompletionsPoint := actualWFPointsMap[testTransform.Prefix+"job.completions"]
		actualJobParallelismPoint := actualWFPointsMap[testTransform.Prefix+"job.parallelism"]
		assert.Less(t, actualJobSucceededPoint.Value, actualJobCompletionsPoint.Value)
		assert.Equal(t, float64(completionsCount), actualJobCompletionsPoint.Value)
		assert.Equal(t, float64(parallelismCount), actualJobParallelismPoint.Value)

		actualWorkloadStatusPoint := actualWFPointsMap[workloadStatusMetricName]
		assert.Equal(t, workloadNotReady, actualWorkloadStatusPoint.Value)
	})

	t.Run("Parallel Job with a with a work queue has a ready workload status", func(t *testing.T) {
		// For a work queue Job, you must leave .spec.completions unset,
		// and set .spec.parallelism to a non-negative integer.
		testJob := setupBasicJob()
		testJob.Status.Succeeded = 2
		parallelismCount := 2
		testJob.Spec.Parallelism = genericPointer(int32(parallelismCount))

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)

		actualJobSucceededPoint := actualWFPointsMap[testTransform.Prefix+"job.succeeded"]
		actualJobCompletionsPoint := actualWFPointsMap[testTransform.Prefix+"job.completions"]
		actualJobParallelismPoint := actualWFPointsMap[testTransform.Prefix+"job.parallelism"]
		assert.Negative(t, actualJobCompletionsPoint.Value)
		assert.Positive(t, actualJobParallelismPoint.Value)
		assert.Positive(t, actualJobSucceededPoint.Value)

		actualWorkloadStatusPoint := actualWFPointsMap[workloadStatusMetricName]
		assert.Equal(t, workloadReady, actualWorkloadStatusPoint.Value)
	})

	t.Run("Parallel Job with a with a work queue does not have a ready workload status", func(t *testing.T) {
		testJob := setupFailedJob()
		testJob.Status.Failed = 5
		parallelismCount := 2
		testJob.Spec.Parallelism = genericPointer(int32(parallelismCount))

		actualWFPoints := pointsForJob(testJob, testTransform)
		actualWFPointsMap := getWFPointsMap(actualWFPoints)

		actualJobSucceededPoint := actualWFPointsMap[testTransform.Prefix+"job.succeeded"]
		actualJobCompletionsPoint := actualWFPointsMap[testTransform.Prefix+"job.completions"]
		actualJobParallelismPoint := actualWFPointsMap[testTransform.Prefix+"job.parallelism"]
		assert.Negative(t, actualJobCompletionsPoint.Value)
		assert.Positive(t, actualJobParallelismPoint.Value)
		assert.Zero(t, actualJobSucceededPoint.Value)

		actualWorkloadStatusPoint := actualWFPointsMap[workloadStatusMetricName]
		assert.Equal(t, workloadNotReady, actualWorkloadStatusPoint.Value)
	})
}
