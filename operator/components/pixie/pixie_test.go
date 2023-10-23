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
	t.Run("valid component", func(t *testing.T) {
		config := validComponentConfig()

		component, err := NewComponent(ComponentDir, config)

		require.NoError(t, err)
		require.NotNil(t, component)
	})

	t.Run("default configuration", func(t *testing.T) {
		component, _ := NewComponent(ComponentDir, validComponentConfig())
		toApply, toDelete, err := component.Resources()

		require.NoError(t, err)
		require.NotEmpty(t, toApply)
		require.Empty(t, toDelete)

		// check all resources for component labels
		test.RequireCommonLabels(t, toApply, "wavefront", "pixie", util.Namespace)

		// cluster name configmap
		configmap, err := test.GetConfigMap("pl-cloud-config", toApply)
		require.NoError(t, err)
		require.Equal(t, component.config.ClusterName, configmap.Data["PL_CLUSTER_NAME"])

		secret, err := test.GetSecret("pl-cluster-secrets", toApply)
		require.NoError(t, err)
		require.Equal(t, component.config.ClusterName, secret.StringData["cluster-name"])
		require.Equal(t, component.config.ClusterUUID, secret.StringData["cluster-id"])
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
		config := Config{Enable: false}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.True(t, result.IsValid())
	})

	t.Run("empty enabled component config is not valid", func(t *testing.T) {
		config := Config{Enable: true}
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
		require.Equal(t, "pixie: missing controller manager uid", result.Message())
	})

	t.Run("empty cluster uuid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterUUID = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing cluster uuid", result.Message())
	})

	t.Run("empty cluster name is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterName = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "pixie: missing cluster name", result.Message())
	})
}

func TestResources(t *testing.T) {
	t.Run("pem resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.PEMResources.Requests.Memory = "500Mi"
		config.PEMResources.Requests.CPU = "50Mi"
		config.PEMResources.Limits.Memory = "1Gi"
		config.PEMResources.Limits.CPU = "100Mi"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources()

		require.NoError(t, err)

		// vizier pem daemon set
		ds, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("table store limits are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.StirlingSources = []string{"a", "b", "c"}

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources()

		require.NoError(t, err)

		// vizier pem daemon set
		ds, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		pemContainer := test.GetContainer("pem", ds.Spec.Template.Spec.Containers)
		test.RequireEnv(t, "a,b,c", "PL_STIRLING_SOURCES", pemContainer)
		test.RequireEnv(t, "1", "PL_TABLE_STORE_HTTP_EVENTS_PERCENT", pemContainer)
		test.RequireEnv(t, "2", "PL_TABLE_STORE_DATA_LIMIT_MB", pemContainer)
	})

	t.Run("http limits are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.MaxHTTPBodyBytes = 128

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources()

		require.NoError(t, err)

		// vizier pem daemon set
		ds, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		pemContainer := test.GetContainer("pem", ds.Spec.Template.Spec.Containers)
		test.RequireEnv(t, "128", "PX_STIRLING_HTTP_BODY_LIMIT_BYTES", pemContainer)
		test.RequireEnv(t, "128", "PL_STIRLING_MAX_BODY_BYTES", pemContainer)
	})
}

func validComponentConfig() Config {
	return Config{
		Enable:               true,
		ControllerManagerUID: "controller-manager-uid",
		ClusterUUID:          "cluster-uuid",
		ClusterName:          wftest.DefaultClusterName,
		PEMResources: wf.Resources{Limits: wf.Resource{
			CPU:    "100Mi",
			Memory: "1Gi",
		}},
		TableStoreLimits: wf.TableStoreLimits{
			TotalMiB:          2,
			HttpEventsPercent: 1,
		},
		StirlingSources: []string{"a", "b", "c"},
	}
}
