package kstate

import (
	batchv1 "k8s.io/api/batch/v1"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	corev1 "k8s.io/api/core/v1"
)

func TestPointsForJob(t *testing.T) {
	t.Run("creates active, failed, succeeded, completions, parallelism metrics", func(t *testing.T) {
		var expectedCompletions int32 = 8
		var expectedParallelism int32 = 10

		fakeJob := &batchv1.Job{
			Status: batchv1.JobStatus{
				Active:    2,
				Failed:    4,
				Succeeded: 6,
			},
			Spec: batchv1.JobSpec{
				Completions: &expectedCompletions,
				Parallelism: &expectedParallelism,
			},
		}

		actualWFPoints := pointsForJob(fakeJob, setupTestTransform())
		actualJobActive := actualWFPoints[0].(*wf.Point)
		actualJobFailed := actualWFPoints[1].(*wf.Point)
		actualJobSucceeded := actualWFPoints[2].(*wf.Point)
		actualJobCompletions := actualWFPoints[3].(*wf.Point)
		actualJobParallelism := actualWFPoints[4].(*wf.Point)

		require.Equal(t, float64(2), actualJobActive.Value)
		require.Equal(t, "testPrefix.job.active", actualJobActive.Name())

		require.Equal(t, float64(4), actualJobFailed.Value)
		require.Equal(t, "testPrefix.job.failed", actualJobFailed.Name())

		require.Equal(t, float64(6), actualJobSucceeded.Value)
		require.Equal(t, "testPrefix.job.succeeded", actualJobSucceeded.Name())

		require.Equal(t, float64(8), actualJobCompletions.Value)
		require.Equal(t, "testPrefix.job.completions", actualJobCompletions.Name())

		require.Equal(t, float64(10), actualJobParallelism.Value)
		require.Equal(t, "testPrefix.job.parallelism", actualJobParallelism.Name())
	})

	t.Run("completions defaults to -1.0 if not provided", func(t *testing.T) {
		actualWFPoints := pointsForJob(&batchv1.Job{}, setupTestTransform())
		actualJobCompletions := actualWFPoints[3].(*wf.Point)

		require.Equal(t, float64(-1), actualJobCompletions.Value)
	})

	t.Run("parallelism defaults to -1.0 if not provided", func(t *testing.T) {
		actualWFPoints := pointsForJob(&batchv1.Job{}, setupTestTransform())
		actualJobParallelism := actualWFPoints[4].(*wf.Point)

		require.Equal(t, float64(-1), actualJobParallelism.Value)
	})

	t.Run("returns nil if type is invalid", func(t *testing.T) {
		require.Nil(t, pointsForJob(batchv1.Job{}, setupTestTransform()))
		require.Nil(t, pointsForJob(&corev1.Pod{}, setupTestTransform()))
	})
}
