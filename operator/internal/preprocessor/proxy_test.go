package preprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnabledPorts(t *testing.T) {
	t.Run("computes default proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		SetEnabledPorts(wfcr)
		require.Equal(t, "2878", wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
	})

	t.Run("computes custom proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
		wfcr.Spec.DataExport.WavefrontProxy.Histogram.Port = 9999
		SetEnabledPorts(wfcr)
		require.Equal(t, "2878,4317,9999", wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
	})

}

func setup(initObjs ...runtime.Object) client.Client {
	return fake.NewClientBuilder().
		WithRuntimeObjects(initObjs...).
		Build()
}

func defaultWFCR() *wf.Wavefront {
	return &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testNamespace",
			Name:      "wavefront",
		},
		Spec: wf.WavefrontSpec{
			ClusterName:  "testClusterName",
			WavefrontUrl: "testWavefrontUrl",
			DataExport: wf.DataExport{
				WavefrontProxy: wf.WavefrontProxy{
					Enable:     true,
					MetricPort: 2878,
				},
			},
			DataCollection: wf.DataCollection{
				Metrics: wf.Metrics{
					Enable: true,
				},
			},
		},
		Status: wf.WavefrontStatus{},
	}
}
