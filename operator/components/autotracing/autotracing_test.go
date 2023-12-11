package autotracing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewPixieComponent(t *testing.T) {
	t.Run("valid component", func(t *testing.T) {
		config := validComponentConfig()
		t.Log(os.Getwd())

		component, err := NewComponent(ComponentDir, config)

		require.NoError(t, err)
		require.NotNil(t, component)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := validComponentConfig()
		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("empty disabled component config is valid", func(t *testing.T) {
		config := ComponentConfig{Enable: false}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.True(t, result.IsValid())
	})

	t.Run("empty enabled component config is not valid", func(t *testing.T) {
		config := ComponentConfig{ShouldValidate: true}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
	})

	t.Run("empty controller manager uid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ControllerManagerUID = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "autotracing: missing controller manager uid", result.Message())
	})

	t.Run("empty namespace is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.Namespace = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "autotracing: missing namespace", result.Message())
	})
}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		component, _ := NewComponent(ComponentDir, validComponentConfig())
		toApply, toDelete, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.Equal(t, 3, len(toApply))
		require.Empty(t, toDelete)

		// check all resources for component labels
		test.RequireCommonLabels(t, toApply, "wavefront", "autotracing", component.config.Namespace)

		clusterSpansConfigMap, err := test.GetConfigMap("wavefront-cluster-spans-script", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, clusterSpansConfigMap)
		checkConfigMapNamespace(t, clusterSpansConfigMap)

		egressSpansConfigMap, err := test.GetConfigMap("wavefront-egress-spans-script", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, egressSpansConfigMap)
		checkConfigMapNamespace(t, egressSpansConfigMap)

		ingressSpansConfigMap, err := test.GetConfigMap("wavefront-ingress-spans-script", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, ingressSpansConfigMap)
		checkConfigMapNamespace(t, ingressSpansConfigMap)
	})

	t.Run("can change namespace", func(t *testing.T) {
		config := validComponentConfig()
		config.Namespace = wftest.DefaultNamespace
		component, _ := NewComponent(ComponentDir, config)
		toApply, toDelete, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.Equal(t, 3, len(toApply))
		require.Empty(t, toDelete)

		// check all resources for component labels
		test.RequireCommonLabels(t, toApply, "wavefront", "autotracing", wftest.DefaultNamespace)

		checkConfigMapNamespaceChanges(t, "wavefront-cluster-spans-script", toApply)
		checkConfigMapNamespaceChanges(t, "wavefront-egress-spans-script", toApply)
		checkConfigMapNamespaceChanges(t, "wavefront-ingress-spans-script", toApply)
	})
}

func checkConfigMapNamespace(t *testing.T, configMap v1.ConfigMap) {
	pxlScript := configMap.Data["script.pxl"]
	configs := configMap.Data["configs.yaml"]
	require.Contains(t, configs, fmt.Sprintf(" wavefront-proxy.%s.svc.cluster.local:4317", util.Namespace))
	require.Contains(t, pxlScript, util.Namespace)
}

func checkConfigMapNamespaceChanges(t *testing.T, metadataName string, toApply []client.Object) {
	configmap, err := test.GetConfigMap(metadataName, toApply)
	require.NoError(t, err)
	require.Equal(t, wftest.DefaultNamespace, configmap.Namespace)

	pxlScript := configmap.Data["script.pxl"]
	require.Contains(t, pxlScript, wftest.DefaultNamespace)
	require.NotContains(t, pxlScript, util.Namespace)

	configs := configmap.Data["configs.yaml"]
	require.Contains(t, configs, fmt.Sprintf(" wavefront-proxy.%s.svc.cluster.local:4317", wftest.DefaultNamespace))
}

func validComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:               true,
		ShouldValidate:       true,
		ControllerManagerUID: "controller-manager-uid",
		Namespace:            util.Namespace,
	}
}
