package preprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProcess(t *testing.T) {
	t.Run("computes default proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		result, err := Process(setup(), wfcr)
		require.NoError(t, err)
		require.Equal(t, "2878", result.EnabledPorts)
	})

	t.Run("computes custom proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
		wfcr.Spec.DataExport.WavefrontProxy.Histogram.Port = 9999
		result, err := Process(setup(), wfcr)
		require.NoError(t, err)
		require.Equal(t, "2878,4317,9999", result.EnabledPorts)
	})

	t.Run("can parse user defined preprocessor rules", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "    '2878':\n      - rule: tag1\n        action: addTag\n        tag: tag1\n        value: \"true\"\n      - rule: tag2\n        action: addTag\n        tag: tag2\n        value: \"true\"\n    'global':\n      - rule: tag3\n        action: addTag\n        tag: tag3\n        value: \"true\"\n"

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		result, err := Process(client, wfcr)

		require.NoError(t, err)
		require.Contains(t, result.UserDefinedPortRules, "- rule: tag1\n  action: addTag\n  tag: tag1\n  value: \"true\"\n")
		require.Contains(t, result.UserDefinedPortRules, "- rule: tag2\n  action: addTag\n  tag: tag2\n  value: \"true\"\n")
		require.Contains(t, result.UserDefinedGlobalRules, "- rule: tag3\n  action: addTag\n  tag: tag3\n  value: \"true\"\n")
	})

	t.Run("returns error if user provides invalid preprocessor rule yaml", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "2878\":\\n- rule: tag1\\n  key: foo\\n"

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		_, err := Process(client, wfcr)

		require.Error(t, err)
	})

	t.Run("returns error proxy if user preprocessor port rules have a rule for cluster", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "'2878':\n      - rule: tag-cluster\n        action: addTag\n        tag: cluster\n        value: \"my-cluster\""

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		_, err := Process(client, wfcr)

		require.Error(t, err)
		require.Equal(t, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port '2878', overriding metric tag 'cluster' is disallowed", err.Error())
	})

	t.Run("returns error proxy if user preprocessor port rules have a rule for cluster_uuid", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "'2878':\n      - rule: tag-all-metrics-processed\n        action: spanAddTag\n        key: cluster_uuid\n        value: \"my-cluster-uuid\""

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		_, err := Process(client, wfcr)

		require.Error(t, err)
		require.Equal(t, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port '2878', overriding span tag 'cluster_uuid' is disallowed", err.Error())
	})

	t.Run("returns error proxy if user preprocessor global rules have a rule for cluster_uuid", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "'global':\n      - rule: tag-all-metrics-processed\n        action: spanAddTag\n        key: cluster_uuid\n        value: \"my-cluster-uuid\""

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		_, err := Process(client, wfcr)

		require.Error(t, err)
		require.Equal(t, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port 'global', overriding span tag 'cluster_uuid' is disallowed", err.Error())
	})

	t.Run("returns error proxy if user preprocessor global rules have a rule for cluster", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
		rules := "'global':\n      - rule: tag-all-metrics-processed\n        action: addTag\n        tag: cluster\n        value: \"my-cluster\""

		rulesConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfcr.Spec.DataExport.WavefrontProxy.Preprocessor,
				Namespace: wfcr.Namespace,
			},
			Data: map[string]string{
				"rules.yaml": rules,
			},
		}

		client := setup(rulesConfigMap)
		_, err := Process(client, wfcr)

		require.Error(t, err)
		require.Equal(t, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port 'global', overriding metric tag 'cluster' is disallowed", err.Error())
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
			Namespace:    "testNamespace",
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
