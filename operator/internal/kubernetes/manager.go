package kubernetes_manager

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error

	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

type KubernetesManager struct {
	objClient Client
}

func NewKubernetesManager(objClient Client) *KubernetesManager {
	return &KubernetesManager{objClient: objClient}
}

func (km *KubernetesManager) ApplyResources(resources []client.Object) error {
	for _, resource := range resources {
		gvk := resource.GetObjectKind().GroupVersionKind()
		var getObj unstructured.Unstructured
		getObj.SetGroupVersionKind(gvk)
		err := km.objClient.Get(context.Background(), util.ObjKey(resource.GetNamespace(), resource.GetName()), &getObj)
		if errors.IsNotFound(err) {
			err = km.objClient.Create(context.Background(), resource)
		} else if err == nil && gvk.Kind != "Job" {
			var diffObj unstructured.Unstructured
			diffObj.SetGroupVersionKind(gvk)
			err = km.objClient.Patch(context.Background(), resource, client.MergeFrom(&diffObj))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (km *KubernetesManager) DeleteResources(resources []client.Object) error {
	for _, resource := range resources {
		err := km.objClient.Delete(context.Background(), resource)
		if err != nil && !errors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return err
		}
	}
	return nil
}
