package kstate

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	v1 "k8s.io/api/core/v1"
)

func TestPointsForHPA(t *testing.T) {
	t.Run("creates metrics for HPA max, min, current, and desired replicas", func(t *testing.T) {
		var expectedMinReplicas int32 = 4
		fakeHPA := &autoscalingv2.HorizontalPodAutoscaler{
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 6,
				DesiredReplicas: 8,
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				MinReplicas: &expectedMinReplicas,
				MaxReplicas: 2,
			},
		}

		actualWFPoints := pointsForHPA(fakeHPA, setupTestTransform())
		actualHPAMaxReplicas := actualWFPoints[0].(*wf.Point)
		actualHPAMinReplicas := actualWFPoints[1].(*wf.Point)
		actualHPACurrentReplicas := actualWFPoints[2].(*wf.Point)
		actualHPADesiredReplicas := actualWFPoints[3].(*wf.Point)

		require.Equal(t, float64(2), actualHPAMaxReplicas.Value)
		require.Equal(t, "testPrefix.hpa.max_replicas", actualHPAMaxReplicas.Name())

		require.Equal(t, float64(4), actualHPAMinReplicas.Value)
		require.Equal(t, "testPrefix.hpa.min_replicas", actualHPAMinReplicas.Name())

		require.Equal(t, float64(6), actualHPACurrentReplicas.Value)
		require.Equal(t, "testPrefix.hpa.current_replicas", actualHPACurrentReplicas.Name())

		require.Equal(t, float64(8), actualHPADesiredReplicas.Value)
		require.Equal(t, "testPrefix.hpa.desired_replicas", actualHPADesiredReplicas.Name())
	})

	t.Run("returns nil if type is invalid", func(t *testing.T) {
		require.Nil(t, pointsForHPA(autoscalingv2.HorizontalPodAutoscaler{}, setupTestTransform()))
		require.Nil(t, pointsForHPA(&v1.Pod{}, setupTestTransform()))
	})
}
