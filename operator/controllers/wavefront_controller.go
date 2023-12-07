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
	stderrors "errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/factory"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/preprocessor"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/version"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric/status"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/health"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		Watches(&source.Kind{Type: &rc.ResourceCustomizations{}}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, maxReconcileInterval),
		}).
		Complete(r)
}

// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=wavefronts/finalizers,verbs=update

// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=resourcecustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=resourcecustomizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wavefront.com,namespace=observability-system,resources=resourcecustomizations/finalizers,verbs=update

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
	crSet, err := r.fetchCRSet(ctx, r.namespace)
	if stderrors.Is(err, CRNotFoundErr) {
		_ = r.readAndDeleteResources()
		return ctrl.Result{}, nil
	}
	if err != nil {
		return errorCRTLResult(err)
	}

	var validationResult result.Aggregate
	err = r.preprocess(crSet, ctx)
	if err != nil {
		validationResult = result.Aggregate{
			crSet.Wavefront.GroupVersionKind(): result.NewError(err),
		}
	} else {
		validationResult = r.validate(crSet)
	}

	if !validationResult.HasErrors() {
		err = r.readAndCreateResources(crSet.Spec())
		if err != nil {
			return errorCRTLResult(err)
		}
	} else {
		_ = r.readAndDeleteResources()
	}
	healthy, err := r.reportHealthStatus(ctx, crSet, validationResult)
	if err != nil {
		return errorCRTLResult(err)
	}

	if !healthy {
		return ctrl.Result{
			Requeue: true,
		}, nil
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: maxReconcileInterval,
	}, nil
}

func (r *WavefrontReconciler) fetchCRSet(ctx context.Context, namespace string) (*api.CRSet, error) {
	wfCR, err := r.fetchWavefrontCR(ctx, namespace)
	if err != nil {
		return nil, err
	}
	rcCR, err := r.fetchResourceCustomizationsCR(ctx, namespace)
	if err != nil && !stderrors.Is(err, CRNotFoundErr) {
		return nil, err
	}
	return &api.CRSet{
		Wavefront:              wfCR,
		ResourceCustomizations: rcCR,
	}, nil
}

var CRNotFoundErr = fmt.Errorf("CR is not found")

func (r *WavefrontReconciler) fetchWavefrontCR(ctx context.Context, namespace string) (wf.Wavefront, error) {
	wavefrontList := &wf.WavefrontList{}
	err := r.Client.List(ctx, wavefrontList, client.InNamespace(namespace))
	if err != nil {
		return wf.Wavefront{}, err
	}
	if len(wavefrontList.Items) == 0 {
		return wf.Wavefront{}, CRNotFoundErr
	}
	if len(wavefrontList.Items) > 1 {
		return wf.Wavefront{}, fmt.Errorf("cannot have more than 1 Wavefront CR (have %d)", len(wavefrontList.Items))
	}
	return wavefrontList.Items[0], nil
}

func (r *WavefrontReconciler) fetchResourceCustomizationsCR(ctx context.Context, namespace string) (rc.ResourceCustomizations, error) {
	rcList := &rc.ResourceCustomizationsList{}
	err := r.Client.List(ctx, rcList, client.InNamespace(namespace))
	if err != nil {
		return rc.ResourceCustomizations{}, err
	}
	if len(rcList.Items) == 0 {
		return rc.ResourceCustomizations{}, CRNotFoundErr
	}
	if len(rcList.Items) > 1 {
		return rc.ResourceCustomizations{}, fmt.Errorf("cannot have more than 1 WorkloadCustomization CR (have %d)", len(rcList.Items))
	}
	return rcList.Items[0], nil
}

// Validating Wavefront CR
func (r *WavefrontReconciler) validate(crSet *api.CRSet) result.Aggregate {
	var res result.Result
	for _, component := range r.components {
		res = component.Validate()
		if res.IsError() {
			break
		}
	}

	if res.IsError() {
		return result.Aggregate{
			crSet.Wavefront.GroupVersionKind(): res,
		}
	}
	//TODO - Component Refactor - move all non cross component validation to components
	return validation.Validate(r.Client, crSet)
}

// Read, Create, Update and Delete Resources.
func (r *WavefrontReconciler) readAndCreateResources(specSet *api.SpecSet) error {
	toApply, toDelete, err := r.readAndInterpolateResources(specSet)
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

func (r *WavefrontReconciler) readAndInterpolateResources(specSet *api.SpecSet) ([]client.Object, []client.Object, error) {
	var resourcesToApply, resourcesToDelete []client.Object
	builder := components.NewK8sResourceBuilder(makeResourcePatch(specSet))
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

func makeResourcePatch(specSet *api.SpecSet) patch.Composed {
	resourcePatches := patch.Composed{}
	if specSet.ResourceCustomizationsSpec.All != nil {
		resourcePatches = append(resourcePatches, patch.Tolerations(specSet.ResourceCustomizationsSpec.All.Tolerations))
	}
	workloadPatches := patch.ByName{}
	for workloadName, customizations := range specSet.ResourceCustomizationsSpec.ByName {
		workloadPatches[workloadName] = patch.Composed{
			patch.ContainerResources(customizations.Resources),
			patch.Tolerations(customizations.Tolerations),
		}
	}
	resourcePatches = append(resourcePatches, workloadPatches)
	return resourcePatches
}

func (r *WavefrontReconciler) readAndDeleteResources() error {
	var err error
	r.MetricConnection.Close()
	crSetToDelete := &api.CRSet{
		Wavefront: wf.Wavefront{
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
		},
		ResourceCustomizations: rc.ResourceCustomizations{},
	}

	r.components, err = factory.BuildComponents(r.ComponentsDeployDir, &crSetToDelete.Wavefront, r.Client)
	if err != nil {
		return err
	}
	resourcesToApply, resourcesToDelete, err := r.readAndInterpolateResources(crSetToDelete.Spec())
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
func (r *WavefrontReconciler) preprocess(crSet *api.CRSet, ctx context.Context) error {

	crSet.Wavefront.Spec.Namespace = r.namespace
	crSet.Wavefront.Spec.ClusterUUID = r.ClusterUUID

	crSet.Wavefront.Spec.DataCollection.Metrics.CollectorVersion = r.Versions.CollectorVersion
	crSet.Wavefront.Spec.DataExport.WavefrontProxy.ProxyVersion = r.Versions.ProxyVersion
	crSet.Wavefront.Spec.DataCollection.Logging.LoggingVersion = r.Versions.LoggingVersion

	err := preprocessor.PreProcess(r.Client, crSet)
	if err != nil {
		return err
	}

	if crSet.Wavefront.Spec.CanExportData {
		err := r.MetricConnection.Connect(crSet.Wavefront.Spec.DataCollection.Metrics.ProxyAddress)
		if err != nil {
			return fmt.Errorf("error setting up proxy connection: %s", err.Error())
		}
	}

	if r.isAnOpenshiftEnvironment() {
		crSet.Wavefront.Spec.Openshift = true
	}

	r.components, err = factory.BuildComponents(r.ComponentsDeployDir, &crSet.Wavefront, r.Client)
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
func (r *WavefrontReconciler) reportHealthStatus(ctx context.Context, crSet *api.CRSet, validationResult result.Aggregate) (bool, error) {
	wfHealthy, wfErr := r.reportWFStatus(ctx, &crSet.Wavefront, validationResult[crSet.Wavefront.GroupVersionKind()])
	rcHealthy, rcErr := r.reportRCStatus(ctx, &crSet.ResourceCustomizations, validationResult[crSet.ResourceCustomizations.GroupVersionKind()])
	return wfHealthy && rcHealthy, utilerrors.NewAggregate([]error{wfErr, rcErr})
}

func (r *WavefrontReconciler) reportWFStatus(ctx context.Context, wavefront *wf.Wavefront, validationResult result.Result) (bool, error) {
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

	return wavefrontStatus.Status == health.Healthy, r.Status().Patch(ctx, &newWavefront, client.MergeFrom(wavefront))
}

func (r *WavefrontReconciler) reportRCStatus(ctx context.Context, rcCR *rc.ResourceCustomizations, result result.Result) (bool, error) {
	if rcCR.Name == "" {
		return true, nil
	}
	rcStatus := rc.ResourceCustomizationsStatus{
		Status: health.Healthy,
	}

	if !result.IsValid() {
		rcStatus.Message = result.Message()
		rcStatus.Status = health.Unhealthy
	}

	newRCCR := *rcCR
	newRCCR.Status = rcStatus

	return result.IsValid(), r.Status().Patch(ctx, &newRCCR, client.MergeFrom(rcCR))
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
