/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/version"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/status"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"

	baseYaml "gopkg.in/yaml.v2"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/health"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

const DeployDir = "../deploy/internal"

type KubernetesManager interface {
	ApplyResources(resourceYAMLs []string, exclude func(*unstructured.Unstructured) bool) error
	DeleteResources(resourceYAMLs []string) error
}

// WavefrontReconciler reconciles a Wavefront object
type WavefrontReconciler struct {
	client.Client

	FS                fs.FS
	KubernetesManager KubernetesManager
	MetricConnection  *metric.Connection
	Versions          Versions
	namespace         string
}

// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/finalizers,verbs=update

// Permissions for creating Kubernetes resources from internal files.
// Possible point of confusion: the collector itself watches resources,
// but the operator doesn't need to... yet?
// +kubebuilder:rbac:groups=apps,namespace=observability-system,resources=deployments,verbs=get;create;update;patch;delete;watch;list
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=services,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,namespace=observability-system,resources=daemonsets,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=serviceaccounts,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=configmaps,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

const maxReconcileInterval = 60 * time.Second

func (r *WavefrontReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.namespace = req.Namespace
	wavefront := &wf.Wavefront{}
	err := r.Client.Get(ctx, req.NamespacedName, wavefront)
	if err != nil && !errors.IsNotFound(err) {
		return errorCRTLResult(err)
	}

	if errors.IsNotFound(err) {
		_ = r.readAndDeleteResources()
		return ctrl.Result{}, nil
	}

	err = r.preprocess(wavefront, ctx)
	if err != nil {
		return errorCRTLResult(err)
	}

	validationResult := validation.Validate(r.Client, wavefront)
	if !validationResult.IsError() {
		err = r.readAndCreateResources(wavefront.Spec)
		if err != nil {
			return errorCRTLResult(err)
		}
	} else {
		_ = r.readAndDeleteResources()
	}

	wavefrontStatus, err := r.reportHealthStatus(ctx, wavefront, validationResult)
	if err != nil {
		return errorCRTLResult(err)
	}

	if wavefrontStatus.Status != health.Healthy {
		return ctrl.Result{
			Requeue: true,
		}, nil
	}
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: maxReconcileInterval,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wf.Wavefront{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, maxReconcileInterval),
		}).
		Complete(r)
}

type Versions struct {
	OperatorVersion  string
	CollectorVersion string
	ProxyVersion     string
	LoggingVersion   string
}

func NewWavefrontReconciler(versions Versions, client client.Client) (operator *WavefrontReconciler, err error) {
	return &WavefrontReconciler{
		Versions:          versions,
		Client:            client,
		FS:                os.DirFS(DeployDir),
		KubernetesManager: kubernetes_manager.NewKubernetesManager(client),
		MetricConnection:  metric.NewConnection(metric.WavefrontSenderFactory()),
	}, nil
}

// Read, Create, Update and Delete Resources.
func (r *WavefrontReconciler) readAndCreateResources(spec wf.WavefrontSpec) error {
	controllerManagerUID, err := r.getControllerManagerUID()
	if err != nil {
		return err
	}
	spec.ControllerManagerUID = string(controllerManagerUID)

	toApply, err := r.readAndInterpolateResources(spec, enabledDirs(spec))
	if err != nil {
		return err
	}

	err = r.KubernetesManager.ApplyResources(toApply, filterDisabledAndConfigMap(spec))
	if err != nil {
		return err
	}

	toDelete, err := r.readAndInterpolateResources(spec, disabledDirs(spec))
	if err != nil {
		return err
	}
	err = r.KubernetesManager.DeleteResources(toDelete)
	if err != nil {
		return err
	}
	return nil
}

func (r *WavefrontReconciler) readAndInterpolateResources(spec wf.WavefrontSpec, dirsToInclude []string) ([]string, error) {
	resourceFiles, err := resourceFiles("yaml", dirsToInclude)
	if err != nil {
		return nil, err
	}
	var resources []string
	for _, resourceFile := range resourceFiles {
		templateName := filepath.Base(resourceFile)
		resourceTemplate, err := newTemplate(templateName).ParseFS(r.FS, resourceFile)
		if err != nil {
			return nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, spec)
		if err != nil {
			return nil, err
		}
		resources = append(resources, buffer.String())
	}
	return resources, nil
}

func allDirs() []string {
	return dirList(true, true, true)
}

func enabledDirs(spec wf.WavefrontSpec) []string {
	return dirList(
		spec.DataExport.WavefrontProxy.Enable,
		spec.CanExportData && spec.DataCollection.Metrics.Enable,
		spec.CanExportData && spec.DataCollection.Logging.Enable,
	)
}

func disabledDirs(spec wf.WavefrontSpec) []string {
	return dirList(
		!spec.DataExport.WavefrontProxy.Enable,
		!spec.DataCollection.Metrics.Enable,
		!spec.DataCollection.Logging.Enable,
	)
}

func dirList(proxy, collector, logging bool) []string {
	dirsToInclude := []string{"internal"}
	if proxy {
		dirsToInclude = append(dirsToInclude, "proxy")
	}
	if collector {
		dirsToInclude = append(dirsToInclude, "collector")
	}
	if logging {
		dirsToInclude = append(dirsToInclude, "logging")
	}
	return dirsToInclude
}

func (r *WavefrontReconciler) readAndDeleteResources() error {
	r.MetricConnection.Close()
	specToDelete := wf.WavefrontSpec{
		Namespace: r.namespace,
		DataCollection: wf.DataCollection{
			Metrics: wf.Metrics{
				CollectorVersion: "none",
			},
			Logging: wf.Logging{
				LoggingVersion: "none",
			},
		},
		DataExport: wf.DataExport{
			WavefrontProxy: wf.WavefrontProxy{
				ProxyVersion: "none",
			},
		},
	}
	resources, err := r.readAndInterpolateResources(specToDelete, allDirs())
	if err != nil {
		return err
	}

	err = r.KubernetesManager.DeleteResources(resources)
	if err != nil {
		return err
	}
	return nil
}

func (r *WavefrontReconciler) deployment(name string) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	err := r.Client.Get(context.Background(), util.ObjKey(r.namespace, name), &deployment)
	if err != nil {
		return nil, err
	}
	return &deployment, err
}

func (r *WavefrontReconciler) getControllerManagerUID() (types.UID, error) {
	deployment, err := r.deployment(util.OperatorName)
	if err != nil {
		return "", err
	}
	return deployment.UID, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func resourceFiles(suffix string, dirsToInclude []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(DeployDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && !contains(dirsToInclude, entry.Name()) {
			return fs.SkipDir
		}
		if strings.HasSuffix(path, suffix) {
			filePath := strings.Replace(path, DeployDir+"/", "", 1)
			files = append(files, filePath)
		}

		return nil
	})

	return files, err
}

func newTemplate(resourceFile string) *template.Template {
	fMap := template.FuncMap{
		"toYaml": func(v interface{}) string {
			data, err := baseYaml.Marshal(v)
			if err != nil {
				log.Log.Error(err, "error in toYaml")
				return ""
			}
			return strings.TrimSuffix(string(data), "\n")
		},
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}
	return template.New(resourceFile).Funcs(fMap)
}

func hashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Preprocessing Wavefront Spec
func (r *WavefrontReconciler) preprocess(wavefront *wf.Wavefront, ctx context.Context) error {
	deployment, err := r.deployment(util.OperatorName)
	if err != nil {
		return err
	}

	wavefront.Spec.Namespace = r.namespace

	wavefront.Spec.ImageRegistry = filepath.Dir(deployment.Spec.Template.Spec.Containers[0].Image)

	if wavefront.Spec.DataCollection.Metrics.Enable {
		if len(wavefront.Spec.DataCollection.Metrics.CustomConfig) == 0 {
			wavefront.Spec.DataCollection.Metrics.CollectorConfigName = "default-wavefront-collector-config"
		} else {
			wavefront.Spec.DataCollection.Metrics.CollectorConfigName = wavefront.Spec.DataCollection.Metrics.CustomConfig
		}
	}

	wavefront.Spec.DataExport.WavefrontProxy.AvailableReplicas = 1
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		deployment, err := r.deployment(util.ProxyName)
		if err == nil && deployment.Status.AvailableReplicas > 0 {
			wavefront.Spec.DataExport.WavefrontProxy.AvailableReplicas = int(deployment.Status.AvailableReplicas)
			wavefront.Spec.CanExportData = true
		}
		wavefront.Spec.DataExport.WavefrontProxy.ConfigHash = ""
		wavefront.Spec.DataCollection.Metrics.ProxyAddress = fmt.Sprintf("%s:%d", util.ProxyName, wavefront.Spec.DataExport.WavefrontProxy.MetricPort)

		// The endpoint for logging requires the "http://" prefix
		wavefront.Spec.DataCollection.Logging.ProxyAddress = fmt.Sprintf("http://%s:%d", util.ProxyName, wavefront.Spec.DataExport.WavefrontProxy.MetricPort)

		err = r.parseHttpProxyConfigs(wavefront, ctx)
		if err != nil {
			errInfo := fmt.Sprintf("error setting up http proxy configuration: %s", err.Error())
			log.Log.Info(errInfo)
			return err
		}
	} else if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
		wavefront.Spec.CanExportData = true
		wavefront.Spec.DataCollection.Metrics.ProxyAddress = wavefront.Spec.DataExport.ExternalWavefrontProxy.Url

		if strings.HasPrefix(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url, "http") {
			wavefront.Spec.DataCollection.Logging.ProxyAddress = wavefront.Spec.DataExport.ExternalWavefrontProxy.Url
			// The endpoint for collector requires it to be in the hostname:port format
			wavefront.Spec.DataCollection.Metrics.ProxyAddress = strings.TrimPrefix(wavefront.Spec.DataCollection.Metrics.ProxyAddress, "http://")
		} else {
			// The endpoint for logging requires the http:// prefix
			wavefront.Spec.DataCollection.Logging.ProxyAddress = fmt.Sprintf("http://%s", wavefront.Spec.DataExport.ExternalWavefrontProxy.Url)
		}
	}

	if wavefront.Spec.DataCollection.Logging.Enable {
		configHashBytes, err := json.Marshal(wavefront.Spec.DataCollection.Logging)
		if err != nil {
			return err
		}
		wavefront.Spec.DataCollection.Logging.ConfigHash = hashValue(configHashBytes)
	}

	wavefront.Spec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wavefront.Spec.DataExport.WavefrontProxy.Args, "\r", "")
	wavefront.Spec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wavefront.Spec.DataExport.WavefrontProxy.Args, "\n", "")

	if wavefront.Spec.CanExportData {
		err := r.MetricConnection.Connect(wavefront.Spec.DataCollection.Metrics.ProxyAddress)
		if err != nil {
			return fmt.Errorf("error setting up proxy connection: %s", err.Error())
		}
	}

	wavefront.Spec.DataCollection.Metrics.CollectorVersion = r.Versions.CollectorVersion
	wavefront.Spec.DataExport.WavefrontProxy.ProxyVersion = r.Versions.ProxyVersion
	wavefront.Spec.DataCollection.Logging.LoggingVersion = r.Versions.LoggingVersion

	return nil
}

func (r *WavefrontReconciler) parseHttpProxyConfigs(wavefront *wf.Wavefront, ctx context.Context) error {
	if len(wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.Secret) != 0 {
		httpProxySecret, err := r.findHttpProxySecret(wavefront, ctx)
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

func (r *WavefrontReconciler) findHttpProxySecret(wavefront *wf.Wavefront, ctx context.Context) (*corev1.Secret, error) {
	secret := client.ObjectKey{
		Namespace: r.namespace,
		Name:      wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.Secret,
	}
	httpProxySecret := &corev1.Secret{}
	err := r.Client.Get(ctx, secret, httpProxySecret)
	if err != nil {
		return nil, err
	}
	return httpProxySecret, nil
}

func setHttpProxyConfigs(httpProxySecret *corev1.Secret, wavefront *wf.Wavefront) error {
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
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyHost = httpUrl.Hostname()
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyPort = httpUrl.Port()
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyUser = httpProxySecretData["basic-auth-username"]
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyPassword = httpProxySecretData["basic-auth-password"]

	configHashBytes, err := json.Marshal(wavefront.Spec.DataExport.WavefrontProxy.HttpProxy)
	if err != nil {
		return err
	}

	if len(httpProxySecretData["tls-root-ca-bundle"]) != 0 {
		wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.UseHttpProxyCAcert = true
		configHashBytes = append(configHashBytes, httpProxySecret.Data["tls-root-ca-bundle"]...)
	}

	wavefront.Spec.DataExport.WavefrontProxy.ConfigHash = hashValue(configHashBytes)

	return nil
}

// Reporting Health Status
func (r *WavefrontReconciler) reportHealthStatus(ctx context.Context, wavefront *wf.Wavefront, validationResult validation.Result) (wf.WavefrontStatus, error) {

	wavefrontStatus := health.GenerateWavefrontStatus(r.Client, wavefront)

	if !validationResult.IsValid() {
		wavefrontStatus.Status = health.Unhealthy
		wavefrontStatus.Message = validationResult.Message()
	}

	r.reportMetrics(!validationResult.IsError(), wavefront.Spec.ClusterName, wavefrontStatus)

	if wavefrontStatus.Status != wavefront.Status.Status {
		log.Log.Info(fmt.Sprintf("Wavefront CR wavefrontStatus changed from %s --> %s", wavefront.Status.Status, wavefrontStatus.Status))
	}
	newWavefront := *wavefront
	newWavefront.Status = wavefrontStatus
	return wavefrontStatus, r.Status().Patch(ctx, &newWavefront, client.MergeFrom(wavefront))
}

func (r *WavefrontReconciler) reportMetrics(sendStatusMetrics bool, clusterName string, wavefrontStatus wf.WavefrontStatus) {
	var metrics []metric.Metric

	if sendStatusMetrics {
		statusMetrics, err := status.Metrics(clusterName, r.Versions.OperatorVersion, wavefrontStatus)
		if err != nil {
			log.Log.Error(err, "could not create status metrics")
		} else {
			metrics = append(metrics, statusMetrics...)
		}
	}

	versionMetrics, err := version.Metrics(clusterName, r.Versions.OperatorVersion)
	if err != nil {
		log.Log.Error(err, "could not create version metrics")
	} else {
		metrics = append(metrics, versionMetrics...)
	}

	r.MetricConnection.Send(metrics)
}

func filterDisabledAndConfigMap(wavefrontSpec wf.WavefrontSpec) func(object *unstructured.Unstructured) bool {
	return func(object *unstructured.Unstructured) bool {
		objLabels := object.GetLabels()
		if labelVal := objLabels["app.kubernetes.io/component"]; labelVal == "collector" && object.GetKind() == "ConfigMap" && wavefrontSpec.DataCollection.Metrics.CollectorConfigName != object.GetName() {
			return true
		}
		return false
	}
}

func errorCRTLResult(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}
