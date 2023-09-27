package preprocessor

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	baseYaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type rule struct {
	Rule   string
	Action string
	Key    string `yaml:",omitempty"`
	Tag    string `yaml:",omitempty"`
	Value  string `yaml:",omitempty"`
}

func PreProcess(client crClient.Client, wavefront *wf.Wavefront) error {
	//TODO: Component Refactor - move all of this to components or the wavefront controller if is cross component specific
	wfSpec := &wavefront.Spec
	operator, err := deployment(client, util.OperatorName, wfSpec.Namespace)
	if err != nil {
		return err
	}
	wfSpec.ControllerManagerUID = string(operator.UID)
	wfSpec.ImageRegistry = filepath.Dir(operator.Spec.Template.Spec.Containers[0].Image)

	err = preProcessExperimental(client, wfSpec)
	if err != nil {
		return err
	}

	preProcessDataCollection(client, wfSpec)

	err = preProcessDataExport(client, wfSpec)
	if err != nil {
		return err
	}

	return nil
}

func preProcessDataExport(client crClient.Client, wfSpec *wf.WavefrontSpec) error {
	wfSpec.DataExport.WavefrontProxy.AvailableReplicas = 1
	if wfSpec.DataExport.WavefrontProxy.Enable {
		err := preProcessProxyConfig(client, wfSpec)
		if err != nil {
			return err
		}
	} else if len(wfSpec.DataExport.ExternalWavefrontProxy.Url) != 0 {
		wfSpec.CanExportData = true
		wfSpec.DataCollection.Metrics.ProxyAddress = wfSpec.DataExport.ExternalWavefrontProxy.Url

		if strings.HasPrefix(wfSpec.DataExport.ExternalWavefrontProxy.Url, "http") {
			wfSpec.DataCollection.Logging.ProxyAddress = wfSpec.DataExport.ExternalWavefrontProxy.Url
			// The endpoint for collector requires it to be in the hostname:port format
			wfSpec.DataCollection.Metrics.ProxyAddress = strings.TrimPrefix(wfSpec.DataCollection.Metrics.ProxyAddress, "http://")
		} else {
			// The endpoint for logging requires the http:// prefix
			wfSpec.DataCollection.Logging.ProxyAddress = fmt.Sprintf("http://%s", wfSpec.DataExport.ExternalWavefrontProxy.Url)
		}
	}

	wfSpec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wfSpec.DataExport.WavefrontProxy.Args, "\r", "")
	wfSpec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wfSpec.DataExport.WavefrontProxy.Args, "\n", "")

	return nil
}

func preProcessDataCollection(client crClient.Client, wfSpec *wf.WavefrontSpec) {
	if wfSpec.DataCollection.Metrics.Enable {
		if len(wfSpec.DataCollection.Metrics.CustomConfig) == 0 {
			wfSpec.DataCollection.Metrics.CollectorConfigName = "default-wavefront-collector-config"
		} else {
			wfSpec.DataCollection.Metrics.CollectorConfigName = wfSpec.DataCollection.Metrics.CustomConfig
		}
	} else if wfSpec.Experimental.Insights.Enable {
		wfSpec.DataCollection.Metrics.CollectorConfigName = "k8s-events-only-wavefront-collector-config"
	}
	if shouldEnableEtcdCollection(client, wfSpec) {
		wfSpec.DataCollection.Metrics.ControlPlane.EnableEtcd = true
	}
}

func shouldEnableEtcdCollection(client crClient.Client, wfSpec *wf.WavefrontSpec) bool {
	// never collect etcd if control plane metrics are disabled
	if !wfSpec.DataCollection.Metrics.ControlPlane.Enable {
		return false
	}

	// only enable collection from etcd if the certs are supplied as a Secret
	key := crClient.ObjectKey{
		Namespace: wfSpec.Namespace,
		Name:      "etcd-certs",
	}
	err := client.Get(context.Background(), key, &corev1.Secret{})

	return err == nil
}

func preProcessProxyConfig(client crClient.Client, wfSpec *wf.WavefrontSpec) error {
	deployment, err := deployment(client, util.ProxyName, wfSpec.Namespace)
	if err == nil && deployment.Status.AvailableReplicas > 0 {
		wfSpec.DataExport.WavefrontProxy.AvailableReplicas = int(deployment.Status.AvailableReplicas)
		wfSpec.CanExportData = true
	}
	wfSpec.DataCollection.Metrics.ProxyAddress = fmt.Sprintf("%s:%d", util.ProxyName, wfSpec.DataExport.WavefrontProxy.MetricPort)

	// The endpoint for logging requires the "http://" prefix
	wfSpec.DataCollection.Logging.ProxyAddress = fmt.Sprintf("http://%s:%d", util.ProxyName, wfSpec.DataExport.WavefrontProxy.MetricPort)
	err = parseHttpProxyConfigs(client, wfSpec)
	if err != nil {
		return err
	}

	if wfSpec.Experimental.Autotracing.Enable {
		wfSpec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
		wfSpec.DataExport.WavefrontProxy.OTLP.ResourceAttrsOnMetricsIncluded = true
	}

	wfSpec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts = getEnabledPorts(wfSpec)
	wfSpec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules,
		wfSpec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedGlobalRules,
		err = getUserDefinedRules(client, wfSpec)

	if err != nil {
		return err
	}

	err = processWavefrontSecret(client, wfSpec)
	if err != nil {
		return err
	}

	return nil
}

func preProcessExperimental(client crClient.Client, wfSpec *wf.WavefrontSpec) error {
	if wfSpec.Experimental.Insights.Enable {
		secret, err := findSecret(client, wfSpec.WavefrontTokenSecret, wfSpec.Namespace)
		if err != nil {
			return fmt.Errorf("Invalid Authentication configured for Experimental Kubernetes Events. Secret '%s' was not found", wfSpec.WavefrontTokenSecret)
		}

		if _, ok := secret.Data["k8s-events-endpoint-token"]; !ok {
			return fmt.Errorf("Invalid Authentication configured for Experimental Kubernetes Events. Secret '%s' is missing Data 'k8s-events-endpoint-token'", wfSpec.WavefrontTokenSecret)
		}
		wfSpec.Experimental.Insights.SecretName = wfSpec.WavefrontTokenSecret
	}
	if secret, err := findSecret(client, util.AriaInsightsSecret, wfSpec.Namespace); err == nil {
		if len(secret.Data["k8s-events-endpoint-token"]) == 0 {
			return fmt.Errorf("Invalid Authentication configured for Experimental Kubernetes Events. Secret '%s' is missing Data 'k8s-events-endpoint-token'", secret.Name)
		}
		if len(wfSpec.Experimental.Insights.ExternalEndpointURL) == 0 {
			wfSpec.Experimental.Insights.ExternalEndpointURL = string(secret.Data["k8s-events-endpoint-url"])
		}
		wfSpec.Experimental.Insights.Enable = true
		wfSpec.Experimental.Insights.SecretName = secret.Name
	}
	if wfSpec.Experimental.Insights.Enable && !wfSpec.DataCollection.Metrics.Enable && len(wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU) == 0 {
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources = wf.Resources{
			Requests: wf.Resource{
				CPU:              "200m",
				Memory:           "10Mi",
				EphemeralStorage: "20Mi",
			},
			Limits: wf.Resource{
				CPU:              "2000m",
				Memory:           "512Mi",
				EphemeralStorage: "1Gi",
			},
		}
	}
	if wfSpec.Experimental.Autotracing.Enable {
		daemonset, err := daemonset(client, util.PixieVizierPEMName, wfSpec.Namespace)
		if err == nil && daemonset.Status.NumberReady > 0 {
			wfSpec.Experimental.Autotracing.CanExportAutotracingScripts = true
		}
	}
	return nil
}

func processWavefrontSecret(client crClient.Client, wfSpec *wf.WavefrontSpec) error {
	secret, err := findSecret(client, wfSpec.WavefrontTokenSecret, wfSpec.Namespace)
	if err != nil {
		wfSpec.DataExport.WavefrontProxy.Auth.Type = util.WavefrontTokenAuthType
		return nil
	}

	secretHashBytes, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	wfSpec.DataExport.WavefrontProxy.SecretHash = hashValue(secretHashBytes)

	_, wavefrontTokenAuth := secret.Data["token"]
	_, cspTokenAuth := secret.Data["csp-api-token"]
	_, cspAppID := secret.Data["csp-app-id"]

	checkTotal := checkVal(wavefrontTokenAuth) + checkVal(cspTokenAuth) + checkVal(cspAppID)
	if checkTotal == 0 {
		return fmt.Errorf("Invalid Authentication configured in Secret '%s'. Missing Authentication type. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id", wfSpec.WavefrontTokenSecret)
	}
	if checkTotal > 1 {
		return fmt.Errorf("Invalid Authentication configured in Secret '%s'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id", wfSpec.WavefrontTokenSecret)
	}
	if wavefrontTokenAuth {
		wfSpec.DataExport.WavefrontProxy.Auth.Type = util.WavefrontTokenAuthType
	}
	if cspTokenAuth {
		wfSpec.DataExport.WavefrontProxy.Auth.Type = util.CSPTokenAuthType
	}
	if cspAppID {
		wfSpec.DataExport.WavefrontProxy.Auth.Type = util.CSPAppAuthType
		wfSpec.DataExport.WavefrontProxy.Auth.CSPAppID = string(secret.Data["csp-app-id"])
		wfSpec.DataExport.WavefrontProxy.Auth.CSPOrgID = string(secret.Data["csp-org-id"])
	}
	return nil
}

func checkVal(check bool) int {
	if check {
		return 1
	}
	return 0
}
func getEnabledPorts(wfSpec *wf.WavefrontSpec) string {
	allPorts := []int{wfSpec.DataExport.WavefrontProxy.MetricPort,
		wfSpec.DataExport.WavefrontProxy.DeltaCounterPort,
		wfSpec.DataExport.WavefrontProxy.OTLP.GrpcPort,
		wfSpec.DataExport.WavefrontProxy.OTLP.HttpPort,
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.Port,
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.Port,
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.GrpcPort,
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.HttpPort,
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.Port,
		wfSpec.DataExport.WavefrontProxy.Histogram.Port,
		wfSpec.DataExport.WavefrontProxy.Histogram.MinutePort,
		wfSpec.DataExport.WavefrontProxy.Histogram.HourPort,
		wfSpec.DataExport.WavefrontProxy.Histogram.DayPort,
	}

	var enabledPorts []int
	for _, value := range allPorts {
		if value != 0 {
			enabledPorts = append(enabledPorts, value)
		}
	}

	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(enabledPorts)), ","), "[]")
}

func getUserDefinedRules(client crClient.Client, wfSpec *wf.WavefrontSpec) (portBasedRules, globalRules string, err error) {
	if len(wfSpec.DataExport.WavefrontProxy.Preprocessor) == 0 {
		return "", "", nil
	}

	preprocessorConfigMap, err := findConfigMap(wfSpec.DataExport.WavefrontProxy.Preprocessor, wfSpec.Namespace, client)
	if err != nil {
		return "", "", err
	}

	rules := make(map[string][]rule)
	if err := baseYaml.Unmarshal([]byte(preprocessorConfigMap.Data["rules.yaml"]), &rules); err != nil {
		return "", "", err
	}

	err = validateUserRules(rules, wfSpec.DataExport.WavefrontProxy.Preprocessor)
	if err != nil {
		return "", "", err
	}

	var globalRulesYAML []byte
	if len(rules["global"]) > 0 {
		globalRulesYAML, err = baseYaml.Marshal(rules["global"])
		if err != nil {
			return "", "", err
		}
		delete(rules, "global")
	}

	userDefinedRulesYAML, err := baseYaml.Marshal(rules)
	if err != nil {
		return "", "", err
	}

	return string(userDefinedRulesYAML), string(globalRulesYAML), nil

}

func validateUserRules(userRules map[string][]rule, configMapName string) error {
	for port, rules := range userRules {
		for _, rule := range rules {
			errMsg := "Invalid rule configured in ConfigMap '%s' on port '%s', overriding %s tag '%s' is disallowed"
			if rule.Tag == "cluster" {
				return fmt.Errorf(errMsg, configMapName, port, "metric", "cluster")
			}
			if rule.Tag == "cluster_uuid" {
				return fmt.Errorf(errMsg, configMapName, port, "metric", "cluster_uuid")
			}
			if rule.Key == "cluster" {
				return fmt.Errorf(errMsg, configMapName, port, "span", "cluster")
			}
			if rule.Key == "cluster_uuid" {
				return fmt.Errorf(errMsg, configMapName, port, "span", "cluster_uuid")
			}
		}
	}

	return nil
}

func findConfigMap(name, namespace string, client crClient.Client) (*corev1.ConfigMap, error) {
	objectKey := util.ObjKey(namespace, name)
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.Background(), objectKey, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

func deployment(client crClient.Client, name, ns string) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	err := client.Get(context.Background(), util.ObjKey(ns, name), &deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, err
}

func daemonset(client crClient.Client, name, ns string) (*appsv1.DaemonSet, error) {
	var daemonset appsv1.DaemonSet
	err := client.Get(context.Background(), util.ObjKey(ns, name), &daemonset)
	if err != nil {
		return nil, err
	}

	return &daemonset, err
}

func parseHttpProxyConfigs(client crClient.Client, wavefront *wf.WavefrontSpec) error {
	if len(wavefront.DataExport.WavefrontProxy.HttpProxy.Secret) != 0 {
		httpProxySecret, err := findSecret(client, wavefront.DataExport.WavefrontProxy.HttpProxy.Secret, wavefront.Namespace)
		if err != nil {
			return err
		}
		err = setHttpProxyConfigs(httpProxySecret, wavefront)
		if err != nil {
			return err
		}
	}

	return nil
}

func findSecret(client crClient.Client, name, ns string) (*corev1.Secret, error) {
	secret := crClient.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	httpProxySecret := &corev1.Secret{}
	err := client.Get(context.Background(), secret, httpProxySecret)
	if err != nil {
		return nil, err
	}

	return httpProxySecret, nil
}

func setHttpProxyConfigs(httpProxySecret *corev1.Secret, wavefront *wf.WavefrontSpec) error {
	httpProxySecretData := map[string]string{}
	for k, v := range httpProxySecret.Data {
		httpProxySecretData[k] = string(v)
	}

	rawHttpUrl := httpProxySecretData["http-url"]

	// append http:// if we receive a service in order to correctly parse it -- only the hostname is used, not the scheme
	if !strings.Contains(rawHttpUrl, "http://") && !strings.Contains(rawHttpUrl, "https://") {
		rawHttpUrl = "http://" + rawHttpUrl
	}

	httpUrl, err := url.Parse(rawHttpUrl)
	if err != nil {
		return err
	}
	wavefront.DataExport.WavefrontProxy.HttpProxy.HttpProxyHost = httpUrl.Hostname()
	wavefront.DataExport.WavefrontProxy.HttpProxy.HttpProxyPort = httpUrl.Port()
	wavefront.DataExport.WavefrontProxy.HttpProxy.HttpProxyUser = httpProxySecretData["basic-auth-username"]
	wavefront.DataExport.WavefrontProxy.HttpProxy.HttpProxyPassword = httpProxySecretData["basic-auth-password"]

	if len(httpProxySecretData["tls-root-ca-bundle"]) != 0 {
		wavefront.DataExport.WavefrontProxy.HttpProxy.UseHttpProxyCAcert = true
		wavefront.DataExport.WavefrontProxy.ConfigHash = hashValue(httpProxySecret.Data["tls-root-ca-bundle"])
	}

	return nil
}

func hashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)

	return fmt.Sprintf("%x", h.Sum(nil))
}
