package kstate

import (
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

type nilTest struct {
	pointFunc  func(interface{}, configuration.Transforms) []wf.Metric
	nonPointer interface{}
	wrongType  interface{}
}

func TestIncorrectInput(t *testing.T) {
	allPointFuncTests := []nilTest{
		{pointsForCronJob, batchv1.CronJob{}, &corev1.Pod{}},
		{pointsForDaemonSet, appsv1.DaemonSet{}, &corev1.Pod{}},
		{pointsForDeployment, appsv1.Deployment{}, &corev1.Pod{}},
		{pointsForHPA, autoscalingv2.HorizontalPodAutoscaler{}, &corev1.Pod{}},
		{pointsForJob, batchv1.Job{}, &corev1.Pod{}},
		{pointsForNode, corev1.Node{}, &corev1.Pod{}},
		{pointsForNonRunningPods(fakeWorkloadCache{}), corev1.Pod{}, &appsv1.Deployment{}},
		{pointsForPV, corev1.PersistentVolume{}, &corev1.Pod{}},
		{pointsForPVC, corev1.PersistentVolumeClaim{}, &corev1.Pod{}},
		{pointsForReplicaSet, appsv1.ReplicaSet{}, &corev1.Pod{}},
		{pointsForReplicationController, corev1.ReplicationController{}, &corev1.Pod{}},
		{pointsForStatefulSet, appsv1.StatefulSet{}, &corev1.Pod{}},
	}

	for _, pointFuncNilTest := range allPointFuncTests {
		require.Nil(t, pointFuncNilTest.pointFunc(pointFuncNilTest.nonPointer, setupTestTransform()))
		require.Nil(t, pointFuncNilTest.pointFunc(pointFuncNilTest.wrongType, setupTestTransform()))
	}
}
