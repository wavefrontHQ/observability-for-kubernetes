package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	kube_api "k8s.io/api/core/v1"
)

func TestGetNodeRole(t *testing.T) {
	t.Run("control-plane if control-plane or master node role is set", func(t *testing.T) {
		fakeNode := kube_api.Node{}

		fakeNode.SetLabels(map[string]string{
			"node-role.kubernetes.io/control-plane": "",
		})
		require.Equal(t, "control-plane", GetNodeRole(&fakeNode))

		fakeNode.SetLabels(map[string]string{
			"node-role.kubernetes.io/master": "",
		})
		require.Equal(t, "control-plane", GetNodeRole(&fakeNode))
	})

	t.Run("worker by default", func(t *testing.T) {
		emptyNode := kube_api.Node{}
		require.Equal(t, "worker", GetNodeRole(&emptyNode))

		fakeNode := kube_api.Node{}
		fakeNode.SetLabels(map[string]string{
			"node-role.kubernetes.io/invalid-node-role": "",
		})
		require.Equal(t, "worker", GetNodeRole(&fakeNode))
	})
}
