package preprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProcess(t *testing.T) {
	t.Run("computes default proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		err := PreProcess(setup(), wfcr)
		require.NoError(t, err)
		require.Equal(t, "2878", wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
	})

	t.Run("computes custom proxy ports", func(t *testing.T) {
		wfcr := defaultWFCR()
		wfcr.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
		wfcr.Spec.DataExport.WavefrontProxy.Histogram.Port = 9999
		err := PreProcess(setup(), wfcr)
		require.NoError(t, err)
		require.Equal(t, "2878,4317,9999", wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
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
		err := PreProcess(client, wfcr)

		require.NoError(t, err)
		require.Contains(t, wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: tag1\n  action: addTag\n  tag: tag1\n  value: \"true\"\n")
		require.Contains(t, wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: tag2\n  action: addTag\n  tag: tag2\n  value: \"true\"\n")
		require.Contains(t, wfcr.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedGlobalRules, "- rule: tag3\n  action: addTag\n  tag: tag3\n  value: \"true\"\n")
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
		err := PreProcess(client, wfcr)

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
		err := PreProcess(client, wfcr)

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
		err := PreProcess(client, wfcr)

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
		err := PreProcess(client, wfcr)

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
		err := PreProcess(client, wfcr)

		require.Error(t, err)
		require.Equal(t, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port 'global', overriding metric tag 'cluster' is disallowed", err.Error())
	})

}

func TestProcessWavefrontProxyAuth(t *testing.T) {
	t.Run("defaults to API token auth if no secret is found", func(t *testing.T) {
		fakeClient := setup()
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.NoError(t, err)
		require.Equal(t, util.WavefrontTokenAuthType, wfcr.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports wavefront api token auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"token": []byte("some-token"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.NoError(t, err)
		require.Equal(t, util.WavefrontTokenAuthType, wfcr.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports csp api token auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"csp-api-token": []byte("some-token"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.NoError(t, err)
		require.Equal(t, util.CSPTokenAuthType, wfcr.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports csp app secret auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("some-app-id"),
				"csp-app-secret": []byte("some-app-secret"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.NoError(t, err)
		require.Equal(t, util.CSPAppAuthType, wfcr.Spec.DataExport.WavefrontProxy.Auth.Type)
		require.Equal(t, "some-app-id", wfcr.Spec.DataExport.WavefrontProxy.Auth.CSPAppID)
		require.Equal(t, "", wfcr.Spec.DataExport.WavefrontProxy.Auth.CSPOrgId)
	})

	t.Run("supports csp app secret auth with org id", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("some-app-id"),
				"csp-org-id":     []byte("some-org-id"),
				"csp-app-secret": []byte("some-app-secret"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.NoError(t, err)
		require.Equal(t, util.CSPAppAuthType, wfcr.Spec.DataExport.WavefrontProxy.Auth.Type)
		require.Equal(t, "some-app-id", wfcr.Spec.DataExport.WavefrontProxy.Auth.CSPAppID)
		require.Equal(t, "some-org-id", wfcr.Spec.DataExport.WavefrontProxy.Auth.CSPOrgId)
	})

	t.Run("returns validation error if wavefront token and csp api token are given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"token":         []byte("some-token"),
				"csp-api-token": []byte("some-other-token"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.Error(t, err)
		require.Equal(t, "Invalid Authentication configured in Secret 'wavefront-secret'. Only one authentication type is allowed. Wavefront API Token or CSP API Token or CSP App OAuth", err.Error())
	})

	t.Run("returns validation error if empty wavefront token and csp api token are given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"token":         []byte(""),
				"csp-api-token": []byte("some-other-token"),
			},
		}
		fakeClient := setup(secret)
		wfcr := defaultWFCR()
		err := PreProcess(fakeClient, wfcr)
		require.Error(t, err)
		require.Equal(t, "Invalid Authentication configured in Secret 'wavefront-secret'. Only one authentication type is allowed. Wavefront API Token or CSP API Token or CSP App OAuth", err.Error())
	})
}

func setup(initObjs ...runtime.Object) client.Client {
	operator := wftest.Operator()
	operator.SetNamespace("testNamespace")
	initObjs = append(initObjs, operator)

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
			ClusterName:          "testClusterName",
			WavefrontTokenSecret: "testWavefrontSecret",
			WavefrontUrl:         "testWavefrontUrl",
			Namespace:            "testNamespace",
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
