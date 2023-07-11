package kstate

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"k8s.io/api/flowcontrol/v1beta2"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	corev1 "k8s.io/api/core/v1"
)

func TestPointsForNode(t *testing.T) {
	t.Run("creates node conditions, node taints, node info metrics", func(t *testing.T) {
		fakeNode := setupFakeNode()

		actualWFPoints := pointsForNode(fakeNode, setupTestTransform())
		require.Len(t, actualWFPoints, 5)
	})

	t.Run("returns nil if type is invalid", func(t *testing.T) {
		require.Nil(t, pointsForNode(corev1.Node{}, setupTestTransform()))
		require.Nil(t, pointsForNode(&corev1.Pod{}, setupTestTransform()))
	})
}

func TestBuildNodeConditions(t *testing.T) {
	t.Run("create node.status.condition metric with appropriate tags", func(t *testing.T) {
		actualNodeConditionsMetrics := buildNodeConditions(setupFakeNode(), setupTestTransform(), 0)
		actualNodeConditionsMetric1 := actualNodeConditionsMetrics[0].(*wf.Point)
		actualNodeConditions1Tags := actualNodeConditionsMetric1.Tags()

		actualNodeConditionsMetric2 := actualNodeConditionsMetrics[1].(*wf.Point)
		actualNodeConditions2Tags := actualNodeConditionsMetric2.Tags()

		require.Equal(t, util.ConditionStatusFloat64(corev1.ConditionStatus(v1beta2.ConditionTrue)), actualNodeConditionsMetric1.Value)
		require.Equal(t, util.ConditionStatusFloat64(corev1.ConditionStatus(v1beta2.ConditionFalse)), actualNodeConditionsMetric2.Value)
		require.Equal(t, "testPrefix.node.status.condition", actualNodeConditionsMetric1.Name())
		require.Equal(t, "fakeNodeName", actualNodeConditions1Tags["nodename"])
		require.Equal(t, "fakeNodeName", actualNodeConditions2Tags["nodename"])
		require.Equal(t, "True", actualNodeConditions1Tags["status"])
		require.Equal(t, "False", actualNodeConditions2Tags["status"])
		require.Equal(t, string(corev1.NodeReady), actualNodeConditions1Tags["condition"])
		require.Equal(t, string(corev1.NodeMemoryPressure), actualNodeConditions2Tags["condition"])
		require.Equal(t, "worker", actualNodeConditions1Tags[metrics.LabelNodeRole.Key])
		require.Equal(t, "worker", actualNodeConditions2Tags[metrics.LabelNodeRole.Key])
	})
}

func TestBuildNodeTaints(t *testing.T) {
	t.Run("create node.spec.taint metric with appropriate tags", func(t *testing.T) {
		actualNodeSpecTaintMetrics := buildNodeTaints(setupFakeNode(), setupTestTransform(), 0)
		actualNodeSpecTaintMetric1 := actualNodeSpecTaintMetrics[0].(*wf.Point)
		actualNodeSpecTaint1Tags := actualNodeSpecTaintMetric1.Tags()

		actualNodeSpecTaintMetric2 := actualNodeSpecTaintMetrics[1].(*wf.Point)
		actualNodeSpecTaint2Tags := actualNodeSpecTaintMetric2.Tags()

		require.Equal(t, 1.0, actualNodeSpecTaintMetric1.Value)
		require.Equal(t, "testPrefix.node.spec.taint", actualNodeSpecTaintMetric1.Name())
		require.Equal(t, "fakeNodeName", actualNodeSpecTaint1Tags["nodename"])
		require.Equal(t, "fakeNodeName", actualNodeSpecTaint2Tags["nodename"])
		require.Equal(t, "fakeTaintKey1", actualNodeSpecTaint1Tags["key"])
		require.Equal(t, "fakeTaintKey2", actualNodeSpecTaint2Tags["key"])
		require.Equal(t, "fakeTaintValue1", actualNodeSpecTaint1Tags["value"])
		require.Equal(t, "fakeTaintValue2", actualNodeSpecTaint2Tags["value"])
		require.Equal(t, string(corev1.TaintEffectNoSchedule), actualNodeSpecTaint1Tags["effect"])
		require.Equal(t, string(corev1.TaintEffectNoExecute), actualNodeSpecTaint2Tags["effect"])
	})
}

func TestBuildNodeInfo(t *testing.T) {
	t.Run("creates node.info metric with appropriate tags", func(t *testing.T) {
		actualNodeInfoMetric := buildNodeInfo(setupFakeNode(), setupTestTransform(), 0).(*wf.Point)
		actualNodeInfoTags := actualNodeInfoMetric.Tags()

		require.Equal(t, 1.0, actualNodeInfoMetric.Value)
		require.Equal(t, "testPrefix.node.info", actualNodeInfoMetric.Name())
		require.Equal(t, "fakeNodeName", actualNodeInfoTags["nodename"])
		require.Equal(t, "fakeKernelVersion", actualNodeInfoTags["kernel_version"])
		require.Equal(t, "fakeOSImage", actualNodeInfoTags["os_image"])
		require.Equal(t, "fakeContainerRuntimeVersion", actualNodeInfoTags["container_runtime_version"])
		require.Equal(t, "fakeKubeletVersion", actualNodeInfoTags["kubelet_version"])
		require.Equal(t, "fakeKubeProxyVersion", actualNodeInfoTags["kubeproxy_version"])
		require.Equal(t, "fakeProviderID", actualNodeInfoTags["provider_id"])
		require.Equal(t, "fakePodCIDR", actualNodeInfoTags["pod_cidr"])
		require.Equal(t, "worker", actualNodeInfoTags["node_role"])
	})

	t.Run("internal_ip tag is set with last InternalIP in node.Status.Addresses", func(t *testing.T) {
		actualNodeInfoMetric := buildNodeInfo(setupFakeNode(), setupTestTransform(), 0).(*wf.Point)
		actualNodeInfoTags := actualNodeInfoMetric.Tags()

		require.Equal(t, "fakeInternalIp", actualNodeInfoTags["internal_ip"])
	})
}

func setupFakeNode() *corev1.Node {
	fakeNode := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionStatus(v1beta2.ConditionTrue),
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionStatus(v1beta2.ConditionFalse),
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KernelVersion:           "fakeKernelVersion",
				OSImage:                 "fakeOSImage",
				ContainerRuntimeVersion: "fakeContainerRuntimeVersion",
				KubeletVersion:          "fakeKubeletVersion",
				KubeProxyVersion:        "fakeKubeProxyVersion",
			},
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "fakeInternalIp",
				},
			},
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{
					Key:    "fakeTaintKey1",
					Value:  "fakeTaintValue1",
					Effect: corev1.TaintEffectNoSchedule,
				},
				{
					Key:    "fakeTaintKey2",
					Value:  "fakeTaintValue2",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
			ProviderID: "fakeProviderID",
			PodCIDR:    "fakePodCIDR",
		},
	}
	fakeNode.SetName("fakeNodeName")
	return fakeNode
}
