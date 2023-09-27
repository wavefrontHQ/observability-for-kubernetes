package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewCollectorComponent(t *testing.T) {
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

	t.Run("Validation error when node collector CPU request is greater than CPU limit", func(t *testing.T) {
		config := validComponentConfig()
		config.NodeCollectorResources.Requests.CPU = "500m"
		config.NodeCollectorResources.Limits.CPU = "200m"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.False(t, result.IsValid())
		require.Equal(t, "collector: invalid wavefront-node-collector.resources.requests.cpu: 500m must be less than or equal to cpu limit", result.Message())
	})

	t.Run("Does not validates node collector resources when metrics is not enabled", func(t *testing.T) {
		config := validComponentConfig()
		config.MetricsEnable = false
		config.NodeCollectorResources = wf.Resources{}

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.True(t, result.IsValid())
	})

	t.Run("Validation error when cluster collector memory request is greater than CPU limit", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterCollectorResources.Requests.Memory = "500Mi"
		config.ClusterCollectorResources.Limits.Memory = "200Mi"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.False(t, result.IsValid())
		require.Equal(t, "collector: invalid wavefront-cluster-collector.resources.requests.memory: 500Mi must be less than or equal to memory limit", result.Message())
	})

	t.Run("CPU expressed differently should not be an error", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterCollectorResources.Requests.CPU = "500m"
		config.ClusterCollectorResources.Limits.CPU = "0.5"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.True(t, result.IsValid())
	})
}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		component, _ := NewComponent(ComponentDir, validComponentConfig())
		toApply, toDelete, err := component.Resources()

		require.NoError(t, err)
		require.Equal(t, 4, len(toApply))
		require.Equal(t, 5, len(toDelete))

		// check all resources for component labels
		var nodeCollector, clusterCollector, others []client.Object
		for _, clientObject := range toApply {
			if clientObject.GetObjectKind().GroupVersionKind().Kind == "DaemonSet" {
				nodeCollector = append(nodeCollector, clientObject)
			} else if clientObject.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
				clusterCollector = append(clusterCollector, clientObject)
			} else {
				others = append(others, clientObject)
			}
		}
		test.RequireCommonLabels(t, nodeCollector, "wavefront", "node-collector", wftest.DefaultNamespace)
		test.RequireCommonLabels(t, clusterCollector, "wavefront", "cluster-collector", wftest.DefaultNamespace)
		test.RequireCommonLabels(t, others, "wavefront", "collector", wftest.DefaultNamespace)

		serviceAccount, err := test.GetAppliedServiceAccount(util.CollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, serviceAccount)

		configMap, err := test.GetAppliedConfigMap("default-wavefront-collector-config", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, configMap)

		daemonSet, err := test.GetAppliedDaemonSet(util.NodeCollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, daemonSet)

		deployment, err := test.GetAppliedDeployment(util.ClusterCollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, deployment)
	})

	// TODO: Component Refactor - move collector wavefront controller test here

}

func validComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:                    true,
		MetricsEnable:             true,
		ControllerManagerUID:      "asdfgh",
		ClusterName:               wftest.DefaultClusterName,
		ClusterUUID:               "uuid",
		DefaultCollectionInterval: "60s",
		ProxyAddress:              fmt.Sprintf("http://%s", wftest.DefaultProxyAddress),
		Namespace:                 wftest.DefaultNamespace,
		ProxyAvailableReplicas:    1,
		ImageRegistry:             wftest.DefaultImageRegistry,
		CollectorVersion:          "1.23",
		ClusterCollectorResources: wf.Resources{Limits: wf.Resource{
			CPU:    "100Mi",
			Memory: "50Mi",
		}},
		NodeCollectorResources: wf.Resources{Limits: wf.Resource{
			CPU:    "100Mi",
			Memory: "50Mi",
		}},
		CollectorConfigName: "collector-config-name",
	}
}
