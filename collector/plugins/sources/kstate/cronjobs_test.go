package kstate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
)

func TestPointsForCronJob(t *testing.T) {
	t.Run("creates cronjob.active metric based on number of active cronjobs", func(t *testing.T) {
		fakeCronJob := &batchv1.CronJob{
			Status: batchv1.CronJobStatus{
				Active: []v1.ObjectReference{
					{
						Kind:      cronJobs,
						Name:      "fake-cron-job",
						Namespace: "fake-namespace",
					},
				},
			},
		}

		actualWFPoints := pointsForCronJob(fakeCronJob, setupTestTransform())
		point := actualWFPoints[0].(*wf.Point)

		require.Equal(t, float64(1), point.Value)
	})

	t.Run("returns nil if type is invalid", func(t *testing.T) {
		require.Nil(t, pointsForCronJob(batchv1.CronJob{}, setupTestTransform()))
		require.Nil(t, pointsForCronJob(&v1.Pod{}, setupTestTransform()))
	})
}
