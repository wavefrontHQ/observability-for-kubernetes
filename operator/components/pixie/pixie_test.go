package pixie

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewPixieComponent(t *testing.T) {
	t.Run("create config hash", func(t *testing.T) {
		config := validComponentConfig()
		t.Log(os.Getwd())

		component, err := NewComponent(config, ComponentDir)

		require.NoError(t, err)
		require.NotNil(t, component)
	})
}

func TestProcessAndValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := validComponentConfig()
		component, _ := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("empty disabled component config is valid", func(t *testing.T) {
		config := ComponentConfig{Enable: false}
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.True(t, result.IsValid())
	})

	t.Run("empty enabled component config is not valid", func(t *testing.T) {
		config := ComponentConfig{Enable: true}
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
	})

	t.Run("empty controller manager uid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ControllerManagerUID = ""
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing controller manager uid", result.Message())
	})

	t.Run("empty cluster uuid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterUUID = ""
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing cluster uuid", result.Message())
	})

	t.Run("empty cluster name is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterName = ""
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing cluster name", result.Message())
	})

	t.Run("empty namespace is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.Namespace = ""
		component, err := NewComponent(config, ComponentDir)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing namespace", result.Message())
	})

	// TODO add in image registry
	//t.Run("empty image registry is not valid", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.ImageRegistry = ""
	//	component, err := NewComponent(config, ComponentDir)
	//	result := component.Validate()
	//	require.NoError(t, err)
	//	require.False(t, result.IsValid())
	//	require.Equal(t, "logging: missing image registry", result.Message())
	//})
}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		component, _ := NewComponent(validComponentConfig(), ComponentDir)
		toApply, toDelete, err := component.Resources()

		require.NoError(t, err)
		require.NotEmpty(t, toApply)
		require.Empty(t, toDelete)

		// daemonSet
		ds, err := test.GetAppliedDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)

		require.Equal(t, util.PixieVizierPEMName, ds.Spec.Template.GetLabels()["name"])
		require.Equal(t, "wavefront", ds.GetLabels()["app.kubernetes.io/name"])
		require.Equal(t, "pixie", ds.GetLabels()["app.kubernetes.io/component"])
		// TODO should template have these automatically created?
		//require.Equal(t, "wavefront", ds.Spec.Template.GetLabels()["app.kubernetes.io/name"])
		//require.Equal(t, "pixie", ds.Spec.Template.GetLabels()["app.kubernetes.io/component"])
		require.Equal(t, wftest.DefaultNamespace, ds.Namespace)
		//require.Equal(t, "1", ds.Spec.Template.GetAnnotations()["proxy-available-replicas"])
		//require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		//require.Equal(t, wftest.DefaultImageRegistry+"/kubernetes-operator-fluentbit:"+component.config.LoggingVersion, ds.Spec.Template.Spec.Containers[0].Image)
		//require.Equal(t, component.config.ClusterName, ds.Spec.Template.Spec.Containers[0].Env[1].Value)
		//
		//// configMap
		//configMap, err := test.GetAppliedConfigMap("wavefront-logging-config", toApply)
		//require.NoError(t, err)
		//
		//require.Equal(t, "wavefront", configMap.GetLabels()["app.kubernetes.io/name"])
		//require.Equal(t, "logging", configMap.GetLabels()["app.kubernetes.io/component"])
		//require.Equal(t, wftest.DefaultNamespace, configMap.Namespace)
		//
		//fluentBitConfig := fluentBitConfiguration(toApply)
		//require.NoError(t, err)
		//require.Contains(t, fluentBitConfig, fmt.Sprintf("Proxy             %s", component.config.ProxyAddress))
	})

	//t.Run("k8s resources are set correctly", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.Resources.Requests.CPU = "200m"
	//	config.Resources.Requests.Memory = "10Mi"
	//	config.Resources.Limits.Memory = "256Mi"
	//	component, _ := NewComponent(config, ComponentDir)
	//	toApply, _, err := component.Resources()
	//
	//	require.NoError(t, err)
	//	require.NotEmpty(t, toApply)
	//	ds, err := test.GetAppliedDaemonSet("wavefront-logging", toApply)
	//	require.NoError(t, err)
	//	require.Equal(t, "10Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
	//	require.Equal(t, "200m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
	//	require.Equal(t, "256Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	//})
	//
	//t.Run("tag allow list is set correctly", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.TagAllowList = map[string][]string{"namespace_name": {"kube-sys", "wavefront"}, "pod_name": {"pet-clinic"}}
	//	component, _ := NewComponent(config, ComponentDir)
	//	toApply, _, err := component.Resources()
	//
	//	require.NoError(t, err)
	//	require.NotEmpty(t, toApply)
	//
	//	fluentBitConfig := fluentBitConfiguration(toApply)
	//	require.NoError(t, err)
	//	require.Contains(t, fluentBitConfig, "Regex  namespace_name ^kube-sys$|^wavefront$")
	//	require.Contains(t, fluentBitConfig, "Regex  pod_name ^pet-clinic$")
	//})
	//
	//t.Run("tags are set correctly", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.Tags = map[string]string{"key1": "value1", "key2": "value2"}
	//	component, _ := NewComponent(config, ComponentDir)
	//	toApply, _, err := component.Resources()
	//
	//	require.NoError(t, err)
	//	require.NotEmpty(t, toApply)
	//
	//	fluentBitConfig := fluentBitConfiguration(toApply)
	//	require.NoError(t, err)
	//	require.Contains(t, fluentBitConfig, "Record          key1 value1")
	//	require.Contains(t, fluentBitConfig, "Record          key2 value2")
	//})
	//
	//t.Run("external wavefront proxy url with http specified in URL is set correctly", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.ProxyAddress = "http://my-proxy:8888"
	//	component, _ := NewComponent(config, ComponentDir)
	//	toApply, _, err := component.Resources()
	//
	//	require.NoError(t, err)
	//	require.NotEmpty(t, toApply)
	//
	//	fluentBitConfig := fluentBitConfiguration(toApply)
	//	require.NoError(t, err)
	//	require.Contains(t, fluentBitConfig, "Proxy             http://my-proxy:8888")
	//})

	//TODO - Component Refactor - move over most of the component level tests from wavefront_controller_test#TestReconcileLogging
}

func validComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:                   true,
		ControllerManagerUID:     "controller-manager-uid",
		ClusterUUID:              "cluster-uuid",
		ClusterName:              wftest.DefaultClusterName,
		Namespace:                wftest.DefaultNamespace,
		EnableOpAppsOptimization: true,
		PemResources:             wf.Resources{},
	}
}
