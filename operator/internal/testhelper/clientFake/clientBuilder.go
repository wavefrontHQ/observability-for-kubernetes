package clientFake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Setup(initObjs ...runtime.Object) client.Client {
	return fake.NewClientBuilder().
		WithRuntimeObjects(initObjs...).
		Build()
}
