package validation

import (
	"context"
	"fmt"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var legacyComponentsToCheck = map[string]map[string]string{
	"wavefront-collector":      {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"default":                  {"wavefront-proxy": util.Deployment},
	"wavefront":                {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"pks-system":               {"wavefront-collector": util.Deployment, "wavefront-proxy": util.Deployment},
	"tanzu-observability-saas": {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
}

func ValidateWF(objClient client.Client, wavefront *wf.Wavefront) result.Result {
	err := validateEnvironment(objClient, wavefront)
	if err != nil {
		return result.New(areAnyComponentsDeployed(objClient, wavefront.Spec.Namespace), err)
	}
	err = validateWavefrontSpec(wavefront)
	if err != nil {
		return result.NewError(err)
	}
	return result.Valid
}

func validateEnvironment(objClient client.Client, wavefront *wf.Wavefront) error {
	if wavefront.Spec.AllowLegacyInstall ||
		!(wavefront.Spec.DataExport.WavefrontProxy.Enable || wavefront.Spec.DataCollection.Metrics.Enable) {
		return nil
	}
	for namespace, resourceMap := range legacyComponentsToCheck {
		for resourceName, resourceType := range resourceMap {
			if resourceType == util.DaemonSet {
				if daemonSetExists(objClient, util.ObjKey(namespace, resourceName)) {
					return legacyEnvironmentError(namespace)
				}
			}
			if resourceType == util.Deployment {
				if deploymentExists(objClient, util.ObjKey(namespace, resourceName)) {
					return legacyEnvironmentError(namespace)
				}
			}
		}
	}
	return nil
}

func validateWavefrontSpec(wavefront *wf.Wavefront) error {
	var errs []error
	//TODO: Component Refactor - move all non cross component validation to individual components

	if !validClusterSize(wavefront) {
		errs = append(errs, fmt.Errorf("clusterSize must be %s", strings.Join(wf.ClusterSizes, ", ")))
	}

	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		errs = append(errs, validateWavefrontProxyConfig(wavefront)...)
	} else if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) == 0 && (wavefront.Spec.DataCollection.Metrics.Enable || wavefront.Spec.DataCollection.Logging.Enable) {
		errs = append(errs, fmt.Errorf("invalid proxy configuration: either set dataExport.proxy.enable to true or configure dataExport.externalWavefrontProxy.url"))
	}
	if wavefront.Spec.Experimental.Autotracing.Enable && !wavefront.Spec.DataExport.WavefrontProxy.Enable {
		errs = append(errs, fmt.Errorf("'wavefrontProxy.enable' must be enabled when the 'experimental.autoTracing.enable' is enabled."))
	}
	if wavefront.Spec.DataCollection.Metrics.Enable {
		if wavefront.Spec.Experimental.Insights.Enable && len(wavefront.Spec.DataCollection.Metrics.CustomConfig) > 0 {
			errs = append(errs, fmt.Errorf("'metrics.customConfig' must not be set when the 'experimental.insights.enable' is enabled."))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func validClusterSize(wavefront *wf.Wavefront) bool {
	for _, clusterSize := range wf.ClusterSizes {
		if clusterSize == wavefront.Spec.ClusterSize {
			return true
		}
	}
	return false
}

func validateWavefrontProxyConfig(wavefront *wf.Wavefront) []error {
	var errs []error
	if len(wavefront.Spec.WavefrontUrl) == 0 {
		errs = append(errs, fmt.Errorf("'wavefrontUrl' should be set"))
	}
	if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
		errs = append(errs, fmt.Errorf("'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time"))
	}
	return errs
}

func deploymentExists(objClient client.Client, key client.ObjectKey) bool {
	return objClient.Get(context.Background(), key, &appsv1.Deployment{}) == nil
}

func daemonSetExists(objClient client.Client, key client.ObjectKey) bool {
	return objClient.Get(context.Background(), key, &appsv1.DaemonSet{}) == nil
}

func areAnyComponentsDeployed(objClient client.Client, namespace string) bool {
	exists := deploymentExists(objClient, util.ObjKey(namespace, util.ProxyName))
	if exists {
		return exists
	}
	exists = daemonSetExists(objClient, util.ObjKey(namespace, util.NodeCollectorName))
	if exists {
		return exists
	}
	exists = deploymentExists(objClient, util.ObjKey(namespace, util.ClusterCollectorName))
	if exists {
		return exists
	}
	return false
}

func legacyEnvironmentError(namespace string) error {
	return fmt.Errorf("Found legacy Wavefront installation in the '%s' namespace", namespace)
}
