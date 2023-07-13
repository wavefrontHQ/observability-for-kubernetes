package kstate

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
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

func Test_pointsForJob(t *testing.T) {
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
		assert.Equal(t, 1, actualWFPoints[5].Points())

	})

	t.Run("test for Failed Job metrics without OwnerReferences", func(t *testing.T) {
		testJob := setupBasicJob()
		testJob.Status.Failed = 1

		actualWFPoints := pointsForJob(testJob, testTransform)
		assert.Equal(t, 0, actualWFPoints[5].Points())

	})
	//t.Run("test for Job metrics without OwnerReferences", func(t *testing.T) {
	//	testJob := setupJobWithOwner()
	//	expectedMetricNames := []string{
	//		"kubernetes.job.desired_replicas",
	//		"kubernetes.job.available_replicas",
	//		"kubernetes.job.ready_replicas",
	//	}
	//
	//	actualWFPoints := pointsForJob(testJob, testTransform)
	//	actualMetricNames := getTestWFMetricNames(actualWFPoints)
	//
	//	assert.Equal(t, len(expectedMetricNames), len(actualMetricNames))
	//
	//	sort.Strings(expectedMetricNames)
	//	sort.Strings(actualMetricNames)
	//
	//	assert.Equal(t, expectedMetricNames, actualMetricNames)
	//})
	//
	//t.Run("test for Job with healthy status and no OwnerReferences", func(t *testing.T) {
	//	testJob := setupBasicJob()
	//	workloadMetricName := "kubernetes.workload.status"
	//
	//	actualWFPointsMap := getWFPointsMap(pointsForJob(testJob, testTransform))
	//	actualWFPoint := actualWFPointsMap[workloadMetricName]
	//
	//	assert.Equal(t, workloadReady, actualWFPoint.Value)
	//})
	//
	//t.Run("test for Job with non healthy status and no OwnerReferences", func(t *testing.T) {
	//	testJob := setupBasicJob()
	//	workloadMetricName := "kubernetes.workload.status"
	//	testJob.Status.ReadyReplicas = 0
	//	testJob.Status.AvailableReplicas = 0
	//
	//	actualWFPointsMap := getWFPointsMap(pointsForJob(testJob, testTransform))
	//	actualWFPoint := actualWFPointsMap[workloadMetricName]
	//
	//	assert.Equal(t, workloadNotReady, actualWFPoint.Value)
	//})

}
