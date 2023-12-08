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
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/factory"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/preprocessor"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/version"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/status"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/health"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

type KubernetesManager interface {
	ApplyResources(resourceYAMLs []client.Object) error
	DeleteResources(resourceYAMLs []client.Object) error
}

// WavefrontReconciler reconciles a Wavefront object
type WavefrontReconciler struct {
	client.Client

	ComponentsDeployDir fs.FS
	KubernetesManager   KubernetesManager
	DiscoveryClient     discovery.ServerGroupsInterface
	MetricConnection    *metric.Connection
	Versions            Versions
	namespace           string
	ClusterUUID         string

	components []components.Component
}

type Versions struct {
	OperatorVersion  string
	CollectorVersion string
	ProxyVersion     string
	LoggingVersion   string
}

func NewWavefrontReconciler(versions Versions, client client.Client, discoveryClient discovery.ServerGroupsInterface, clusterUUID string) (operator *WavefrontReconciler, err error) {
	return &WavefrontReconciler{
		Versions:            versions,
		Client:              client,
		ComponentsDeployDir: os.DirFS(components.DeployDir),
		KubernetesManager:   kubernetes_manager.NewKubernetesManager(client),
		DiscoveryClient:     discoveryClient,
		MetricConnection:    metric.NewConnection(metric.WavefrontSenderFactory()),
		ClusterUUID:         clusterUUID,
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

var reconcileCounter = 0

func (r *WavefrontReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.Info(fmt.Sprintf("Iteration %d: Beginning Reconcile loop.", reconcileCounter))
	r.namespace = req.Namespace
	wavefront := &wf.Wavefront{}
	err := r.Client.Get(ctx, req.NamespacedName, wavefront)
	if err != nil && !errors.IsNotFound(err) {
		log.Log.Info(fmt.Sprintf("Iteration %d: Returning from lookup failure.", reconcileCounter))
		reconcileCounter += 1
		return errorCRTLResult(err)
	}

	if errors.IsNotFound(err) {
		log.Log.Info(fmt.Sprintf("Iteration %d: Deleting Resources because Namespace not found", reconcileCounter))
		reconcileCounter += 1
		_ = r.readAndDeleteResources()
		return ctrl.Result{}, nil
	}

	var validationResult validation.Result
	err = r.preprocess(wavefront, ctx)
	if err != nil {
		log.Log.Info(fmt.Sprintf("Iteration %d: Encountered an error during preproccessing", reconcileCounter))
		validationResult = validation.NewErrorResult(err)
	} else {
		validationResult = r.validate(wavefront)
	}

	if !validationResult.IsError() {
		log.Log.Info(fmt.Sprintf("Iteration %d: Vaidation successful. Creating resources.", reconcileCounter))
		err = r.readAndCreateResources(wavefront.Spec)
		if err != nil {
			log.Log.Info(fmt.Sprintf("Iteration %d: Encountered error creating resources.", reconcileCounter))
			reconcileCounter += 1
			return errorCRTLResult(err)
		}
		//}
	} else {
		log.Log.Info(fmt.Sprintf("Iteration %d: Found a Validation error; deleting resources: %s.", reconcileCounter, validationResult.Message()))
		_ = r.readAndDeleteResources()
	}
	wavefrontStatus, err := r.reportHealthStatus(ctx, wavefront, validationResult)
	if err != nil {
		log.Log.Info(fmt.Sprintf("Iteration %d: Encountered an error reporting health status.", reconcileCounter))
		reconcileCounter += 1
		return errorCRTLResult(err)
	}

	reconcileCounter += 1
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

// Validating Wavefront CR
func (r *WavefrontReconciler) validate(wavefront *wf.Wavefront) validation.Result {
	var result validation.Result
	for _, component := range r.components {
		result = component.Validate()
		if result.IsError() {
			break
		}
	}

	if result.IsError() {
		return result
	}
	//TODO - Component Refactor - move all non cross component validation to components
	return validation.Validate(r.Client, wavefront)
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
	var resourcesToApply, resourcesToDelete []client.Object
	resourcePatches := patch.ByName{}
	for workloadName, resources := range spec.WorkloadResources {
		resourcePatches[workloadName] = patch.ContainerResources(resources)
	}
	builder := components.NewK8sResourceBuilder(resourcePatches)
	for _, component := range r.components {
		toApply, toDelete, err := component.Resources(builder)
		if err != nil {
			log.Log.Error(err, "could not get resources", "component", component.Name())
		}
		resourcesToApply = append(resourcesToApply, toApply...)
		resourcesToDelete = append(resourcesToDelete, toDelete...)
	}
	return resourcesToApply, resourcesToDelete, nil
}

func (r *WavefrontReconciler) readAndDeleteResources() error {
	var err error
	r.MetricConnection.Close()
	wfToDelete := &wf.Wavefront{
		Spec: wf.WavefrontSpec{
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
		},
	}

	r.components, err = factory.BuildComponents(r.ComponentsDeployDir, wfToDelete, r.Client)
	if err != nil {
		return err
	}
	resourcesToApply, resourcesToDelete, err := r.readAndInterpolateResources(wfToDelete.Spec)
	if err != nil {
		return err
	}

	err = r.KubernetesManager.DeleteResources(append(resourcesToApply, resourcesToDelete...))
	if err != nil {
		return err
	}

	return nil
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

	if r.isAnOpenshiftEnvironment() {
		wavefront.Spec.Openshift = true
	}

	r.components, err = factory.BuildComponents(r.ComponentsDeployDir, wavefront, r.Client)
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

// Reporting Metrics
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
