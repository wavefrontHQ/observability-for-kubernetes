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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/preprocessor"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/version"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/status"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"

	baseYaml "gopkg.in/yaml.v2"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/health"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

const DeployDir = "../deploy/internal"

type KubernetesManager interface {
	ApplyResources(resourceYAMLs []client.Object) error
	DeleteResources(resourceYAMLs []client.Object) error
}

// WavefrontReconciler reconciles a Wavefront object
type WavefrontReconciler struct {
	client.Client

	FS                fs.FS
	KubernetesManager KubernetesManager
	DiscoveryClient   discovery.ServerGroupsInterface
	MetricConnection  *metric.Connection
	Versions          Versions
	namespace         string
	ClusterUUID       string

	components map[components.Component]bool
}

// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/finalizers,verbs=update

// Permissions for creating Kubernetes resources from internal files.
// Possible point of confusion: the collector itself watches resources,
// but the operator doesn't need to... yet?
// +kubebuilder:rbac:groups=apps,namespace=observability-system,resources=deployments,verbs=get;create;update;patch;delete;watch;list;
// +kubebuilder:rbac:groups=apps,namespace=observability-system,resources=daemonsets,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups=apps,namespace=observability-system,resources=statefulsets,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=services,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=serviceaccounts,verbs=get;create;update;patch;delete;watch;list;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=configmaps,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups="",namespace=observability-system,resources=persistentvolumeclaims,verbs=get;create;update;patch;delete;
// +kubebuilder:rbac:groups="",namespace="",resources=namespaces,verbs=get;list;watch;
// +kubebuilder:rbac:groups=batch,namespace=observability-system,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=policy,namespace=observability-system,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete;

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
		// create all components readAndDeleteResources
		_ = r.readAndDeleteResources()
		return ctrl.Result{}, nil
	}

	var validationResult validation.Result
	err = r.preprocess(wavefront, ctx)
	if err != nil {
		validationResult = validation.NewErrorResult(err)
	} else {
		validationResult = r.validate(wavefront)
	}

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

func (r *WavefrontReconciler) validate(wavefront *wf.Wavefront) validation.Result {
	var result validation.Result
	for component, enable := range r.components {
		if enable {
			result = component.Validate()
			if result.IsError() {
				break
			}
		}
	}

	if result.IsError() {
		return result
	}
	//TODO - Component Refactor - move all non cross component validation to components
	return validation.Validate(r.Client, wavefront)
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

func NewWavefrontReconciler(versions Versions, client client.Client, discoveryClient discovery.ServerGroupsInterface, clusterUUID string) (operator *WavefrontReconciler, err error) {
	return &WavefrontReconciler{
		Versions:          versions,
		Client:            client,
		FS:                os.DirFS(DeployDir),
		KubernetesManager: kubernetes_manager.NewKubernetesManager(client),
		DiscoveryClient:   discoveryClient,
		MetricConnection:  metric.NewConnection(metric.WavefrontSenderFactory()),
		ClusterUUID:       clusterUUID,
	}, nil
}

// Read, Create, Update and Delete Resources.
func (r *WavefrontReconciler) readAndCreateResources(spec wf.WavefrontSpec) error {
	toApply, toDelete, err := r.readAndInterpolateResources(spec)
	if err != nil {
		return err
	}

	err = r.KubernetesManager.ApplyResources(toApply)
	if err != nil {
		return err
	}

	err = r.KubernetesManager.DeleteResources(toDelete)
	if err != nil {
		return err
	}

	return nil
}

func (r *WavefrontReconciler) readAndInterpolateResources(spec wf.WavefrontSpec) ([]client.Object, []client.Object, error) {
	files, err := resourceFiles("yaml", spec)
	if err != nil {
		return nil, nil, err
	}

	var resourcesToApply, resourcesToDelete []client.Object
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for resourceFile, shouldApply := range files {
		templateName := filepath.Base(resourceFile)
		resourceTemplate, err := newTemplate(templateName).ParseFS(r.FS, resourceFile)
		if err != nil {
			return nil, nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, spec)
		if err != nil {
			return nil, nil, err
		}

		resourceYAML := buffer.String()
		resource := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(resourceYAML), nil, resource)
		if err != nil {
			return nil, nil, err
		}

		labels := resource.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels["app.kubernetes.io/name"] = "wavefront"
		if labels["app.kubernetes.io/component"] == "" {
			labels["app.kubernetes.io/component"] = filepath.Base(filepath.Dir(resourceFile))
		}
		resource.SetLabels(labels)

		resource.SetOwnerReferences([]v1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "wavefront-controller-manager",
			UID:        types.UID(spec.ControllerManagerUID),
		}})

		if shouldApply && resource.GetAnnotations()["wavefront.com/conditionally-provision"] != "false" {
			resourcesToApply = append(resourcesToApply, resource)
		} else {
			resourcesToDelete = append(resourcesToDelete, resource)
		}
	}
	//TODO: Component Refactor - remove above templating code once everything has been moved to components ^^^

	for component, enable := range r.components {
		toApply, toDelete, _ := component.Resources()
		resourcesToDelete = append(resourcesToDelete, toDelete...)
		if enable {
			resourcesToApply = append(resourcesToApply, toApply...)
		} else {
			resourcesToDelete = append(resourcesToDelete, toApply...)
		}
	}
	return resourcesToApply, resourcesToDelete, nil
}

func enabledDirs(spec wf.WavefrontSpec) []string {
	//TODO: Component Refactor - this should all be moved to component factor / components
	dirsToInclude := []string{"internal"}
	if spec.DataExport.WavefrontProxy.Enable {
		dirsToInclude = append(dirsToInclude, "proxy")
	}

	if (spec.CanExportData && spec.DataCollection.Metrics.Enable) || spec.Experimental.KubernetesEvents.Enable {
		dirsToInclude = append(dirsToInclude, "collector")
	}

	if spec.Experimental.Autotracing.Enable || spec.Experimental.Hub.Pixie.Enable {
		dirsToInclude = append(dirsToInclude, "pixie")
	}

	if spec.CanExportData && spec.Experimental.Autotracing.Enable && spec.Experimental.Autotracing.CanExportAutotracingScripts {
		dirsToInclude = append(dirsToInclude, "autotracing")
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

	resourcesToApply, resourcesToDelete, err := r.readAndInterpolateResources(specToDelete)
	if err != nil {
		return err
	}

	err = r.KubernetesManager.DeleteResources(append(resourcesToApply, resourcesToDelete...))
	if err != nil {
		return err
	}

	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func resourceFiles(suffix string, spec wf.WavefrontSpec) (map[string]bool, error) {
	files := make(map[string]bool)
	dirsToApply := enabledDirs(spec)

	var currentDir string
	err := filepath.WalkDir(DeployDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			currentDir = entry.Name()
		}

		if strings.HasSuffix(path, suffix) {
			filePath := strings.Replace(path, DeployDir+"/", "", 1)
			if contains(dirsToApply, currentDir) {
				files[filePath] = true
			} else {
				files[filePath] = false
			}
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

// Preprocessing Wavefront Spec
func (r *WavefrontReconciler) preprocess(wavefront *wf.Wavefront, ctx context.Context) error {

	wavefront.Spec.Namespace = r.namespace
	wavefront.Spec.ClusterUUID = r.ClusterUUID

	wavefront.Spec.DataCollection.Metrics.CollectorVersion = r.Versions.CollectorVersion
	wavefront.Spec.DataExport.WavefrontProxy.ProxyVersion = r.Versions.ProxyVersion
	wavefront.Spec.DataCollection.Logging.LoggingVersion = r.Versions.LoggingVersion

	err := preprocessor.PreProcess(r.Client, wavefront)
	if err != nil {
		return err
	}

	if wavefront.Spec.CanExportData {
		err := r.MetricConnection.Connect(wavefront.Spec.DataCollection.Metrics.ProxyAddress)
		if err != nil {
			return fmt.Errorf("error setting up proxy connection: %s", err.Error())
		}
	}

	if r.shouldEnableEtcdCollection(wavefront, ctx) {
		wavefront.Spec.DataCollection.Metrics.ControlPlane.EnableEtcd = true
	}

	if r.isAnOpenshiftEnvironment() {
		wavefront.Spec.Openshift = true
	}

	r.components, err = components.BuildComponents(wavefront)
	if err != nil {
		return err
	}

	return nil
}

func (r *WavefrontReconciler) isAnOpenshiftEnvironment() bool {
	serverGroups, err := r.DiscoveryClient.ServerGroups()
	if err != nil {
		return false
	}

	for _, group := range serverGroups.Groups {
		if strings.Contains(group.Name, "openshift") {
			return true
		}
	}

	return false
}

func (r *WavefrontReconciler) shouldEnableEtcdCollection(wavefront *wf.Wavefront, ctx context.Context) bool {
	// never collect etcd if control plane metrics are disabled
	if !wavefront.Spec.DataCollection.Metrics.ControlPlane.Enable {
		return false
	}

	// only enable collection from etcd if the certs are supplied as a Secret
	key := client.ObjectKey{
		Namespace: r.namespace,
		Name:      "etcd-certs",
	}
	err := r.Client.Get(ctx, key, &corev1.Secret{})

	return err == nil
}

// Reporting Health Status
func (r *WavefrontReconciler) reportHealthStatus(ctx context.Context, wavefront *wf.Wavefront, validationResult validation.Result) (wf.WavefrontStatus, error) {

	// TODO: Component Refactor - use components to get which resources should be queried for status
	wavefrontStatus := health.GenerateWavefrontStatus(r.Client, wavefront)

	if !validationResult.IsValid() {
		wavefrontStatus.Status = health.Unhealthy
		wavefrontStatus.Message = validationResult.Message()
	}

	r.reportMetrics(!validationResult.IsError(), wavefront.Spec.ClusterName, wavefrontStatus)

	if wavefrontStatus.Status != wavefront.Status.Status {
		log.Log.Info(fmt.Sprintf("Wavefront CR wavefrontStatus changed from %s --> %s", wavefront.Status.Status, wavefrontStatus.Status))
		if !validationResult.IsValid() {
			log.Log.Info(fmt.Sprintf("Wavefront CR wavefrontStatus Unhealthy reasons: %s", validationResult.Message()))
		}
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

func errorCRTLResult(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}
