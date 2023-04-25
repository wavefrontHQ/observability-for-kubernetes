package controlplane

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/options"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"k8s.io/client-go/kubernetes/fake"
)

func TestProvider(t *testing.T) {
	leadership.SetLeading(true)
	util.SetAgentType(options.AllAgentType)
	var endpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{{IP: "127.0.0.1"}, {IP: "127.0.0.2"}},
			Ports:     []corev1.EndpointPort{{Name: "https", Port: 6443}},
		}},
	}

	t.Run("is identified as the correct provider", func(t *testing.T) {
		provider, _ := NewProvider(configuration.ControlPlaneSourceConfig{}, configuration.SummarySourceConfig{}, fake.NewSimpleClientset().CoreV1())

		assert.Equal(t, "control_plane_source", provider.Name())
	})

	t.Run("builds sources for each kubernetes api server instances", func(t *testing.T) {
		provider, _ := NewProvider(configuration.ControlPlaneSourceConfig{}, configuration.SummarySourceConfig{URL: "https://kube", InClusterConfig: "false"}, fake.NewSimpleClientset(endpoints).CoreV1())

		sources := provider.GetMetricsSources()
		assert.Equal(t, 4, len(sources), "2 prom providers querying the api x 2 instances of the api")
		assert.Equal(t, "prometheus_source: https://127.0.0.1:6443/metrics", sources[0].Name())
		assert.Equal(t, "prometheus_source: https://127.0.0.2:6443/metrics", sources[1].Name())
		assert.Equal(t, "prometheus_source: https://127.0.0.1:6443/metrics", sources[2].Name())
		assert.Equal(t, "prometheus_source: https://127.0.0.2:6443/metrics", sources[3].Name())
	})
}
