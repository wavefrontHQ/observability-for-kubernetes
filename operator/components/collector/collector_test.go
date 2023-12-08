package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewCollectorComponent(t *testing.T) {
	t.Run("valid component", func(t *testing.T) {
		config := minimalComponentConfig()
		t.Log(os.Getwd())
		component, err := NewComponent(ComponentDir, config)
		require.NoError(t, err)
		require.NotNil(t, component)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := minimalComponentConfig()
		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("Validation error when node collector CPU request is greater than CPU limit", func(t *testing.T) {
		config := minimalComponentConfig()
		config.NodeCollectorResources.Requests.CPU = "500m"
		config.NodeCollectorResources.Limits.CPU = "200m"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.False(t, result.IsValid())
		require.Equal(t, "collector: invalid wavefront-node-collector.resources.requests.cpu: 500m must be less than or equal to cpu limit", result.Message())
	})

	t.Run("Validation error when cluster collector memory request is greater than CPU limit", func(t *testing.T) {
		config := minimalComponentConfig()
		config.ClusterCollectorResources.Requests.Memory = "500Mi"
		config.ClusterCollectorResources.Limits.Memory = "200Mi"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.False(t, result.IsValid())
		require.Equal(t, "collector: invalid wavefront-cluster-collector.resources.requests.memory: 500Mi must be less than or equal to memory limit", result.Message())
	})

	t.Run("CPU expressed differently should not be an error", func(t *testing.T) {
		config := minimalComponentConfig()
		config.ClusterCollectorResources.Requests.CPU = "500m"
		config.ClusterCollectorResources.Limits.CPU = "0.5"

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.True(t, result.IsValid())
	})

	t.Run("error when insights enabled but ingestion url is not set", func(t *testing.T) {
		config := minimalComponentConfig()
		config.KubernetesEvents.Enable = true

		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()

		require.False(t, result.IsValid())
	})
}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		component, _ := NewComponent(ComponentDir, minimalComponentConfig())
		toApply, toDelete, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.Equal(t, 4, len(toApply))
		require.Equal(t, 5, len(toDelete))

		var nodeCollectors, clusterCollectors, others []client.Object
		for _, clientObject := range toApply {
			if clientObject.GetObjectKind().GroupVersionKind().Kind == "DaemonSet" {
				nodeCollectors = append(nodeCollectors, clientObject)
			} else if clientObject.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
				clusterCollectors = append(clusterCollectors, clientObject)
			} else {
				others = append(others, clientObject)
			}
		}

		// check all resources for component labels
		test.RequireCommonLabels(t, nodeCollectors, "wavefront", "node-collector", wftest.DefaultNamespace)
		test.RequireCommonLabels(t, clusterCollectors, "wavefront", "cluster-collector", wftest.DefaultNamespace)
		test.RequireCommonLabels(t, others, "wavefront", "collector", wftest.DefaultNamespace)

		serviceAccount, err := test.GetServiceAccount(util.CollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, serviceAccount)

		configMap, err := test.GetConfigMap("default-wavefront-collector-config", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, configMap)

		daemonSet, err := test.GetDaemonSet(util.NodeCollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, daemonSet)

		deployment, err := test.GetDeployment(util.ClusterCollectorName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, deployment)
	})

	// TODO: Component Refactor - move collector wavefront controller test here
}

func minimalComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:                    true,
		MetricsEnable:             true,
		ShouldValidate:            true,
		ControllerManagerUID:      "asdfgh",
		ClusterName:               wftest.DefaultClusterName,
		ClusterUUID:               "uuid",
		DefaultCollectionInterval: "60s",
		ProxyAddress:              fmt.Sprintf("http://%s", wftest.DefaultProxyAddress),
		Namespace:                 wftest.DefaultNamespace,
		ProxyAvailableReplicas:    1,
		ImageRegistry:             wftest.DefaultImageRegistry,
		CollectorVersion:          "1.23",
		ClusterCollectorResources: wf.Resources{
			Requests: wf.Resource{
				CPU:    "100m",
				Memory: "50Mi",
			},
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "50Mi",
			},
		},
		NodeCollectorResources: wf.Resources{
			Requests: wf.Resource{
				CPU:    "100m",
				Memory: "50Mi",
			},
			Limits: wf.Resource{
				CPU:    "100Mi",
				Memory: "50Mi",
			},
		},
		CollectorConfigName: "collector-config-name",
	}
}
