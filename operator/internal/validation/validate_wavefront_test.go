package validation

import (
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidate(t *testing.T) {
	t.Run("wf spec and environment are valid", func(t *testing.T) {
		appsV1 := setup()
		require.True(t, ValidateWF(appsV1, &defaultCRSet().Wavefront).IsValid())
		require.False(t, ValidateWF(appsV1, &defaultCRSet().Wavefront).IsError())
	})

	t.Run("wf spec is invalid", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		appsV1 := setup()
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install is running", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		result := ValidateWF(appsV1, &defaultCRSet().Wavefront)
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install is running after operator install", func(t *testing.T) {
		crSet := defaultCRSet()
		legacyCollector := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: "wavefront-collector",
			},
		}
		legacyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront-collector",
			},
		}
		nodeCollector := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.NodeCollectorName,
				Namespace: crSet.Wavefront.Spec.Namespace,
			},
		}
		proxy := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ProxyName,
				Namespace: crSet.Wavefront.Spec.Namespace,
			},
		}
		appsV1 := setup(legacyCollector, legacyDeployment, nodeCollector, proxy)

		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.False(t, result.IsValid())
		require.False(t, result.IsError())
		require.True(t, result.IsWarning())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install if only proxy is enabled", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = false
			w.Spec.DataCollection.Logging.Enable = false
		})
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install if only metrics is enabled", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.ExternalWavefrontProxy.Url = "myproxy.com"
			w.Spec.DataExport.WavefrontProxy.Enable = false
			w.Spec.DataCollection.Logging.Enable = false
		})
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("allow legacy install", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.AllowLegacyInstall = true
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.True(t, result.IsValid())
		require.False(t, result.IsError())
	})

	t.Run("allow legacy install if metrics and proxy are not enabled", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Insights.Enable = true
			w.Spec.Experimental.Insights.IngestionUrl = "my.endpoint.com"
		})
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.True(t, result.IsValid())
		require.False(t, result.IsError())
	})

	t.Run("allow legacy install if only k8s events are enabled", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.NothingEnabledCR()
		result := ValidateWF(appsV1, &crSet.Wavefront)
		require.True(t, result.IsValid())
		require.False(t, result.IsError())
	})

}

func TestValidateWavefrontSpec(t *testing.T) {
	t.Run("Has no validation errors", func(t *testing.T) {
		crSet := defaultCRSet()
		require.Empty(t, validateWavefrontSpec(&crSet.Wavefront))
	})

	t.Run("Validation error when both wavefront proxy and external proxy are defined", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		require.Equal(t, "'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time", validateWavefrontSpec(&crSet.Wavefront).Error())
	})

	t.Run("Validation error wavefront url is required", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.WavefrontUrl = ""
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "'wavefrontUrl' should be set", validationError.Error())
	})

	t.Run("Validation error wavefront url is not required when proxy is not enabled", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.WavefrontUrl = ""
		crSet.Wavefront.Spec.DataCollection.Metrics.Enable = false
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.Nilf(t, validationError, "expected no validation error")
	})

	t.Run("Validation error when auto instrumentation in enabled and an external proxy is configured", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		crSet.Wavefront.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		crSet.Wavefront.Spec.Experimental.Autotracing.Enable = true
		require.Equal(t, "'wavefrontProxy.enable' must be enabled when the 'experimental.autoTracing.enable' is enabled.", validateWavefrontSpec(&crSet.Wavefront).Error())
	})

	t.Run("Test multiple errors", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.Experimental.Autotracing.Enable = true
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		crSet.Wavefront.Spec.DataCollection.Metrics.Enable = true
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		crSet.Wavefront.Spec.DataCollection.Metrics.CustomConfig = "fake custom config"
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "[invalid proxy configuration: either set dataExport.proxy.enable to true or configure dataExport.externalWavefrontProxy.url, 'wavefrontProxy.enable' must be enabled when the 'experimental.autoTracing.enable' is enabled., 'metrics.customConfig' must not be set when the 'experimental.insights.enable' is enabled.]", validationError.Error())
	})

	t.Run("Test No Proxy configuration", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
	})

	t.Run("Test No Proxy configuration with kubernetes events only enabled", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		crSet.Wavefront.Spec.DataCollection.Metrics.Enable = false
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.Nilf(t, validationError, "expected no validation error")
	})

	t.Run("Test custom config with kubernetes events enabled", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataCollection.Metrics.CustomConfig = "my-custom-config"
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
	})

	t.Run("Test No Proxy configuration with kubernetes events and metrics enabled", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		crSet.Wavefront.Spec.DataExport.ExternalWavefrontProxy.Url = ""
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		crSet.Wavefront.Spec.DataCollection.Metrics.Enable = true
		validationError := validateWavefrontSpec(&crSet.Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
	})

	t.Run("Test External Proxy configuration", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.DataExport.WavefrontProxy.Enable = false
		crSet.Wavefront.Spec.DataExport.ExternalWavefrontProxy.Url = "https://external-wf-proxy"
		require.Empty(t, validateWavefrontSpec(&crSet.Wavefront))
	})

	t.Run("must have a valid ClusterSize", func(t *testing.T) {
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.ClusterSize += "_bad"
		err := validateWavefrontSpec(&crSet.Wavefront)
		require.ErrorContains(t, err, "clusterSize must be small, medium, large")
	})
}

func TestValidateEnvironment(t *testing.T) {
	t.Run("No existing collector daemonset", func(t *testing.T) {
		appsV1 := setup()
		require.NoError(t, validateEnvironment(appsV1, &defaultCRSet().Wavefront))
	})

	t.Run("Return error when only proxy deployment found", func(t *testing.T) {
		namespace := "wavefront"
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: namespace,
			},
		}
		appsV1 := setup(proxyDeployment)
		validationError := validateEnvironment(appsV1, &defaultCRSet().Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when legacy manual install found in namespace wavefront-collector", func(t *testing.T) {
		namespace := "wavefront-collector"
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: namespace,
			},
		}
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "default",
			},
		}
		appsV1 := setup(collectorDaemonSet, proxyDeployment)
		validationError := validateEnvironment(appsV1, &defaultCRSet().Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		require.Contains(t, validationError.Error(), "Found legacy Wavefront installation in")
	})

	t.Run("Return error when legacy tkgi install found in namespace tanzu-observability-saas", func(t *testing.T) {
		namespace := "tanzu-observability-saas"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, &defaultCRSet().Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when collector daemonset found in legacy helm install namespace wavefront", func(t *testing.T) {
		namespace := "wavefront"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, &defaultCRSet().Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when collector deployment found in legacy tkgi install namespace pks-system", func(t *testing.T) {
		namespace := "pks-system"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, &defaultCRSet().Wavefront)
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Returns nil when allow legacy install is enabled", func(t *testing.T) {
		namespace := "wavefront"
		appsV1 := legacyEnvironmentSetup(namespace)
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.AllowLegacyInstall = true
		validationError := validateEnvironment(appsV1, &crSet.Wavefront)
		require.Nilf(t, validationError, "expected validation error")
	})

}

func requireValidationMessage(t *testing.T, validationError error, namespace string) {
	require.Equal(t, legacyEnvironmentError(namespace).Error(), validationError.Error())
}

func legacyEnvironmentSetup(namespace string) client.Client {
	return setup(
		&appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: namespace,
			},
		},
		&appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: namespace,
			},
		},
	)
}

func setup(initObjs ...runtime.Object) client.Client {
	return fake.NewClientBuilder().
		WithRuntimeObjects(initObjs...).
		Build()
}

func defaultCRSet() *api.CRSet {
	return &api.CRSet{
		Wavefront: wf.Wavefront{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testNamespace",
				Name:      "wavefront",
			},
			Spec: wf.WavefrontSpec{
				ClusterSize:  wf.ClusterSizeSmall,
				ClusterName:  "testClusterName",
				WavefrontUrl: "testWavefrontUrl",
				DataExport: wf.DataExport{
					WavefrontProxy: wf.WavefrontProxy{
						Enable: true,
					},
				},
				DataCollection: wf.DataCollection{
					Metrics: wf.Metrics{
						Enable: true,
					},
				},
			},
			Status: wf.WavefrontStatus{},
		},
		ResourceCustomizations: rc.ResourceCustomizations{},
	}
}
