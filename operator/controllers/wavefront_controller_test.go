package controllers_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/health"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/wavefront/metric"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/controllers"
)

func TestReconcileAll(t *testing.T) {
	t.Run("produces well formed YAML", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
		}), nil, wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		mockKM.ForAllAppliedYAMLs(func(appliedYAML client.Object) {
			kind := appliedYAML.GetObjectKind().GroupVersionKind().Kind
			namespace := appliedYAML.GetNamespace()
			name := appliedYAML.GetName()
			expectResource := fmt.Sprintf("expect %s %s/%s to", kind, namespace, name)
			require.Equalf(t, "wavefront", appliedYAML.GetLabels()["app.kubernetes.io/name"], "%s have an app.kubernetes.io/name label", expectResource)
			require.NotEmptyf(t, appliedYAML.GetLabels()["app.kubernetes.io/component"], "%s have an app.kubernetes.io/component label", expectResource)

			require.NotEmptyf(t, appliedYAML.GetOwnerReferences(), "%s have an owner reference to the controller-manager", expectResource)
			ownerRef := appliedYAML.GetOwnerReferences()[0]
			require.Equalf(t, "apps/v1", ownerRef.APIVersion, "%s have the proper owner reference api version", expectResource)
			require.Equalf(t, "Deployment", ownerRef.Kind, "%s have the proper owner reference kind", expectResource)
			require.Equalf(t, "wavefront-controller-manager", ownerRef.Name, "%s have the proper owner reference name", expectResource)
			require.NotEmptyf(t, ownerRef.UID, "%s have the proper owner reference UID", expectResource)
		})
	})

	t.Run("does not create other services until the proxy is running", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(), nil, wftest.Proxy(wftest.WithReplicas(0, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.CollectorServiceAccountContains())
		require.False(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains())
		require.False(t, mockKM.ClusterCollectorDeploymentContains())
		require.False(t, mockKM.LoggingDaemonSetContains())

		require.True(t, mockKM.ProxyServiceContains("port: 2878"))
		require.True(t, mockKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878"))

		require.Equal(t, 0, len(mockSender.SentMetrics), "should not have sent metrics")
	})

	t.Run("creates other components after the proxy is running", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(), nil, wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.True(t, mockKM.CollectorServiceAccountContains("OperatorUUID"))
		require.True(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878", "OperatorUUID"))
		require.True(t, mockKM.NodeCollectorDaemonSetContains(fmt.Sprintf("kubernetes-collector:%s", r.Versions.CollectorVersion), "OperatorUUID"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains(fmt.Sprintf("kubernetes-collector:%s", r.Versions.CollectorVersion), "OperatorUUID"))
		require.True(t, mockKM.LoggingDaemonSetContains(fmt.Sprintf("kubernetes-operator-fluentbit:%s", r.Versions.LoggingVersion), "OperatorUUID"))
		require.True(t, mockKM.ProxyDeploymentContains(fmt.Sprintf("proxy:%s", r.Versions.ProxyVersion), "OperatorUUID"))

		require.Greater(t, len(mockSender.SentMetrics), 0, "should not have sent metrics")
		require.Equal(t, 99.9999, VersionSent(mockSender), "should send OperatorVersion")

	})

	t.Run("transitions status when sub-components change (even if overall health is still unhealthy)", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Status.Status = health.Unhealthy
		})
		r, _ := componentScenario(wfCR, nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		var reconciledWFCR wf.Wavefront

		require.NoError(t, r.Client.Get(
			context.Background(),
			util.ObjKey(wfCR.Namespace, wfCR.Name),
			&reconciledWFCR,
		))

		require.Contains(t, reconciledWFCR.Status.Status, health.Unhealthy)
		require.Contains(t, reconciledWFCR.Status.ResourceStatuses, wf.ResourceStatus{Status: "Running (1/1)", Name: "wavefront-proxy"})
	})

	t.Run("doesn't create any resources if wavefront spec is invalid", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = true
			w.Spec.DataExport.ExternalWavefrontProxy.Url = "http://some_url.com"
		}), nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.AppliedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.False(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		require.False(t, mockKM.AppliedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))
		require.False(t, mockKM.AppliedContains("v1", "ServiceAccount", "wavefront", "proxy", "wavefront-proxy"))
		require.False(t, mockKM.AppliedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		require.False(t, mockKM.AppliedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))

		require.Equal(t, 0, StatusMetricsSent(mockSender), "should not have sent status metrics")
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		r, mockKM := emptyScenario(nil, nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))
		_ = r.MetricConnection.Connect("http://example.com")

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.DeletedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		require.True(t, mockKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.True(t, mockKM.DeletedContains("apps/v1", "DaemonSet", "wavefront", "node-collector", "wavefront-node-collector"))
		require.True(t, mockKM.DeletedContains("apps/v1", "Deployment", "wavefront", "cluster-collector", "wavefront-cluster-collector"))
		require.True(t, mockKM.DeletedContains("v1", "ServiceAccount", "wavefront", "proxy", "wavefront-proxy"))
		require.True(t, mockKM.DeletedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		require.True(t, mockKM.DeletedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))

		require.Equal(t, 1, mockSender.Closes)
	})

	t.Run("Defaults Custom Registry", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-collector"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-collector"))
		require.True(t, mockKM.LoggingDaemonSetContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentbit"))
		require.True(t, mockKM.ProxyDeploymentContains("image: wavefronthq/proxy"))
	})

	t.Run("Can Configure Custom Registry", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), nil, wftest.Operator(func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Image = "docker.io/kubernetes-operator:latest"
		}))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("image: docker.io/kubernetes-collector"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("image: docker.io/kubernetes-collector"))
		require.True(t, mockKM.LoggingDaemonSetContains("image: docker.io/kubernetes-operator-fluentbit"))
		require.True(t, mockKM.ProxyDeploymentContains("image: docker.io/proxy"))
	})

	t.Run("Can configure imagePullSecrets for a private Custom Registry", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.ImagePullSecret = "private-reg-cred"
		})
		r, mockKM := componentScenario(wfCR, nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		request := defaultRequest()
		request.Namespace = wfCR.Namespace
		_, err := r.Reconcile(context.Background(), request)
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("imagePullSecrets:\n      - name: private-reg-cred"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("imagePullSecrets:\n      - name: private-reg-cred"))
		require.True(t, mockKM.LoggingDaemonSetContains("imagePullSecrets:\n      - name: private-reg-cred"))
		require.True(t, mockKM.ProxyDeploymentContains("imagePullSecrets:\n      - name: private-reg-cred"))
	})

	t.Run("Child components inherits controller's namespace", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Namespace = "customNamespace"
		})
		r, mockKM := componentScenario(wfCR, nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		request := defaultRequest()
		request.Namespace = wfCR.Namespace
		_, err := r.Reconcile(context.Background(), request)
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("namespace: customNamespace"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("namespace: customNamespace"))
		require.True(t, mockKM.LoggingDaemonSetContains("namespace: customNamespace"))
		require.True(t, mockKM.ProxyDeploymentContains("namespace: customNamespace"))
	})

	t.Run("Can configure additional data collection daemonset tolerations", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Enable = true
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.DataCollection.Tolerations = []wf.Toleration{
				wf.Toleration{
					Key:    "my-toleration",
					Value:  "my-value",
					Effect: "NoSchedule",
				},
			}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("- effect: NoSchedule\n        key: my-toleration\n        value: my-value"))
		require.True(t, mockKM.LoggingDaemonSetContains("- effect: NoSchedule\n        key: my-toleration\n        value: my-value"))
		require.False(t, mockKM.ClusterCollectorDeploymentContains("- effect: NoSchedule\n        key: my-toleration\n        value: my-value"))
		require.False(t, mockKM.ProxyDeploymentContains("- effect: NoSchedule\n        key: my-toleration\n        value: my-value"))
	})

	t.Run("deploys openshift resources if on an openshift environment", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), []string{"security.openshift.io"})
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "openshift-service-ca-bundle"))
		require.True(t, mockKM.AppliedContains("security.openshift.io/v1", "SecurityContextConstraints", "wavefront", "collector", "wavefront-collector-scc"))
		require.True(t, mockKM.AppliedContains("security.openshift.io/v1", "SecurityContextConstraints", "wavefront", "proxy", "wavefront-proxy-scc"))
	})

	t.Run("does not deploy openshift resources if not on an openshift environment", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.AppliedContains("security.openshift.io/v1", "Configmap", "wavefront", "collector", "openshift-service-ca-bundle"))
		require.False(t, mockKM.AppliedContains("security.openshift.io/v1", "SecurityContextConstraints", "wavefront", "collector", "wavefront-collector-scc"))
		require.False(t, mockKM.AppliedContains("security.openshift.io/v1", "SecurityContextConstraints", "wavefront", "proxy", "wavefront-proxy-scc"))
	})

	t.Run("can override workload resources", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
			wavefront.Spec.WorkloadResources = map[string]wf.Resources{
				"wavefront-proxy": {
					Requests: wf.Resource{
						CPU:    "50m",
						Memory: "50Mi",
					},
					Limits: wf.Resource{
						CPU:    "100m",
						Memory: "100Mi",
					},
				},
			}
		}), nil, wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		proxy, err := mockKM.GetProxyDeployment()

		require.NoError(t, err)

		require.Equal(t, "50m", proxy.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "50Mi", proxy.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "2Gi", proxy.Spec.Template.Spec.Containers[0].Resources.Requests.StorageEphemeral().String())

		require.Equal(t, "100m", proxy.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
		require.Equal(t, "100Mi", proxy.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		require.Equal(t, "8Gi", proxy.Spec.Template.Spec.Containers[0].Resources.Limits.StorageEphemeral().String())
	})
}

func TestReconcileCollector(t *testing.T) {
	t.Run("does not create configmap if user specified one", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.CustomConfig = "myconfig"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.True(t, mockKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
	})

	t.Run("can change the default collection interval", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.DefaultCollectionInterval = "90s"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("defaultCollectionInterval: 90s"))
	})

	t.Run("can disable discovery", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.EnableDiscovery = false
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("enableDiscovery: false"))
	})

	t.Run("control plane metrics are propagated to default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		configMap, err := mockKM.GetAppliedYAML(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:2878",
		)
		require.NoError(t, err)

		configStr, found, err := unstructured.NestedString(configMap.Object, "data", "config.yaml")
		require.Equal(t, true, found)
		require.NoError(t, err)

		var configs map[string]interface{}
		err = yaml.Unmarshal([]byte(configStr), &configs)
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("prometheus_sources"))
		require.True(t, mockKM.CollectorConfigMapContains("kubernetes_state_source"))
		require.True(t, mockKM.CollectorConfigMapContains("prefix: kubernetes.controlplane."))
	})

	t.Run("control plane metrics can be enabled when not on an openshift environment", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.DataCollection.Metrics.ControlPlane.Enable = true
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("prometheus_sources"))
		require.True(t, mockKM.CollectorConfigMapContains("prefix: kubernetes.controlplane."))

		require.True(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "coredns-control-plane-config", "kube-dns"))
	})

	t.Run("control plane metrics can be enabled when on an openshift environment", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.DataCollection.Metrics.ControlPlane.Enable = true
		}), []string{"security.openshift.io"})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("prometheus_sources"))
		require.True(t, mockKM.CollectorConfigMapContains("prefix: kubernetes.controlplane."))

		require.True(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "coredns-control-plane-config", "bearer_token_file"))
	})

	t.Run("control plane metrics can be disabled", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.DataCollection.Metrics.ControlPlane.Enable = false
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.CollectorConfigMapContains("prometheus_sources"))
		require.False(t, mockKM.CollectorConfigMapContains("prefix: kubernetes.controlplane."))

		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "coredns-control-plane-config"))
		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "openshift-coredns-control-plane-config"))
	})

	t.Run("control plane metrics can be disabled when on openshift", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.DataCollection.Metrics.ControlPlane.Enable = false
		}), []string{"security.openshift.io"})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.CollectorConfigMapContains("prometheus_sources"))
		require.False(t, mockKM.CollectorConfigMapContains("prefix: kubernetes.controlplane."))

		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "coredns-control-plane-config"))
		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "openshift-coredns-control-plane-config"))
	})

	t.Run("can add custom metric filters", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Filters.AllowList = []string{"allowSomeTag", "allowOtherTag"}
			w.Spec.DataCollection.Metrics.Filters.DenyList = []string{"denyAnotherTag", "denyThisTag"}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("metricAllowList:\\n    - allowSomeTag\\n\n    \\   - allowOtherTag"))
		require.True(t, mockKM.CollectorConfigMapContains("metricDenyList:\\n\n    \\   - denyAnotherTag\\n    - denyThisTag"))
	})

	t.Run("can add custom metric tag filters", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Filters.TagAllowList = map[string][]string{"env1": {"prod", "staging"}}
			w.Spec.DataCollection.Metrics.Filters.TagDenyList = map[string][]string{"env2": {"test"}}
			w.Spec.DataCollection.Metrics.Filters.TagInclude = []string{"includeSomeTag"}
			w.Spec.DataCollection.Metrics.Filters.TagExclude = []string{"excludeSomeTag"}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("metricTagAllowList:", "env1:", "- prod", "- staging"))
		require.True(t, mockKM.CollectorConfigMapContains("metricTagDenyList:", "env2:", "- test"))
		require.True(t, mockKM.CollectorConfigMapContains("tagInclude:", "includeSomeTag"))
		require.True(t, mockKM.CollectorConfigMapContains("tagExclude:", "excludeSomeTag"))
	})

	t.Run("can add custom metric filter with tag guarantee list", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Filters.TagGuaranteeList = []string{"someTagToAlwaysProtect", "someOtherTagToAlwaysProtect"}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("tagGuaranteeList:\\n\n    \\   - someTagToAlwaysProtect\\n    - someOtherTagToAlwaysProtect"))
	})

	t.Run("can add custom tags", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{"env": "non-production"}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("tags:\\n    env: non-production"))
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "200m"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "10Mi"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "256Mi"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.ClusterCollectorDeploymentContains("memory: 10Mi"))
	})

	t.Run("resources set for node collector", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "200m"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.Memory = "10Mi"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.Memory = "256Mi"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("memory: 10Mi"))
	})

	t.Run("Values from metrics.filters is propagated to default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Filters = wf.Filters{
				DenyList:  []string{"first_deny", "second_deny"},
				AllowList: []string{"first_allow", "second_allow"},
			}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		configMap, err := mockKM.GetAppliedYAML(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:2878",
		)
		require.NoError(t, err)

		configStr, found, err := unstructured.NestedString(configMap.Object, "data", "config.yaml")
		require.Equal(t, true, found)
		require.NoError(t, err)

		var configs map[string]interface{}
		err = yaml.Unmarshal([]byte(configStr), &configs)
		require.NoError(t, err)
		sinks := configs["sinks"]
		sinkArray := sinks.([]interface{})
		sinkMap := sinkArray[0].(map[string]interface{})
		filters := sinkMap["filters"].(map[string]interface{})
		require.Equal(t, 2, len(filters["metricDenyList"].([]interface{})))
		require.Equal(t, 2, len(filters["metricAllowList"].([]interface{})))
	})

	t.Run("Tags can be set for default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{"key1": "value1", "key2": "value2"}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("key1: value1", "key2: value2"))
	})

	t.Run("Empty tags map should not populate in default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.CollectorConfigMapContains("tags"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataCollection.Metrics.Enable = false
			}),
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.NodeCollectorName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "node-collector",
					},
				},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.ClusterCollectorName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "cluster-collector",
					},
				},
			},
		)
	})

	t.Run("adds the etcd secrets as a volume for the node collector and creates the etcd auto-discovery configmap when there is an etcd-certs secret in the same namespace", func(t *testing.T) {
		r, mockKM := componentScenario(
			wftest.CR(),
			nil,
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-certs",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string][]byte{
					"ca.crt":   []byte("some-ca-cert"),
					"peer.crt": []byte("some-peer-cert"),
					"peer.key": []byte("some-peer-key"),
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		daemonSet, err := mockKM.GetAppliedDaemonSet("node-collector", util.NodeCollectorName)
		require.NoError(t, err)

		volumeMountHasPath(t, daemonSet.Spec.Template.Spec.Containers[0], "etcd-certs", "/etc/etcd-certs/", "DaemonSet", daemonSet.Name)
		volumeHasSecret(t, daemonSet.Spec.Template.Spec.Volumes, "etcd-certs", "etcd-certs", "DaemonSet", daemonSet.Name)

		require.True(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "etcd-control-plane-config"))
	})

	t.Run("does not add the etcd secrets as a volume for the node collector or create the etcd auto-discovery configmap when there is no etcd-certs secret in the same namespace", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.NodeCollectorDaemonSetContains("etcd-certs"))

		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "etcd-control-plane-config"))

	})

	t.Run("does not add the etcd secrets as a volume for the node collector or create the etcd auto-discovery configmap when control plane metrics are disabled", func(t *testing.T) {
		r, mockKM := componentScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataCollection.Metrics.ControlPlane.Enable = false
			}),
			nil,
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-certs",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string][]byte{
					"ca.crt":   []byte("some-ca-cert"),
					"peer.crt": []byte("some-peer-cert"),
					"peer.key": []byte("some-peer-key"),
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.NodeCollectorDaemonSetContains("etcd-certs"))

		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "etcd-control-plane-config"))

	})
}

func TestReconcileProxy(t *testing.T) {
	t.Run("creates proxy and proxy service", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "name: WAVEFRONT_TOKEN", "containerPort: 2878"))

		require.True(t, mockKM.ProxyServiceContains("port: 2878"))
	})

	t.Run("with csp api token auth", func(t *testing.T) {

		wfCR := wftest.CR()
		r, mockKM := emptyScenario(wfCR, nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfCR.Spec.WavefrontTokenSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"csp-api-token": []byte("foo-bar"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("name: CSP_API_TOKEN"))
	})

	t.Run("with csp app oauth", func(t *testing.T) {

		wfCR := wftest.CR()
		r, mockKM := emptyScenario(wfCR, nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfCR.Spec.WavefrontTokenSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("my-app"),
				"csp-app-secret": []byte("app secret"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("name: CSP_APP_ID", "name: CSP_APP_SECRET"))
	})

	t.Run("with csp app oauth with org id", func(t *testing.T) {

		wfCR := wftest.CR()
		r, mockKM := emptyScenario(wfCR, nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfCR.Spec.WavefrontTokenSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"csp-app-id":     []byte("my-app"),
				"csp-app-secret": []byte("app secret"),
				"csp-org-id":     []byte("my-org-id"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("name: CSP_APP_ID", "name: CSP_APP_SECRET", "name: CSP_ORG_ID"))
	})

	t.Run("does not create proxy when it is configured to use an external proxy", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = false
			w.Spec.DataExport.ExternalWavefrontProxy.Url = "https://example.com"
		}), nil, wftest.Proxy(wftest.WithReplicas(0, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.False(t, mockKM.ProxyDeploymentContains())
		require.False(t, mockKM.ProxyServiceContains())
		require.Greater(t, len(mockSender.SentMetrics), 0)
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.WavefrontTokenSecret = "updatedToken"
			w.Spec.WavefrontUrl = "updatedUrl"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains(
			"name: updatedToken",
			"value: updatedUrl/api/",
		))
	})

	t.Run("updates proxy when the wavefront secret is updated", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.WavefrontTokenSecret = "some-token"
		})
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfCR.Spec.WavefrontTokenSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"token": []byte("some-token"),
			},
		}

		r, mockKM := emptyScenario(wfCR, nil, secret)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("name: WAVEFRONT_TOKEN"))
		require.False(t, mockKM.ProxyDeploymentContains("name: CSP_API_TOKEN"))

		secret.Data = map[string][]byte{
			"csp-api-token": []byte("some-csp-api-token"),
		}
		err = r.Client.Update(context.Background(), secret)
		require.NoError(t, err)

		_, err = r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
	})

	t.Run("updates proxy when the wavefront secret is updated", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.WavefrontTokenSecret = "some-token"
		})
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfCR.Spec.WavefrontTokenSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"token": []byte("some-token"),
			},
		}

		r, mockKM := emptyScenario(wfCR, nil, secret)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		proxyDep, err := mockKM.GetProxyDeployment()
		require.NoError(t, err)

		secretHash := proxyDep.Spec.Template.ObjectMeta.Annotations["secretHash"]

		require.NotEmpty(t, secretHash)

		secret.Data = map[string][]byte{
			"token": []byte("some-other-token"),
		}
		err = r.Client.Update(context.Background(), secret)
		require.NoError(t, err)

		_, err = r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		proxyDep, err = mockKM.GetProxyDeployment()
		require.NoError(t, err)

		require.NotEqual(t, secretHash, proxyDep.Spec.Template.ObjectMeta.Annotations["secretHash"])
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.MetricPort = 1234
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "pushListenerPorts", *mockKM, 1234)
		containsPortInServicePort(t, 1234, *mockKM)

		require.True(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:1234"))
	})

	t.Run("can create proxy with a user defined delta counter port", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.DeltaCounterPort = 50000
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "deltaCounterPorts", *mockKM, 50000)
		containsPortInServicePort(t, 50000, *mockKM)
	})

	t.Run("can create proxy with a user defined Wavefront tracing", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing = wf.Tracing{
				Wavefront: wf.WavefrontTracing{
					Port:             30000,
					SamplingRate:     ".1",
					SamplingDuration: 45,
				},
			}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceListenerPorts", *mockKM, 30000)
		containsPortInServicePort(t, 30000, *mockKM)

		containsProxyArg(t, "--traceSamplingRate .1", *mockKM)
		containsProxyArg(t, "--traceSamplingDuration 45", *mockKM)
	})

	t.Run("can create proxy with a user defined Jaeger distributed tracing", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing = wf.Tracing{
				Jaeger: wf.JaegerTracing{
					Port:            30001,
					GrpcPort:        14250,
					HttpPort:        30080,
					ApplicationName: "jaeger",
				},
			}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceJaegerListenerPorts", *mockKM, 30001)
		containsPortInServicePort(t, 30001, *mockKM)

		containsPortInContainers(t, "traceJaegerGrpcListenerPorts", *mockKM, 14250)
		containsPortInServicePort(t, 14250, *mockKM)

		containsPortInContainers(t, "traceJaegerHttpListenerPorts", *mockKM, 30080)
		containsPortInServicePort(t, 30080, *mockKM)

		containsProxyArg(t, "--traceJaegerApplicationName jaeger", *mockKM)
	})

	t.Run("can create proxy with a user defined ZipKin distributed tracing", func(t *testing.T) {
		wfSpec := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing.Zipkin.Port = 9411
			w.Spec.DataExport.WavefrontProxy.Tracing.Zipkin.ApplicationName = "zipkin"
		})

		r, mockKM := emptyScenario(wfSpec, nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceZipkinListenerPorts", *mockKM, 9411)
		containsPortInServicePort(t, 9411, *mockKM)

		containsProxyArg(t, "--traceZipkinApplicationName zipkin", *mockKM)
	})

	t.Run("can create proxy with OTLP enabled", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.OTLP = wf.OTLP{
				GrpcPort:                       4317,
				HttpPort:                       4318,
				ResourceAttrsOnMetricsIncluded: true,
			}
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "otlpGrpcListenerPorts", *mockKM, 4317)
		containsPortInServicePort(t, 4317, *mockKM)

		containsPortInContainers(t, "otlpHttpListenerPorts", *mockKM, 4318)
		containsPortInServicePort(t, 4318, *mockKM)

		containsProxyArg(t, "--otlpResourceAttrsOnMetricsIncluded true", *mockKM)
	})

	t.Run("can create proxy with histogram ports enabled", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Histogram.Port = 40000
			w.Spec.DataExport.WavefrontProxy.Histogram.MinutePort = 40001
			w.Spec.DataExport.WavefrontProxy.Histogram.HourPort = 40002
			w.Spec.DataExport.WavefrontProxy.Histogram.DayPort = 40003
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "histogramDistListenerPorts", *mockKM, 40000)
		containsPortInServicePort(t, 40000, *mockKM)

		containsPortInContainers(t, "histogramMinuteListenerPorts", *mockKM, 40001)
		containsPortInServicePort(t, 40001, *mockKM)

		containsPortInContainers(t, "histogramHourListenerPorts", *mockKM, 40002)
		containsPortInServicePort(t, 40002, *mockKM)

		containsPortInContainers(t, "histogramDayListenerPorts", *mockKM, 40003)
		containsPortInServicePort(t, 40003, *mockKM)
	})

	t.Run("can create proxy with a user defined proxy args", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Args = "--prefix dev \r\n --customSourceTags mySource"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--prefix dev", *mockKM)
		containsProxyArg(t, "--customSourceTags mySource", *mockKM)
	})

	t.Run("can create proxy with the default cluster_uuid and cluster preprocessor rules", func(t *testing.T) {
		clusterName := "my_cluster_name"
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) { w.Spec.ClusterName = clusterName }), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *mockKM)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		volumeMountHasPath(t, deployment.Spec.Template.Spec.Containers[0], "preprocessor", "/etc/wavefront/preprocessor", "Deployment", deployment.Name)
		volumeHasConfigMap(t, deployment, "preprocessor", "operator-proxy-preprocessor-rules-config")

		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("2878,49151:", "global:", wftest.DefaultNamespace, "ownerReferences"))

		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains(fmt.Sprintf("- rule: metrics-add-cluster-uuid\n      action: addTag\n      tag: cluster_uuid\n      value: \"%s\"", r.ClusterUUID)))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains(fmt.Sprintf("- rule: metrics-add-cluster-name\n      action: addTag\n      tag: cluster\n      value: \"%s\"", clusterName)))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("- rule: span-drop-cluster-uuid\n      action: spanDropTag\n      key: cluster_uuid"))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains(fmt.Sprintf("- rule: span-add-cluster-uuid\n      action: spanAddTag\n      key: cluster_uuid\n      value: \"%s\"", r.ClusterUUID)))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("- rule: span-drop-cluster-name\n      action: spanDropTag\n      key: cluster"))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains(fmt.Sprintf("- rule: span-add-cluster-name\n      action: spanAddTag\n      key: cluster\n      value: \"%s\"", clusterName)))
	})

	t.Run("can create proxy with the default cluster_uuid and cluster preprocessor rules for OTLP", func(t *testing.T) {
		clusterName := "my_cluster_name"
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.ClusterName = clusterName
			w.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort = 4317
			w.Spec.DataExport.WavefrontProxy.OTLP.HttpPort = 4318
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *mockKM)

		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("4317", "4318"))
	})

	t.Run("can merge 'global' user proxy rules with operator preprocessor rules", func(t *testing.T) {
		rules := "    '2878':\n      - rule: tag1\n        action: addTag\n        tag: tag1\n        value: \"true\"\n      - rule: tag2\n        action: addTag\n        tag: tag2\n        value: \"true\"\n    'global':\n      - rule: tag3\n        action: addTag\n        tag: tag3\n        value: \"true\"\n"
		r, mockKM := emptyScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Preprocessor = "user-preprocessor-rules"
			}), nil,
			&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-preprocessor-rules",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string]string{
					"rules.yaml": rules,
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *mockKM)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		volumeMountHasPath(t, deployment.Spec.Template.Spec.Containers[0], "preprocessor", "/etc/wavefront/preprocessor", "Deployment", deployment.Name)
		volumeHasConfigMap(t, deployment, "preprocessor", "operator-proxy-preprocessor-rules-config")

		configMap, err := mockKM.GetProxyPreprocessorRulesConfigMap()
		require.NoError(t, err)

		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("2878,49151:", "\"2878\":", "global:", wftest.DefaultNamespace, "ownerReferences"))

		require.Equal(t, 1, strings.Count(configMap.Data["rules.yaml"], "global"))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains(fmt.Sprintf("- rule: metrics-add-cluster-uuid\n      action: addTag\n      tag: cluster_uuid\n      value: \"%s\"", r.ClusterUUID)))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("- rule: tag1\n      action: addTag\n      tag: tag1\n      value: \"true\"\n"))
		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("- rule: tag3\n      action: addTag\n      tag: tag3\n      value: \"true\"\n"))
	})

	t.Run("can't create proxy if user preprocessor rules have a rule for cluster or cluster_uuid", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Preprocessor = "preprocessor-rules"
		})
		r, _ := emptyScenario(
			wfCR,
			nil,
			&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "preprocessor-rules",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string]string{
					"rules.yaml": "'2878':\n      - rule: tag-cluster\n        action: addTag\n        tag: cluster\n        value: \"my-cluster\"",
				},
			},
		)

		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		var reconciledWFCR wf.Wavefront

		require.NoError(t, r.Client.Get(
			context.Background(),
			util.ObjKey(wfCR.Namespace, wfCR.Name),
			&reconciledWFCR,
		))

		require.Contains(t, reconciledWFCR.Status.Status, health.Unhealthy)
		require.Equal(t, "Invalid rule configured in ConfigMap 'preprocessor-rules' on port '2878', overriding metric tag 'cluster' is disallowed", reconciledWFCR.Status.Message)
	})

	t.Run("resources set for the proxy", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "100m"
			w.Spec.DataExport.WavefrontProxy.Resources.Requests.Memory = "1Gi"
			w.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "1000m"
			w.Spec.DataExport.WavefrontProxy.Resources.Limits.Memory = "4Gi"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		require.Equal(t, "1Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "4Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("adjusting proxy replicas", func(t *testing.T) {
		t.Run("changes the number of desired replicas", func(t *testing.T) {
			r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Replicas = 2
			}), nil)

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
			require.NoError(t, err)

			require.Equal(t, int32(2), *deployment.Spec.Replicas)
		})

		t.Run("updates available replicas when based availability", func(t *testing.T) {
			r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Replicas = 2
			}), nil, wftest.Proxy(wftest.WithReplicas(2, 2)))

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			require.True(t, mockKM.NodeCollectorDaemonSetContains("proxy-available-replicas: \"2\""))
			require.True(t, mockKM.ClusterCollectorDeploymentContains("proxy-available-replicas: \"2\""))
			require.True(t, mockKM.LoggingDaemonSetContains("proxy-available-replicas: \"2\""))
		})
	})

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		}), nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"http-url":            []byte("https://myproxyhost_url:8080"),
				"basic-auth-username": []byte("myUser"),
				"basic-auth-password": []byte("myPassword"),
				"tls-root-ca-bundle":  []byte("myCert"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)
		containsProxyArg(t, "--proxyUser myUser", *mockKM)
		containsProxyArg(t, "--proxyPassword myPassword", *mockKM)

		initContainerVolumeMountHasPath(t, deployment, "http-proxy-ca", "/tmp/ca")
		volumeHasSecret(t, deployment.Spec.Template.Spec.Volumes, "http-proxy-ca", "testHttpProxySecret", "Deployment", deployment.Name)

		require.NotEmpty(t, deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
	})

	t.Run("can create proxy with HTTP configurations only contains http-url", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		}), nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"http-url": []byte("https://myproxyhost_url:8080"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)
	})

	t.Run("can create proxy with HTTP configuration where url is a service", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		}), nil, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"http-url": []byte("myproxyservice:8080"),
			},
		})

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyservice", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)

		require.NotEmpty(t, deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Enable = false
			}),
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.ProxyName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "proxy",
					},
				},
			},
		)
	})
}

func TestReconcileLogging(t *testing.T) {
	t.Run("Create logging if DataCollection.Logging.Enable is set to true", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		ds, err := mockKM.GetAppliedDaemonSet("logging", util.LoggingName)
		require.NoError(t, err)
		require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])

		require.NoError(t, err)
		require.True(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", util.LoggingName))
		require.True(t, mockKM.LoggingConfigMapContains("Proxy             http://wavefront-proxy:2878"))
		require.True(t, mockKM.LoggingConfigMapContains("URI               /logs/json_lines?f=logs_json_lines"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataCollection.Logging.Enable = false
			}),
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.LoggingName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "logging",
					},
				},
			},
		)
	})

	//TODO - Component Refactor remove once url logic is moved to build components

	t.Run("Verify external wavefront proxy url without http specified in URL", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = false
			w.Spec.DataExport.ExternalWavefrontProxy.Url = "my-proxy:8888"
		}), nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.LoggingConfigMapContains("Proxy             http://my-proxy:8888"))
	})

}

func TestReconcileAutoTracing(t *testing.T) {
	t.Run("creates Pixie components when AutoTracing is enabled", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.Enable = true
			w.Spec.ClusterName = "test-clusterName"
		})

		sslSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.PixieTLSCertsName,
				Namespace: wfCR.Spec.Namespace,
			},
			StringData: map[string]string{
				"server.key": "server-key-secret",
				"ca.crt":     "ca-crt-secret",
				"client.crt": "client-crt-secret",
				"client.key": "client-key-secret",
				"server.crt": "server-crt-secret",
			},
		}

		r, mockKM := emptyScenario(wfCR, nil, wftest.Proxy(wftest.WithReplicas(1, 1)), sslSecret)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))
		r.ClusterUUID = "12345"

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.True(t, mockKM.PixieComponentContains("batch/v1", "Job", "cert-provisioner-job"))

		require.True(t, mockKM.PixieComponentContains("apps/v1", "StatefulSet", "pl-nats"))
		require.True(t, mockKM.PixieComponentContains(
			"apps/v1", "DaemonSet", "vizier-pem",
			"name: PL_TABLE_STORE_DATA_LIMIT_MB",
			"name: PL_TABLE_STORE_HTTP_EVENTS_PERCENT",
			"name: PL_STIRLING_SOURCES",
		))
		require.True(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "kelvin"))
		require.True(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "vizier-query-broker"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets"))
		require.True(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config"))
		require.True(t, mockKM.PixieComponentContains("v1", "ServiceAccount", "metadata-service-account"))

		require.True(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config", "PL_CLUSTER_NAME: test-clusterName"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets", "cluster-name: test-clusterName"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets", "cluster-id: 12345"))

		require.True(t, mockKM.ProxyPreprocessorRulesConfigMapContains("4317"))
		containsPortInContainers(t, "otlpGrpcListenerPorts", *mockKM, 4317)
		containsPortInServicePort(t, 4317, *mockKM)
		containsProxyArg(t, "--otlpResourceAttrsOnMetricsIncluded true", *mockKM)

	})

	t.Run("does not create Pixie components when AutoTracing is not enabled", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = false
		}), nil, wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.PixieComponentContains("batch/v1", "Job", "cert-provisioner-job"))

		require.False(t, mockKM.PixieComponentContains("apps/v1", "StatefulSet", "pl-nats"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "DaemonSet", "vizier-pem"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "kelvin"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "vizier-query-broker"))
		require.False(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets"))
		require.False(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config"))
		require.False(t, mockKM.PixieComponentContains("v1", "ServiceAccount", "metadata-service-account"))

		doesNotContainPortInContainers(t, "otlpGrpcListenerPorts", *mockKM, 4317)
		doesNotContainPortInServicePort(t, 4317, *mockKM)
	})

	t.Run("does not deploy OpApps pxl scripts when canExportAutotracingScripts is false", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
		}), nil, wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-cluster-spans-script"))
		require.False(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-egress-spans-script"))
		require.False(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-ingress-spans-script"))
	})

	t.Run("does deploy OpApps pxl scripts when canExportAutotracingScripts is true", func(t *testing.T) {
		daemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vizier-pem",
				Namespace: wftest.DefaultNamespace,
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}
		r, mockKM := emptyScenario(wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Autotracing.Enable = true
		}), nil, wftest.Proxy(wftest.WithReplicas(1, 1)), daemonset)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.True(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-cluster-spans-script"))
		require.True(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-egress-spans-script"))
		require.True(t, mockKM.AutotracingComponentContains("v1", "ConfigMap", "wavefront-ingress-spans-script"))
	})
}

func TestReconcileHubPixie(t *testing.T) {
	t.Run("creates Pixie components when Hub Pixie is enabled", func(t *testing.T) {
		wfCR := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.ClusterName = "test-clusterName"
			w.Spec.Experimental.Hub.Enable = true
			w.Spec.Experimental.Hub.Pixie.Enable = true
		})

		sslSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.PixieTLSCertsName,
				Namespace: wfCR.Spec.Namespace,
			},
			StringData: map[string]string{
				"server.key": "server-key-secret",
				"ca.crt":     "ca-crt-secret",
				"client.crt": "client-crt-secret",
				"client.key": "client-key-secret",
				"server.crt": "server-crt-secret",
			},
		}

		r, mockKM := emptyScenario(wfCR, nil, sslSecret)

		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))
		r.ClusterUUID = "12345"

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.True(t, mockKM.PixieComponentContains("apps/v1", "StatefulSet", "pl-nats"))
		require.True(t, mockKM.PixieComponentContains("apps/v1", "DaemonSet", "vizier-pem"))
		require.True(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "kelvin"))
		require.True(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "vizier-query-broker"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets"))
		require.True(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config"))
		require.True(t, mockKM.PixieComponentContains("v1", "ServiceAccount", "metadata-service-account"))

		require.True(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config", "PL_CLUSTER_NAME: test-clusterName"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets", "cluster-name: test-clusterName"))
		require.True(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets", "cluster-id: 12345"))

		require.True(t, mockKM.PixieComponentContains("batch/v1", "Job", "cert-provisioner-job"))

		require.False(t, mockKM.ProxyDeploymentContains(""))
	})

	t.Run("does not create Pixie components when Hub Pixie is not enabled", func(t *testing.T) {
		wfCR := wftest.NothingEnabledCR(func(wfCR *wf.Wavefront) {
			wfCR.Spec.Experimental.Hub.Enable = true
			wfCR.Spec.Experimental.Hub.Pixie.Enable = false
		})
		r, mockKM := emptyScenario(wfCR, nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.PixieComponentContains("apps/v1", "StatefulSet", "pl-nats"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "DaemonSet", "vizier-pem"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "kelvin"))
		require.False(t, mockKM.PixieComponentContains("apps/v1", "Deployment", "vizier-query-broker"))
		require.False(t, mockKM.PixieComponentContains("v1", "Secret", "pl-cluster-secrets"))
		require.False(t, mockKM.PixieComponentContains("v1", "Secret", "pl-deploy-secrets"))
		require.False(t, mockKM.PixieComponentContains("v1", "ConfigMap", "pl-cloud-config"))
		require.False(t, mockKM.PixieComponentContains("v1", "ServiceAccount", "metadata-service-account"))
		require.False(t, mockKM.PixieComponentContains("batch/v1", "Job", "cert-provisioner-job"))
	})
}

func TestReconcileInsightsByCR(t *testing.T) {
	t.Run("can enable K8s events only", func(t *testing.T) {
		cr := wftest.CR(func(wavefront *wf.Wavefront) {
			wavefront.Spec.Experimental.Insights.Enable = true
			wavefront.Spec.Experimental.Insights.IngestionUrl = "https://example.com"
			wavefront.Spec.DataExport.WavefrontProxy.Enable = false
			wavefront.Spec.DataCollection.Metrics.Enable = false
			wavefront.Spec.DataCollection.Logging.Enable = false
		})
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.InsightsSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"ingestion-token": []byte("anything"),
			},
		}
		r, mockKM := componentScenario(cr, nil, secret)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ConfigMapContains("k8s-events-only-wavefront-collector-config", "externalEndpointURL: \"https://example.com\""))
		require.True(t, mockKM.ConfigMapContains("k8s-events-only-wavefront-collector-config", "enableEvents: true"))
		require.True(t, mockKM.ConfigMapContains("k8s-events-only-wavefront-collector-config", "events:", "filters:", "tagAllowList:", "important:", "- \"true\"", "tagDenyList:", "kind:", "- \"Job\""))

		require.False(t, mockKM.ConfigMapContains("k8s-events-only-wavefront-collector-config", "proxyAddress", "kubeletHttps", "kubernetes_state_source"))
		require.False(t, mockKM.CollectorConfigMapContains())

		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: K8S_EVENTS_ENDPOINT_TOKEN"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: insights-secret"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("key: ingestion-token"))
		require.True(t, mockKM.CollectorServiceAccountContains())
		require.False(t, mockKM.NodeCollectorDaemonSetContains())
		require.False(t, mockKM.LoggingDaemonSetContains())
		require.False(t, mockKM.ProxyServiceContains())
		require.False(t, mockKM.ProxyDeploymentContains())
	})

	t.Run("can enable external K8s events and WF metrics with yaml spec", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Insights.Enable = true
			w.Spec.Experimental.Insights.IngestionUrl = "https://example.com"
		})
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.InsightsSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"ingestion-token": []byte("anything"),
			},
		}
		r, mockKM := componentScenario(cr, nil, secret)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("externalEndpointURL: \\\"https://example.com\\\""))
		require.True(t, mockKM.CollectorConfigMapContains("enableEvents:\n    false"))
		require.True(t, mockKM.CollectorConfigMapContains("type: \\\"external\\\"", "enableEvents: true"))
		require.True(t, mockKM.CollectorConfigMapContains("events:\\n\n    \\ filters:\\n    tagAllowListSets:\\n    - type:\\n      - \\\"Warning\\\"\\n    - type:\\n\n    \\     - \\\"Normal\\\"\\n      kind:\\n      - \\\"Pod\\\"\\n      reason:\\n      - \\\"Backoff\\\""))
		require.True(t, mockKM.CollectorConfigMapContains("proxyAddress: wavefront-proxy:2878"))

		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: K8S_EVENTS_ENDPOINT_TOKEN"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("key: ingestion-token"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: insights-secret"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains("name: K8S_EVENTS_ENDPOINT_TOKEN"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains("key: ingestion-token"))
		require.True(t, mockKM.ProxyDeploymentContains("name: WAVEFRONT_TOKEN", "key: token"))
	})

	t.Run("can enable external K8s events only with resource limits", func(t *testing.T) {
		cr := &wf.Wavefront{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront",
				Namespace: wftest.DefaultNamespace,
			},
			Spec: wf.WavefrontSpec{
				ClusterName: "a-cluster",
				ClusterSize: wf.ClusterSizeMedium,
				DataCollection: wf.DataCollection{
					Metrics: wf.Metrics{
						Enable: false,
						ClusterCollector: wf.Collector{
							Resources: wf.Resources{
								Requests: wf.Resource{CPU: "100m", Memory: "10Mi"},
								Limits:   wf.Resource{CPU: "250Mi", Memory: "200Mi"},
							},
						},
					},
				},
				Experimental: wf.Experimental{Insights: wf.Insights{
					Enable:       true,
					IngestionUrl: "https://example.com",
				}},
			},
		}

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.InsightsSecret,
				Namespace: wftest.DefaultNamespace,
			},
			Data: map[string][]byte{
				"ingestion-token": []byte("ignored"),
			},
		}
		r, mockKM := componentScenario(cr, nil, secret)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ConfigMapContains(
			"k8s-events-only-wavefront-collector-config",
			"externalEndpointURL: \"https://example.com\""))

		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: K8S_EVENTS_ENDPOINT_TOKEN"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: insights-secret"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("key: ingestion-token"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("name: k8s-events-only-wavefront-collector-config"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("limits:\n            cpu: 250Mi\n            memory: 200Mi"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("requests:\n            cpu: 100m\n            memory: 10Mi"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains())
		require.False(t, mockKM.ProxyDeploymentContains())
	})
}

func VersionSent(mockSender *testhelper.MockSender) float64 {
	var versionSent float64
	for _, m := range mockSender.SentMetrics {
		if m.Name == "kubernetes.observability.version" {
			versionSent = m.Value
		}
	}
	return versionSent
}

func StatusMetricsSent(mockSender *testhelper.MockSender) int {
	var statusMetricsSent int
	for _, m := range mockSender.SentMetrics {
		if strings.HasSuffix(m.Name, ".status") {
			statusMetricsSent += 1
		}
	}
	return statusMetricsSent
}

func CanBeDisabled(t *testing.T, wfCR *wf.Wavefront, existingResources ...runtime.Object) {
	t.Run("on CR creation", func(t *testing.T) {
		r, mockKM := emptyScenario(wfCR, nil)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			require.Falsef(t, mockKM.AppliedContains(
				gvk.GroupVersion().String(), gvk.Kind,
				objMeta.GetLabels()["app.kubernetes.io/name"],
				objMeta.GetLabels()["app.kubernetes.io/component"],
				objMeta.GetName(),
			), "%s/%s should not have been applied", gvk.Kind, objMeta.GetName())
		}
	})

	t.Run("on CR update", func(t *testing.T) {
		r, mockKM := emptyScenario(wfCR, nil, existingResources...)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			require.True(t, mockKM.DeletedContains(
				gvk.GroupVersion().String(), gvk.Kind,
				objMeta.GetLabels()["app.kubernetes.io/name"],
				objMeta.GetLabels()["app.kubernetes.io/component"],
				objMeta.GetName(),
			), "%s/%s should have been deleted", gvk.Kind, objMeta.GetName())
		}
	})
}

func volumeMountHasPath(t *testing.T, container v1.Container, name, path string, objectKind string, objectName string) {
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name == name {
			require.Equal(t, path, volumeMount.MountPath)
			return
		}
	}
	require.Failf(t, "could not find volume mount", "could not find volume mount named %s on %s %s", name, objectKind, objectName)
}

func volumeHasConfigMap(t *testing.T, deployment appsv1.Deployment, name string, configMapName string) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == name {
			require.Equal(t, configMapName, volume.ConfigMap.Name)
			return
		}
	}
	require.Failf(t, "could not find volume", "could not find volume named %s on deployment %s", name, deployment.Name)
}

func volumeHasSecret(t *testing.T, volumes []v1.Volume, name string, secretName string, objectKind string, objectName string) {
	for _, volume := range volumes {
		if volume.Name == name {
			require.Equal(t, secretName, volume.Secret.SecretName)
			return
		}
	}
	require.Failf(t, "could not find secret", "could not find secret named %s on %s %s", name, objectKind, objectName)
}

func initContainerVolumeMountHasPath(t *testing.T, deployment appsv1.Deployment, name, path string) {
	for _, volumeMount := range deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts {
		if volumeMount.Name == name {
			require.Equal(t, path, volumeMount.MountPath)
			return
		}
	}
	require.Failf(t, "could not find init container volume mount", "could not find init container volume mount named %s on deployment %s", name, deployment.Name)
}

func containsPortInServicePort(t *testing.T, port int32, mockKM testhelper.MockKubernetesManager) {
	serviceYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
	require.NoError(t, err)

	var service v1.Service

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(serviceYAMLUnstructured.Object, &service)
	require.NoError(t, err)

	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			return
		}
	}
	require.Fail(t, fmt.Sprintf("Did not find the port: %d", port))
}

func doesNotContainPortInServicePort(t *testing.T, port int32, mockKM testhelper.MockKubernetesManager) {
	serviceYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
	require.NoError(t, err)

	var service v1.Service

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(serviceYAMLUnstructured.Object, &service)
	require.NoError(t, err)

	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			require.Fail(t, fmt.Sprintf("Should not find port: %d", port))
		}
	}
}

func containsPortInContainers(t *testing.T, proxyArgName string, mockKM testhelper.MockKubernetesManager, port int32) bool {
	t.Helper()
	deploymentYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		util.ProxyName,
	)
	require.NoError(t, err)

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	require.NoError(t, err)

	foundPort := false
	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if containerPort.ContainerPort == port {
			foundPort = true
			break
		}
	}
	require.True(t, foundPort, fmt.Sprintf("Did not find the port: %d", port))

	proxyArgsEnvValue := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	require.Contains(t, proxyArgsEnvValue, fmt.Sprintf("--%s %d", proxyArgName, port))
	return true
}

func doesNotContainPortInContainers(t *testing.T, proxyArgName string, mockKM testhelper.MockKubernetesManager, port int32) bool {
	t.Helper()
	deploymentYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		util.ProxyName,
	)
	require.NoError(t, err)

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	require.NoError(t, err)

	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if containerPort.ContainerPort == port {
			require.Fail(t, fmt.Sprintf("Should not find port: %d", port))
		}
	}

	proxyArgsEnvValue := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	require.NotContains(t, proxyArgsEnvValue, fmt.Sprintf("--%s %d", proxyArgName, port))
	return true
}

func getEnvValueForName(envs []v1.EnvVar, name string) string {
	for _, envVar := range envs {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}

func containsProxyArg(t *testing.T, proxyArg string, mockKM testhelper.MockKubernetesManager) {
	deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
	require.NoError(t, err)

	value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	require.Contains(t, value, proxyArg)
}

func emptyScenario(wfCR *wf.Wavefront, apiGroups []string, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *testhelper.MockKubernetesManager) {
	s := scheme.Scheme
	s.AddKnownTypes(wf.GroupVersion, &wf.Wavefront{})

	namespace := wftest.DefaultNamespace
	if wfCR != nil {
		namespace = wfCR.Namespace
	}

	if !containsObject(initObjs, operatorInNamespace(namespace)) {
		operator := wftest.Operator()
		operator.SetNamespace(namespace)
		initObjs = append(initObjs, operator)
	}

	clientBuilder := fake.NewClientBuilder().WithScheme(s)
	if wfCR != nil {
		clientBuilder = clientBuilder.WithObjects(wfCR)
	}
	clientBuilder = clientBuilder.WithRuntimeObjects(initObjs...)
	objClient := clientBuilder.Build()

	mockKM := testhelper.NewMockKubernetesManager()
	mockDiscoveryClient := testhelper.NewMockKubernetesDiscoveryClient(apiGroups)

	r := &controllers.WavefrontReconciler{
		Versions: controllers.Versions{
			CollectorVersion: "12.34.56",
			ProxyVersion:     "99.99.99",
			LoggingVersion:   "99.99.99",
			OperatorVersion:  "99.99.99",
		},
		Client:              objClient,
		ComponentsDeployDir: os.DirFS(filepath.Join("..", components.DeployDir)),
		KubernetesManager:   mockKM,
		DiscoveryClient:     mockDiscoveryClient,
		MetricConnection:    metric.NewConnection(testhelper.StubSenderFactory(nil, nil)),
		ClusterUUID:         "cluster-uuid",
	}

	return r, mockKM
}

func componentScenario(wfCR *wf.Wavefront, apiGroups []string, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *testhelper.MockKubernetesManager) {
	if !containsObject(initObjs, proxyInNamespace(wfCR.Namespace)) {
		proxy := wftest.Proxy(wftest.WithReplicas(1, 1))
		proxy.SetNamespace(wfCR.Namespace)
		initObjs = append(initObjs, proxy)
	}
	return emptyScenario(wfCR, apiGroups, initObjs...)
}

func operatorInNamespace(namespace string) func(obj client.Object) bool {
	return func(obj client.Object) bool {
		labels := obj.GetLabels()
		return obj.GetNamespace() == namespace &&
			labels["app.kubernetes.io/name"] == "wavefront" &&
			labels["app.kubernetes.io/component"] == "controller-manager"
	}
}

func proxyInNamespace(namespace string) func(obj client.Object) bool {
	return func(obj client.Object) bool {
		labels := obj.GetLabels()
		return obj.GetNamespace() == namespace &&
			labels["app.kubernetes.io/name"] == "wavefront" &&
			labels["app.kubernetes.io/component"] == "proxy"
	}
}

func containsObject(runtimeObjs []runtime.Object, matches func(obj client.Object) bool) bool {
	for _, runtimeObj := range runtimeObjs {
		obj, ok := runtimeObj.(client.Object)
		if !ok {
			continue
		}
		if matches(obj) {
			return true
		}
	}
	return false
}

func defaultRequest() reconcile.Request {
	return reconcile.Request{NamespacedName: util.ObjKey(wftest.DefaultNamespace, "wavefront")}
}
