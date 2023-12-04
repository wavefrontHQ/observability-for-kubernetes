package preprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = wftest.DefaultNamespace

func TestProcess(t *testing.T) {
	t.Run("Wavefront", func(t *testing.T) {
		t.Run("computes default proxy ports", func(t *testing.T) {
			crSet := defaultCRSet()
			err := PreProcess(setup(), crSet)
			require.NoError(t, err)
			require.Equal(t, "2878", crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
		})

		t.Run("computes custom proxy ports", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Histogram.Port = 9999
			err := PreProcess(setup(), crSet)
			require.NoError(t, err)
			require.Equal(t, "2878,4317,9999", crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts)
		})

		t.Run("can parse user defined preprocessor rules", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule: tag1\n        action: addTag\n        tag: tag1\n        value: \"true\"\n      - rule: tag2\n        action: addTag\n        tag: tag2\n        value: \"true\"\n    'global':\n      - rule: tag3\n        action: addTag\n        tag: tag3\n        value: \"true\"\n"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: tag1\n  action: addTag\n  tag: tag1\n  value: \"true\"\n")
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: tag2\n  action: addTag\n  tag: tag2\n  value: \"true\"\n")
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedGlobalRules, "- rule: tag3\n  action: addTag\n  tag: tag3\n  value: \"true\"\n")
		})

		t.Run("can parse user defined preprocessor rules with scope, search, replace, source", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule    : example-replace-badchars\n        action  : replaceRegex\n        scope   : pointLine\n        search  : \"[&\\\\$\\\\*]\"\n        replace : \"_\"\n        source  : sourceName"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: example-replace-badchars\n  action: replaceRegex\n  scope: pointLine\n  search: '[&\\$\\*]'\n  replace: _\n  source: sourceName")
		})

		t.Run("can parse user defined preprocessor rules with match", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule    : drop-az-tag\n        action  : dropTag\n        tag     : az\n        match   : dev.*"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: drop-az-tag\n  action: dropTag\n  tag: az\n  match: dev.*\n")
		})

		t.Run("can parse user defined preprocessor rules with function, names, opts", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			// TODO: revisit for metric\.2
			rules := "    '2878':\n      - rule: allow-selected-metrics\n        action: metricsFilter\n        function: allow\n        opts:\n          cacheSize: 10000\n        names:\n          - \"metrics.1\"\n          - \"/.*.ok$/\"\n          - \"/metrics.2.*/\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: allow-selected-metrics\n  action: metricsFilter\n  function: allow\n  names:\n  - metrics.1\n  - /.*.ok$/\n  - /metrics.2.*/")
		})

		t.Run("can parse user defined preprocessor rules with newtag", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule    : rename-dc-to-datacenter\n        action  : renameTag\n        tag     : dc\n        newtag  : datacenter"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: rename-dc-to-datacenter\n  action: renameTag\n  tag: dc\n  newtag: datacenter")
		})

		t.Run("can parse user defined preprocessor rules with actionSubtype, maxLength", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule          : limit-metric-name-length\n        action        : limitLength\n        scope         : metricName\n        actionSubtype : truncateWithEllipsis\n        maxLength     : 16\n        match         : \"^metric.*\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: limit-metric-name-length\n  action: limitLength\n  match: ^metric.*\n  scope: metricName")
		})

		t.Run("can parse user defined preprocessor rules with iterations,firstMatchOnly", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule          : example-span-force-lowercase\n        action        : spanForceLowercase\n        scope         : spanName\n        match         : \"^UPPERCASE.*$\"\n        firstMatchOnly: false\n        iterations : '10'"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: example-span-force-lowercase\n  action: spanForceLowercase\n  match: ^UPPERCASE.*$\n  scope: spanName\n  iterations: \"10\"")
		})

		t.Run("can parse user defined preprocessor rules with input, replaceInput", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule          : example-extract-tag-from-span\n        action        : spanExtractTag\n        key           : serviceTag\n        input         : spanName\n        match         : \"span.*\"\n        search        : \"^([^\\\\.]*\\\\.[^\\\\.]*\\\\.)([^\\\\.]*)\\\\.(.*)$\"\n        replaceInput  : \"$1$3\"\n        replace       : \"$2\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: example-extract-tag-from-span\n  action: spanExtractTag\n  key: serviceTag\n  match: span.*\n  search: ^([^\\.]*\\.[^\\.]*\\.)([^\\.]*)\\.(.*)$\n  replace: $2\n  input: spanName\n  replaceInput: $1$3\n")
		})

		t.Run("can parse user defined preprocessor rules with newKey", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule   : rename-span-tag-x-request-id\n        action : spanRenameTag\n        key    : guid:x-request-id\n        newkey : guid-x-request-id"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: rename-span-tag-x-request-id\n  action: spanRenameTag\n  key: guid:x-request-id\n  newkey: guid-x-request-id\n")
		})

		t.Run("can parse user defined preprocessor rules with if condition", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "    '2878':\n      - rule: test-spanblock-list\n        action: spanBlock\n        if:\n          equals:\n            scope: http.status_code\n            value: [\"302, 404\"]"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: test-spanblock-list\n  action: spanBlock\n  if:\n    equals:\n      scope: http.status_code\n      value:\n      - 302, 404\n")
		})

		t.Run("can parse raw string user defined preprocessor rules with if condition", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := `
    '2878':
      - rule: tag-all-metrics-processed
        action: addTag
        tag: processed
        value: "true"
        if:
          startsWith:
            scope: metricName
            value: "kubernetes.collector.version"
`

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: tag-all-metrics-processed\n  action: addTag\n  tag: processed\n  value: \"true\"")
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "if:\n    startsWith:\n      scope: metricName\n      value: kubernetes.collector.version")
		})

		t.Run("can parse raw string user defined preprocessor point filtering rules", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := `
    '2878':
      - rule: example-block-west
        action: block
        scope: datacenter
        match: "west.*"
      - rule: example-allow-only-prod
        action: allow
        scope: pointLine
        match: ".*prod.*"
      - rule: allow-selected-metrics
        action: metricsFilter
        function: allow
        names:
          - "metrics.1"
          - "/metrics\\.2.*/"
          - "/.*.ok$/"
        opts:
          cacheSize: 10000
`

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.NoError(t, err)
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: example-block-west\n  action: block\n  match: west.*\n  scope: datacenter")
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: example-allow-only-prod\n  action: allow\n  match: .*prod.*\n  scope: pointLine")
			require.Contains(t, crSet.Wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedPortRules, "- rule: allow-selected-metrics\n  action: metricsFilter\n  function: allow\n  names:\n  - metrics.1\n  - /metrics\\.2.*/\n  - /.*.ok$/\n  opts:\n    cacheSize: 10000")
		})

		t.Run("returns error if user provides invalid preprocessor rule yaml", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "2878\":\\n- rule: tag1\\n  key: foo\\n"

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.Error(t, err)
		})

		t.Run("returns error proxy if user preprocessor port rules have a rule for cluster", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "'2878':\n      - rule: tag-cluster\n        action: addTag\n        tag: cluster\n        value: \"my-cluster\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.ErrorContains(t, err, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port '2878', overriding metric tag 'cluster' is disallowed")
		})

		t.Run("returns error proxy if user preprocessor port rules have a rule for cluster_uuid", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "'2878':\n      - rule: tag-all-metrics-processed\n        action: spanAddTag\n        key: cluster_uuid\n        value: \"my-cluster-uuid\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.ErrorContains(t, err, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port '2878', overriding span tag 'cluster_uuid' is disallowed")
		})

		t.Run("returns error proxy if user preprocessor global rules have a rule for cluster_uuid", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "'global':\n      - rule: tag-all-metrics-processed\n        action: spanAddTag\n        key: cluster_uuid\n        value: \"my-cluster-uuid\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.ErrorContains(t, err, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port 'global', overriding span tag 'cluster_uuid' is disallowed")
		})

		t.Run("returns error proxy if user preprocessor global rules have a rule for cluster", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			rules := "'global':\n      - rule: tag-all-metrics-processed\n        action: addTag\n        tag: cluster\n        value: \"my-cluster\""

			rulesConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crSet.Wavefront.Spec.DataExport.WavefrontProxy.Preprocessor,
					Namespace: crSet.Wavefront.Namespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			}

			client := setup(rulesConfigMap)
			err := PreProcess(client, crSet)

			require.ErrorContains(t, err, "Invalid rule configured in ConfigMap 'user-preprocessor-rules' on port 'global', overriding metric tag 'cluster' is disallowed", err.Error())
		})
	})

	t.Run("ResourceCustomizations", func(t *testing.T) {
		t.Run("when only limit is set, sets request to match", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.ResourceCustomizations.Spec.ByName = map[string]rc.WorkloadCustomization{
				"some-deployment": {
					Resources: common.ContainerResources{
						Limits: common.ContainerResource{
							CPU:              "100m",
							Memory:           "100Mi",
							EphemeralStorage: "200Mi",
						},
					},
				},
			}

			require.NoError(t, PreProcess(setup(), crSet))

			resources := crSet.ResourceCustomizations.Spec.ByName["some-deployment"].Resources
			require.Equal(t, resources.Limits, resources.Requests)
		})

		t.Run("does not override request when request is set", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.ResourceCustomizations.Spec.ByName = map[string]rc.WorkloadCustomization{
				"some-deployment": {
					Resources: common.ContainerResources{
						Requests: common.ContainerResource{
							CPU:              "50m",
							Memory:           "50Mi",
							EphemeralStorage: "100Mi",
						},
						Limits: common.ContainerResource{
							CPU:              "100m",
							Memory:           "100Mi",
							EphemeralStorage: "200Mi",
						},
					},
				},
			}

			require.NoError(t, PreProcess(setup(), crSet))

			resources := crSet.ResourceCustomizations.Spec.ByName["some-deployment"].Resources
			require.NotEqual(t, resources.Limits, resources.Requests)
		})
	})
}

func TestProcessWavefrontProxyAuth(t *testing.T) {
	t.Run("defaults to API token auth if no secret is found", func(t *testing.T) {
		fakeClient := setup()
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.NoError(t, err)
		require.Equal(t, util.WavefrontTokenAuthType, crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports wavefront api token auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"token": []byte("some-token"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.NoError(t, err)
		require.Equal(t, util.WavefrontTokenAuthType, crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports csp api token auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"csp-api-token": []byte("some-token"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.NoError(t, err)
		require.Equal(t, util.CSPTokenAuthType, crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.Type)
	})

	t.Run("supports csp app secret auth", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("some-app-id"),
				"csp-app-secret": []byte("some-app-secret"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.NoError(t, err)
		require.Equal(t, util.CSPAppAuthType, crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.Type)
		require.Equal(t, "some-app-id", crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.CSPAppID)
		require.Equal(t, "", crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.CSPOrgID)
	})

	t.Run("supports csp app secret auth with org id", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("some-app-id"),
				"csp-org-id":     []byte("some-org-id"),
				"csp-app-secret": []byte("some-app-secret"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.NoError(t, err)
		require.Equal(t, util.CSPAppAuthType, crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.Type)
		require.Equal(t, "some-app-id", crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.CSPAppID)
		require.Equal(t, "some-org-id", crSet.Wavefront.Spec.DataExport.WavefrontProxy.Auth.CSPOrgID)
	})

	t.Run("returns validation error if wavefront token and csp api token are given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"token":         []byte("some-token"),
				"csp-api-token": []byte("some-other-token"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'testWavefrontSecret'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})

	t.Run("returns validation error if empty auth key and non empty auth key given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"token":         []byte(""),
				"csp-api-token": []byte("some-other-token"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'testWavefrontSecret'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})

	t.Run("returns validation error if wavefront token and csp app oauth are given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"token":      []byte("some-token"),
				"csp-app-id": []byte("some-id"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'testWavefrontSecret'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})

	t.Run("returns validation error if csp api token and csp app oauth are given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"csp-api-token": []byte("some-token"),
				"csp-app-id":    []byte("some-id"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()

		err := PreProcess(fakeClient, crSet)

		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'testWavefrontSecret'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})

	t.Run("returns correct secret name", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"csp-api-token": []byte("some-token"),
				"csp-app-id":    []byte("some-id"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.WavefrontTokenSecret = "my-secret"

		err := PreProcess(fakeClient, crSet)

		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'my-secret'. Only one authentication type is allowed. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})

	t.Run("returns error if no auth type given", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testWavefrontSecret",
				Namespace: testNamespace,
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		err := PreProcess(fakeClient, crSet)
		require.ErrorContains(t, err, "Invalid authentication configured in Secret 'testWavefrontSecret'. Missing Authentication type. Wavefront API Token 'token' or CSP API Token 'csp-api-token' or CSP App OAuth 'csp-app-id")
	})
}

func TestProcessExperimental(t *testing.T) {
	t.Run("succeeds when insights secret exists", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.InsightsSecret,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"ingestion-token": []byte("ignored"),
			},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		crSet.Wavefront.Spec.Experimental.Insights.IngestionUrl = "https://example.com"

		err := PreProcess(fakeClient, crSet)

		require.NoError(t, err)
	})

	t.Run("surfaces error when insights-secret doesn't exist when insights enabled", func(t *testing.T) {
		fakeClient := setup()
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		crSet.Wavefront.Spec.Experimental.Insights.IngestionUrl = "https://example.com"

		err := PreProcess(fakeClient, crSet)

		require.ErrorContains(t, err, "Invalid authentication configured for Experimental Insights. Missing Secret 'insights-secret'")
	})

	t.Run("surfaces error when token key doesn't exist in insights-secret", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.InsightsSecret,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{},
		}
		fakeClient := setup(secret)
		crSet := defaultCRSet()
		crSet.Wavefront.Spec.Experimental.Insights.Enable = true
		crSet.Wavefront.Spec.Experimental.Insights.IngestionUrl = "https://example.com"

		err := PreProcess(fakeClient, crSet)

		require.ErrorContains(t, err, "Invalid authentication configured for Experimental Insights. Secret 'insights-secret' is missing Data 'ingestion-token'")
	})

	t.Run("properly sets canExportAutotracingScripts when pixie components are not running", func(t *testing.T) {
		fakeClient := setup()
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
		})

		_ = PreProcess(fakeClient, crSet)

		require.False(t, crSet.Wavefront.Spec.Experimental.Autotracing.CanExportAutotracingScripts)
	})

	t.Run("properly sets canExportAutotracingScripts when pixie components are running", func(t *testing.T) {
		daemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vizier-pem",
				Namespace: testNamespace,
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		fakeClient := setup(daemonset)
		crSet := defaultCRSet()
		crSet.Wavefront = *wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
		})

		_ = PreProcess(fakeClient, crSet)

		require.True(t, crSet.Wavefront.Spec.Experimental.Autotracing.CanExportAutotracingScripts)
	})

}

func setup(initObjs ...runtime.Object) client.Client {
	operator := wftest.Operator()
	operator.SetNamespace(testNamespace)
	initObjs = append(initObjs, operator)

	return fake.NewClientBuilder().
		WithRuntimeObjects(initObjs...).
		Build()
}

func defaultCRSet() *api.CRSet {
	return &api.CRSet{
		Wavefront: wf.Wavefront{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "wavefront",
			},
			Spec: wf.WavefrontSpec{
				ClusterName:          "testClusterName",
				WavefrontTokenSecret: "testWavefrontSecret",
				WavefrontUrl:         "testWavefrontUrl",
				Namespace:            testNamespace,
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
		},
	}
}
