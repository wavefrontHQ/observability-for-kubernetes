package kstate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func setupBasicPod() *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
			Labels:    map[string]string{"name": "testLabelName"},
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment",
				Name: "pod-owner",
			}},
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
		},
	}
	return pod
}

func setupPendingPod() *corev1.Pod {
	pendingPod := setupBasicPod()
	pendingPod.Spec.NodeName = ""
	pendingPod.Status = corev1.PodStatus{
		Phase: corev1.PodPending,
		Conditions: []corev1.PodCondition{
			{
				Type:    "PodScheduled",
				Status:  "False",
				Reason:  "Unschedulable",
				Message: "0/1 nodes are available: 1 Insufficient memory.",
			},
		},
	}
	return pendingPod
}

func setupTerminatingPod(t *testing.T, deletionTime string, nodeName string) *corev1.Pod {
	terminatingPod := setupBasicPod()
	terminatingPod.Status = corev1.PodStatus{
		Phase: corev1.PodPending,
	}

	terminatingPod.Spec.NodeName = nodeName

	if len(nodeName) == 0 {
		terminatingPod.Status.Conditions = []corev1.PodCondition{
			{
				Type:    "PodScheduled",
				Status:  "False",
				Reason:  "Unschedulable",
				Message: "0/1 nodes are available: 1 Insufficient memory.",
			},
		}
	}

	date, err := time.Parse(time.RFC3339, deletionTime)
	assert.Nil(t, err)

	terminatingPod.DeletionTimestamp = &metav1.Time{
		Time: date,
	}
	return terminatingPod
}

func setupContainerCreatingPod() *corev1.Pod {
	containerCreatingPod := setupBasicPod()
	containerCreatingPod.Status = corev1.PodStatus{
		Phase: corev1.PodPending,
		Conditions: []corev1.PodCondition{
			{
				Type:   "Initialized",
				Status: "True",
			},
			{
				Type:    "Ready",
				Status:  "False",
				Reason:  "ContainersNotReady",
				Message: "containers with unready status: [wavefront-proxy]",
			},
			{
				Type:    "ContainersReady",
				Status:  "False",
				Reason:  "ContainersNotReady",
				Message: "containers with unready status: [wavefront-proxy]",
			},
			{
				Type:   "PodScheduled",
				Status: "True",
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "testContainerName",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "ContainerCreating",
					},
				},
				Ready:   false,
				Image:   "testImage",
				ImageID: "",
			},
		},
	}
	return containerCreatingPod
}

func setupCompletedPod() *corev1.Pod {
	completedPod := setupBasicPod()
	completedPod.Status = corev1.PodStatus{
		Phase: corev1.PodSucceeded,
		Conditions: []corev1.PodCondition{
			{
				Type:   "Initialized",
				Status: "True",
				Reason: "PodCompleted",
			},
			{
				Type:   "Ready",
				Status: "False",
				Reason: "PodCompleted",
			},
			{
				Type:   "ContainersReady",
				Status: "False",
				Reason: "PodCompleted",
			},
			{
				Type:   "PodScheduled",
				Status: "True",
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "testContainerName",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode:    0,
						Reason:      "Completed",
						ContainerID: "testContainerID",
					},
				},
				Ready:   false,
				Image:   "testImage",
				ImageID: "testImageID",
			},
		},
	}
	return completedPod
}

func setupFailedPod() *corev1.Pod {
	failedPod := setupBasicPod()
	failedPod.Status = corev1.PodStatus{
		Phase: corev1.PodFailed,
		Conditions: []corev1.PodCondition{
			{
				Type:   "Initialized",
				Status: "True",
			},
			{
				Type:    "Ready",
				Status:  "False",
				Reason:  "ContainersNotReady",
				Message: "containers with unready status: [hello], and this message exceeds 255 characters point tag. Maximum allowed length for a combination of a point tag key and value is 254 characters (255 including the = separating key and value). If the value is longer, the point is rejected and logged. Keep the number of distinct time series per metric and host to under 1000.",
			},
			{
				Type:    "ContainersReady",
				Status:  "False",
				Reason:  "ContainersNotReady",
				Message: "containers with unready status: [hello], and this message exceeds 255 characters point tag. Maximum allowed length for a combination of a point tag key and value is 254 characters (255 including the = separating key and value). If the value is longer, the point is rejected and logged. Keep the number of distinct time series per metric and host to under 1000.",
			},
			{
				Type:   "PodScheduled",
				Status: "True",
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "testContainerName",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode:    1,
						Reason:      "Error",
						Message:     "Error message.",
						ContainerID: "testContainerID",
					},
				},
				Ready:   false,
				Image:   "testImage",
				ImageID: "testImageID",
			},
		},
	}
	return failedPod
}

func setupTestTransform() configuration.Transforms {
	return configuration.Transforms{
		Source:  "testSource",
		Prefix:  "testPrefix.",
		Tags:    nil,
		Filters: filter.Config{},
	}
}

func TestPointsForNonRunningPods(t *testing.T) {
	testTransform := setupTestTransform()

	t.Run("test for pending pod", func(t *testing.T) {
		testPod := setupPendingPod()
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 1, len(actualWFPoints))
		point := actualWFPoints[0].(*wf.Point)
		assert.Equal(t, float64(util.POD_PHASE_PENDING), point.Value)
		assert.Equal(t, "pod1", point.Tags()["pod_name"])
		assert.Equal(t, string(corev1.PodPending), point.Tags()["phase"])
		assert.Equal(t, "testLabelName", point.Tags()["label.name"])
		assert.Equal(t, "Unschedulable", point.Tags()["reason"])
		assert.Equal(t, "none", point.Tags()["nodename"])
		assert.Equal(t, "0/1 nodes are available: 1 Insufficient memory.", point.Tags()["message"])
		assert.Equal(t, "some-workload-name", point.Tags()[workloadNameTag])
	})

	t.Run("test for terminating pod without nodename", func(t *testing.T) {
		deletionTime := "2023-06-08T18:35:37Z"
		testPod := setupTerminatingPod(t, deletionTime, "")
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 2, len(actualWFPoints))
		point := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, fmt.Sprintf("%spod.terminating", testTransform.Prefix), point.Metric)
		assert.Equal(t, float64(util.POD_PHASE_PENDING), point.Value)
		assert.Equal(t, deletionTime, point.Tags()["DeletionTimestamp"])
		assert.Equal(t, "pod1", point.Tags()["pod_name"])
		assert.Equal(t, "testLabelName", point.Tags()["label.name"])
		assert.Equal(t, "none", point.Tags()["nodename"])
		assert.Equal(t, "Terminating", point.Tags()["reason"])
	})

	t.Run("test for terminating pod with nodename", func(t *testing.T) {
		deletionTime := "2023-06-08T18:35:37Z"
		testPod := setupTerminatingPod(t, deletionTime, "some-node")
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 2, len(actualWFPoints))
		point := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, fmt.Sprintf("%spod.terminating", testTransform.Prefix), point.Metric)
		assert.Equal(t, float64(util.POD_PHASE_PENDING), point.Value)
		assert.Equal(t, deletionTime, point.Tags()["DeletionTimestamp"])
		assert.Equal(t, "pod1", point.Tags()["pod_name"])
		assert.Equal(t, "testLabelName", point.Tags()["label.name"])
		assert.Equal(t, "some-node", point.Tags()["nodename"])
		assert.Equal(t, "Terminating", point.Tags()["reason"])
	})

	t.Run("test for completed pod", func(t *testing.T) {
		testPod := setupCompletedPod()
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 2, len(actualWFPoints))

		// check for pod metrics
		podPoint := actualWFPoints[0].(*wf.Point)
		assert.Equal(t, float64(util.POD_PHASE_SUCCEEDED), podPoint.Value)
		assert.Equal(t, string(corev1.PodSucceeded), podPoint.Tags()["phase"])
		assert.Equal(t, "", podPoint.Tags()["reason"])
		assert.Equal(t, "node1", podPoint.Tags()["nodename"])
		assert.Equal(t, "some-workload-name", podPoint.Tags()[workloadNameTag])

		// check for container metrics
		containerPoint := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, float64(util.CONTAINER_STATE_TERMINATED), containerPoint.Value)
		assert.Equal(t, "0", containerPoint.Tags()["exit_code"])
		assert.Equal(t, "Completed", containerPoint.Tags()["reason"])
		assert.Equal(t, "terminated", containerPoint.Tags()["status"])
		assert.Equal(t, "some-workload-name", containerPoint.Tags()[workloadNameTag])
	})

	t.Run("test for failed pod", func(t *testing.T) {
		testPod := setupFailedPod()
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 2, len(actualWFPoints))

		// check for pod metrics
		podPoint := actualWFPoints[0].(*wf.Point)
		assert.Equal(t, float64(util.POD_PHASE_FAILED), podPoint.Value)
		assert.Equal(t, string(corev1.PodFailed), podPoint.Tags()["phase"])
		assert.Equal(t, "ContainersNotReady", podPoint.Tags()["reason"])
		assert.Equal(t, 255, len(podPoint.Tags()["message"])+len("message")+len("="))
		assert.Contains(t, podPoint.Tags()["message"], "containers with unready status: [hello]")
		assert.Equal(t, "node1", podPoint.Tags()["nodename"])
		assert.Equal(t, "some-workload-name", podPoint.Tags()[workloadNameTag])

		// check for container metrics
		containerMetric := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, float64(util.CONTAINER_STATE_TERMINATED), containerMetric.Value)
		assert.Equal(t, "1", containerMetric.Tags()["exit_code"])
		assert.Equal(t, "Error", containerMetric.Tags()["reason"])
		assert.Equal(t, "terminated", containerMetric.Tags()["status"])
		assert.Equal(t, "some-workload-name", containerMetric.Tags()[workloadNameTag])
	})

	t.Run("test for container creating pod", func(t *testing.T) {
		testPod := setupContainerCreatingPod()
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.Equal(t, 2, len(actualWFPoints))

		// check for pod metrics
		podMetric := actualWFPoints[0].(*wf.Point)
		assert.Equal(t, float64(util.POD_PHASE_PENDING), podMetric.Value)
		assert.Equal(t, string(corev1.PodPending), podMetric.Tags()["phase"])
		assert.Equal(t, "ContainersNotReady", podMetric.Tags()["reason"])
		assert.Equal(t, "containers with unready status: [wavefront-proxy]", podMetric.Tags()["message"])
		assert.Equal(t, "node1", podMetric.Tags()["nodename"])
		assert.Equal(t, "some-workload-name", podMetric.Tags()[workloadNameTag])

		// check for container metrics
		containerMetric := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, float64(util.CONTAINER_STATE_WAITING), containerMetric.Value)
		assert.Equal(t, "ContainerCreating", containerMetric.Tags()["reason"])
		assert.Equal(t, "waiting", containerMetric.Tags()["status"])
		assert.Equal(t, "some-workload-name", containerMetric.Tags()[workloadNameTag])
	})

	t.Run("metrics should have workload name and type", func(t *testing.T) {
		testPod := setupContainerCreatingPod()
		workloadCache := fakeWorkloadCache{}
		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)

		podMetric := actualWFPoints[0].(*wf.Point)
		assert.Equal(t, "some-workload-name", podMetric.Tags()[workloadNameTag])
		assert.Equal(t, "some-workload-kind", podMetric.Tags()[workloadKindTag])

		containerMetric := actualWFPoints[1].(*wf.Point)
		assert.Equal(t, "some-workload-name", containerMetric.Tags()[workloadNameTag])
		assert.Equal(t, "some-workload-kind", containerMetric.Tags()[workloadKindTag])
	})

	t.Run("workload status metrics should have available and desired tags", func(t *testing.T) {
		testPod := setupBasicPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotEmpty(t, actualWFPoints)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		expectedAvailable := "0"
		expectedDesired := "1"
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, expectedAvailable, podPoint.Tags()[workloadAvailableTag])
		assert.Equal(t, expectedDesired, podPoint.Tags()[workloadDesiredTag])
	})

	t.Run("healthy workload status metrics should not have reason and message tags", func(t *testing.T) {
		testPod := setupCompletedPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotEmpty(t, actualWFPoints)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadReady, podPoint.Value)
		assert.NotContains(t, podPoint.Tags(), workloadFailedReasonTag)
		assert.NotContains(t, podPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("unhealthy workload status metrics should have reason and message tags", func(t *testing.T) {
		testPod := setupFailedPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		workloadCache := fakeWorkloadCache{}
		expectedReason := testPod.Status.ContainerStatuses[0].State.Terminated.Reason
		expectedMessage := testPod.Status.ContainerStatuses[0].State.Terminated.Message

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotEmpty(t, actualWFPoints)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Contains(t, podPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, podPoint.Tags(), workloadFailedMessageTag)
		assert.Equal(t, expectedReason, podPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, podPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("failed pods without owner references should have an unhealthy workload status", func(t *testing.T) {
		testPod := setupFailedPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])
		assert.Contains(t, podPoint.Tags(), workloadFailedReasonTag)
		assert.Contains(t, podPoint.Tags(), workloadFailedMessageTag)
	})

	t.Run("pod image cannot be loaded without owner references should have an unhealthy workload status", func(t *testing.T) {
		testPod := setupContainerCreatingPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		expectedReason := "ImagePullBackOff"
		expectedMessage := "Back-off pulling image busybox123."
		testPod.Status.ContainerStatuses[0].State.Waiting.Reason = expectedReason
		testPod.Status.ContainerStatuses[0].State.Waiting.Message = "Back-off pulling image \"busybox123\"."

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])
		assert.Equal(t, expectedReason, podPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, podPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("pod cannot be scheduled without owner references should have an unhealthy workload status", func(t *testing.T) {
		testPod := setupPendingPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])

		expectedReason := testPod.Status.Conditions[0].Reason
		expectedMessage := testPod.Status.Conditions[0].Message
		assert.Equal(t, expectedReason, podPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, podPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("terminating pending pods without owner references should have an unhealthy workload status", func(t *testing.T) {
		deletionTime := "2023-06-08T18:35:37Z"
		testPod := setupTerminatingPod(t, deletionTime, "")
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])

		expectedReason := "Terminating"
		expectedMessage := testPod.Status.Conditions[0].Message
		assert.Equal(t, expectedReason, podPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, podPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("terminating failed pods without owner references should have an unhealthy workload status", func(t *testing.T) {
		deletionTime := "2023-06-08T18:35:37Z"
		testPod := setupTerminatingPod(t, deletionTime, "")
		testPod.OwnerReferences = nil
		testPod.Status.Phase = corev1.PodFailed
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadNotReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])

		expectedReason := "Terminating"
		expectedMessage := testPod.Status.Conditions[0].Message
		assert.Equal(t, expectedReason, podPoint.Tags()[workloadFailedReasonTag])
		assert.Equal(t, expectedMessage, podPoint.Tags()[workloadFailedMessageTag])
	})

	t.Run("completed pods without owner references should have a healthy workload status", func(t *testing.T) {
		testPod := setupCompletedPod()
		testPod.OwnerReferences = nil
		expectedMetric := testTransform.Prefix + workloadStatusMetric
		expectedWorkloadName := testPod.Name
		workloadCache := fakeWorkloadCache{}

		actualWFPoints := pointsForNonRunningPods(workloadCache)(testPod, testTransform)
		assert.NotNil(t, actualWFPoints)
		assert.Greater(t, len(actualWFPoints), 0)

		podPoint := getWFPointsMap(actualWFPoints)[expectedMetric]
		assert.Equal(t, expectedMetric, podPoint.Metric)
		assert.Equal(t, workloadReady, podPoint.Value)
		assert.Equal(t, expectedWorkloadName, podPoint.Tags()[workloadNameTag])
		assert.Equal(t, workloadKindPod, podPoint.Tags()[workloadKindTag])
	})
}

type fakeWorkloadCache struct{}

func (f fakeWorkloadCache) GetWorkloadForPod(pod *corev1.Pod) (string, string) {
	return "some-workload-name", "some-workload-kind"
}

func (f fakeWorkloadCache) GetWorkloadForPodName(podName, ns string) (string, string, string) {
	return "some-workload", "some-workload-kind", "some-node-name"
}
