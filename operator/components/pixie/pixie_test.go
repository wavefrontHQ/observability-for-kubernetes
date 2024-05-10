package pixie

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
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
		toApply, toDelete, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.NotEmpty(t, toApply)
		require.Empty(t, toDelete)

		// check all resources for component labels
		test.RequireCommonLabels(t, toApply, "wavefront", "pixie", wftest.DefaultNamespace)

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
		config := Config{ShouldValidate: true}
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
	t.Run("TLSCertsSecretExists set to true", func(t *testing.T) {
		config := validComponentConfig()
		config.TLSCertsSecretExists = true
		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		for _, deploymentName := range []string{util.PixieKelvinName, util.PixieVizierQueryBrokerName} {
			deployment, err := test.GetDeployment(deploymentName, toApply)
			require.NoError(t, err)
			require.Equal(t, "true", deployment.GetAnnotations()["wavefront.com/conditionally-provision"])
		}

		for _, statefulSetName := range []string{util.PixieVizierMetadataName, util.PixieNatsName} {
			statefulSet, err := test.GetStatefulSet(statefulSetName, toApply)
			require.NoError(t, err)
			require.Equal(t, "true", statefulSet.GetAnnotations()["wavefront.com/conditionally-provision"])
		}

		daemonSet, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		require.Equal(t, "true", daemonSet.GetAnnotations()["wavefront.com/conditionally-provision"])
	})

	t.Run("TLSCertsSecretExists set to false", func(t *testing.T) {
		config := validComponentConfig()
		config.TLSCertsSecretExists = false
		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		for _, deploymentName := range []string{util.PixieKelvinName, util.PixieVizierQueryBrokerName} {
			_, err := test.GetDeployment(deploymentName, toApply)
			require.ErrorContains(t, err, "not found")
		}

		for _, statefulSetName := range []string{util.PixieVizierMetadataName, util.PixieNatsName} {
			_, err := test.GetStatefulSet(statefulSetName, toApply)
			require.ErrorContains(t, err, "not found")
		}

		_, err = test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("pem resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.PEMResources.Requests.Memory = "500Mi"
		config.PEMResources.Requests.CPU = "50m"
		config.PEMResources.Limits.Memory = "1Gi"
		config.PEMResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		// vizier pem daemon set
		res, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("table store limits are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.StirlingSources = []string{"a", "b", "c"}

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		// vizier pem daemon set
		res, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		pemContainer := test.GetContainer("pem", res.Spec.Template.Spec.Containers)
		test.RequireEnv(t, "a,b,c", "PL_STIRLING_SOURCES", pemContainer)
		test.RequireEnv(t, "1", "PL_TABLE_STORE_HTTP_EVENTS_PERCENT", pemContainer)
		test.RequireEnv(t, "2", "PL_TABLE_STORE_DATA_LIMIT_MB", pemContainer)
	})

	t.Run("http limits are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.MaxHTTPBodyBytes = 128

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		// vizier pem daemon set
		res, err := test.GetDaemonSet(util.PixieVizierPEMName, toApply)
		require.NoError(t, err)
		pemContainer := test.GetContainer("pem", res.Spec.Template.Spec.Containers)
		test.RequireEnv(t, "128", "PX_STIRLING_HTTP_BODY_LIMIT_BYTES", pemContainer)
		test.RequireEnv(t, "128", "PL_STIRLING_MAX_BODY_BYTES", pemContainer)
	})

	t.Run("Query Broker resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.QueryBrokerResources.Requests.Memory = "500Mi"
		config.QueryBrokerResources.Requests.CPU = "50m"
		config.QueryBrokerResources.Limits.Memory = "1Gi"
		config.QueryBrokerResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		res, err := test.GetDeployment(util.PixieVizierQueryBrokerName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("NATS resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.NATSResources.Requests.Memory = "500Mi"
		config.NATSResources.Requests.CPU = "50m"
		config.NATSResources.Limits.Memory = "1Gi"
		config.NATSResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		res, err := test.GetStatefulSet(util.PixieNatsName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("Metadata resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.MetadataResources.Requests.Memory = "500Mi"
		config.MetadataResources.Requests.CPU = "50m"
		config.MetadataResources.Limits.Memory = "1Gi"
		config.MetadataResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		res, err := test.GetStatefulSet(util.PixieVizierMetadataName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("Kelvin resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.KelvinResources.Requests.Memory = "500Mi"
		config.KelvinResources.Requests.CPU = "50m"
		config.KelvinResources.Limits.Memory = "1Gi"
		config.KelvinResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		res, err := test.GetDeployment(util.PixieKelvinName, toApply)
		require.NoError(t, err)
		require.Equal(t, "500Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "1Gi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})

	t.Run("Job Cert Provisioner resources are configurable", func(t *testing.T) {
		config := validComponentConfig()
		config.CertProvisionerJobResources.Requests.Memory = "50Mi"
		config.CertProvisionerJobResources.Requests.CPU = "50m"
		config.CertProvisionerJobResources.Limits.Memory = "100Mi"
		config.CertProvisionerJobResources.Limits.CPU = "100m"

		component, _ := NewComponent(ComponentDir, config)
		toApply, _, err := component.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)

		res, err := test.GetJob(util.PixieCertProvisionerJobName, toApply)
		require.NoError(t, err)
		require.Equal(t, "50Mi", res.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "50m", res.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "100Mi", res.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "100m", res.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	})
}

func validComponentConfig() Config {
	return Config{
		Enable:               true,
		ShouldValidate:       true,
		TLSCertsSecretExists: true,
		ControllerManagerUID: "controller-manager-uid",
		ClusterUUID:          "cluster-uuid",
		ClusterName:          wftest.DefaultClusterName,
		Namespace:            wftest.DefaultNamespace,
		PEMResources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "1Gi",
			},
			Requests: wf.Resource{
				CPU:    "50m",
				Memory: "500Mi",
			},
		},
		QueryBrokerResources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "1Gi",
			},
			Requests: wf.Resource{
				CPU:    "50m",
				Memory: "500Mi",
			},
		},
		MetadataResources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "1Gi",
			},
			Requests: wf.Resource{
				CPU:    "50m",
				Memory: "500Mi",
			},
		},
		KelvinResources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "1Gi",
			},
			Requests: wf.Resource{
				CPU:    "50m",
				Memory: "500Mi",
			},
		},
		NATSResources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "1Gi",
			},
			Requests: wf.Resource{
				CPU:    "50m",
				Memory: "500Mi",
			},
		},
		TableStoreLimits: wf.TableStoreLimits{
			TotalMiB:          2,
			HttpEventsPercent: 1,
		},
		StirlingSources: []string{"a", "b", "c"},
	}
}
