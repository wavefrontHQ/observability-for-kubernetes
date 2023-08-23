package validation

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var legacyComponentsToCheck = map[string]map[string]string{
	"wavefront-collector":      {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"default":                  {"wavefront-proxy": util.Deployment},
	"wavefront":                {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"pks-system":               {"wavefront-collector": util.Deployment, "wavefront-proxy": util.Deployment},
	"tanzu-observability-saas": {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
}

type Result struct {
	error   error
	isError bool
}

func (result Result) Message() string {
	if result.IsValid() {
		return ""
	} else {
		return result.error.Error()
	}
}

func (result Result) IsValid() bool {
	return result.error == nil
}

func (result Result) IsError() bool {
	return result.error != nil && result.isError
}

func (result Result) IsWarning() bool {
	return result.error != nil && !result.isError
}

func NewErrorResult(err error) Result {
	return Result{err, true}
}

func Validate(objClient client.Client, wavefront *wf.Wavefront) Result {
	err := validateEnvironment(objClient, wavefront)
	if err != nil {
		return Result{err, !areAnyComponentsDeployed(objClient, wavefront.Spec.Namespace)}
	}
	err = validateWavefrontSpec(wavefront)
	if err != nil {
		return Result{err, true}
	}
	return Result{}
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

	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		errs = append(errs, validateWavefrontProxyConfig(wavefront)...)
	} else if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) == 0 && (wavefront.Spec.DataCollection.Metrics.Enable || wavefront.Spec.DataCollection.Logging.Enable) {
		errs = append(errs, fmt.Errorf("invalid proxy configuration: either set dataExport.proxy.enable to true or configure dataExport.externalWavefrontProxy.url"))
	}
	if wavefront.Spec.Experimental.AutoTracing.Enable && !wavefront.Spec.DataExport.WavefrontProxy.Enable {
		errs = append(errs, fmt.Errorf("'wavefrontProxy.enable' must be enabled when the 'experimental.autoTracing.enable' is enabled."))
	}
	if wavefront.Spec.DataCollection.Metrics.Enable {
		errs = append(errs, validateResources(&wavefront.Spec.DataCollection.Metrics.NodeCollector.Resources, "spec.dataCollection.metrics.nodeCollector")...)
		errs = append(errs, validateResources(&wavefront.Spec.DataCollection.Metrics.ClusterCollector.Resources, "spec.dataCollection.metrics.clusterCollector")...)
		if wavefront.Spec.Experimental.KubernetesEvents.Enable && len(wavefront.Spec.DataCollection.Metrics.CustomConfig) > 0 {
			errs = append(errs, fmt.Errorf("'metrics.customConfig' must not be set when the 'experimental.kubernetesEvents.enable' is enabled."))
		}

	}
	return utilerrors.NewAggregate(errs)
}

func validateWavefrontProxyConfig(wavefront *wf.Wavefront) []error {
	var errs []error
	if len(wavefront.Spec.WavefrontUrl) == 0 {
		errs = append(errs, fmt.Errorf("'wavefrontUrl' should be set"))
	}
	if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
		errs = append(errs, fmt.Errorf("'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time"))
	}
	errs = append(errs, validateResources(&wavefront.Spec.DataExport.WavefrontProxy.Resources, "spec.dataExport.wavefrontProxy")...)

	return errs
}

func validateResources(resources *wf.Resources, resourcePath string) []error {
	var errs []error

	if compareQuantities(resources.Requests.CPU, resources.Limits.CPU) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.cpu: %s must be less than or equal to cpu limit", resourcePath, resources.Requests.CPU))
	}
	if compareQuantities(resources.Requests.Memory, resources.Limits.Memory) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.memory: %s must be less than or equal to memory limit", resourcePath, resources.Requests.Memory))
	}
	if compareQuantities(resources.Requests.EphemeralStorage, resources.Limits.EphemeralStorage) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.ephemeral-storage: %s must be less than or equal to ephemeral-storage limit", resourcePath, resources.Requests.EphemeralStorage))
	}
	return errs
}

func compareQuantities(request string, limit string) int {
	requestQuantity, _ := resource.ParseQuantity(request)
	limitQuanity, _ := resource.ParseQuantity(limit)
	return requestQuantity.Cmp(limitQuanity)
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
