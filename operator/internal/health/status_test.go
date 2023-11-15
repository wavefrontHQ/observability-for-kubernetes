package health

import (
	"fmt"
	"testing"
	"time"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileReportHealthStatus(t *testing.T) {
	t.Run("report health status when no components are running", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = true
			w.Spec.DataCollection.Metrics.Enable = true
		})
		fakeClient := setup()

		status := GenerateWavefrontStatus(fakeClient, cr)

		assert.Equal(t, Unhealthy, status.Status)
		assert.Equal(t, "", status.Message)
		proxyStatus := getComponentStatusWithName(util.ProxyName, status.ResourceStatuses)
		assert.False(t, proxyStatus.Healthy)
		assert.Equal(t, NotRunning, proxyStatus.Status)

		clusterCollectorStatus := getComponentStatusWithName(util.ClusterCollectorName, status.ResourceStatuses)
		assert.False(t, clusterCollectorStatus.Healthy)
		assert.Equal(t, NotRunning, clusterCollectorStatus.Status)

		nodeCollectorStatus := getComponentStatusWithName(util.NodeCollectorName, status.ResourceStatuses)
		assert.False(t, nodeCollectorStatus.Healthy)
		assert.Equal(t, NotRunning, nodeCollectorStatus.Status)
	})

	t.Run("report health status as installing until MaxInstallTime has elapsed", func(t *testing.T) {
		cr := wftest.NothingEnabledCR()
		fakeClient := setup()

		cr.CreationTimestamp.Time = time.Now().Add(-MaxInstallTime).Add(time.Second * 10)
		cr.Spec.DataCollection.Metrics.Enable = true
		cr.Spec.DataExport.WavefrontProxy.Enable = true
		status := GenerateWavefrontStatus(fakeClient, cr)

		assert.Equal(t, Installing, status.Status)
		assert.Equal(t, "Installing components", status.Message)
		for _, resourceStatus := range status.ResourceStatuses {
			assert.True(t, resourceStatus.Installing)
		}
	})

	t.Run("report health status as unhealthy after MaxInstallTime has elapsed", func(t *testing.T) {
		cr := wftest.NothingEnabledCR()
		fakeClient := setup()

		cr.CreationTimestamp.Time = pastMaxInstallTime().Add(time.Second * 10)
		cr.Spec.DataCollection.Metrics.Enable = true
		cr.Spec.DataExport.WavefrontProxy.Enable = true
		status := GenerateWavefrontStatus(fakeClient, cr)

		assert.Equal(t, Unhealthy, status.Status)
	})

	t.Run("logging", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "logging",
		}
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Enable = true
		})
		apps := []client.Object{
			daemonSet(labels, util.LoggingName),
		}
		RespondsToAppStatuses(t, cr, apps)
		RespondsToAppsOOMKilled(t, cr, apps, apps)
	})

	t.Run("metrics", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "collector",
		}
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
		})
		apps := []client.Object{
			daemonSet(labels, util.NodeCollectorName),
			deployment(labels, util.ClusterCollectorName),
		}
		RespondsToAppStatuses(t, cr, apps)
		RespondsToAppsOOMKilled(t, cr, apps, apps)
	})

	t.Run("proxy", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "proxy",
		}
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = true
		})
		apps := []client.Object{deployment(labels, util.ProxyName)}

		RespondsToAppStatuses(t, cr, apps)
		RespondsToAppsOOMKilled(t, cr, apps, apps)
	})

	t.Run("controller-manager", func(t *testing.T) {
		cr := wftest.NothingEnabledCR()
		apps := []client.Object{operatorDeployment(1, 1)}
		RespondsToAppsOOMKilled(t, cr, apps, apps)
	})

	t.Run("pixie", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "pixie",
		}
		appsWithoutPEM := []client.Object{
			deployment(labels, util.PixieKelvinName),
			statefulSet(labels, util.PixieNatsName),
			statefulSet(labels, util.PixieVizierMetadataName),
			deployment(labels, util.PixieVizierQueryBrokerName),
		}
		apps := append([]client.Object{daemonSet(labels, util.PixieVizierPEMName)}, appsWithoutPEM...)

		t.Run("hub", func(t *testing.T) {
			cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
				w.Spec.Experimental.Hub.Pixie.Enable = true
			})
			RespondsToAppStatuses(t, cr, apps)

			RespondsToAppsOOMKilled(t, cr, apps, appsWithoutPEM)

			t.Run("healthy when only the PEMs have been OOM killed in the last five minutes", func(t *testing.T) {
				ourPod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: wftest.DefaultNamespace,
						Labels:    apps[0].GetLabels(),
						Name:      "vizier-pem-lxrhq",
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							LastTerminationState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode:   137, // OOMKilled
									FinishedAt: metav1.Time{Time: time.Now()},
								},
							},
						}},
					},
				}
				client := setup(append([]client.Object{&ourPod}, apps...)...)

				status := GenerateWavefrontStatus(client, cr)

				require.Equal(t, Healthy, status.Status)
				require.Contains(t, status.Message, "All components are healthy")
				for _, resourceStatus := range status.ResourceStatuses {
					if resourceStatus.Name == util.LoggingName {
						require.Equal(t, Unhealthy, resourceStatus.Status)
					}
				}
			})
		})

		t.Run("autotracing", func(t *testing.T) {
			cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
				w.Spec.Experimental.Autotracing.Enable = true
			})
			RespondsToAppStatuses(t, cr, apps)

			RespondsToAppsOOMKilled(t, cr, apps, appsWithoutPEM)

			t.Run("healthy when only the PEMs have been OOM killed in the last five minutes", func(t *testing.T) {
				ourPod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: wftest.DefaultNamespace,
						Labels:    apps[0].GetLabels(),
						Name:      "vizier-pem-lxrhq",
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							LastTerminationState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode:   137, // OOMKilled
									FinishedAt: metav1.Time{Time: time.Now()},
								},
							},
						}},
					},
				}
				client := setup(append([]client.Object{&ourPod}, apps...)...)

				status := GenerateWavefrontStatus(client, cr)

				require.Equal(t, Healthy, status.Status)
				require.Contains(t, status.Message, "All components are healthy")
				for _, resourceStatus := range status.ResourceStatuses {
					if resourceStatus.Name == util.LoggingName {
						require.Equal(t, Unhealthy, resourceStatus.Status)
					}
				}
			})
		})
	})
}

func RespondsToAppsOOMKilled(t *testing.T, cr *wf.Wavefront, init []client.Object, appsToVerify []client.Object) {
	t.Run("apps OOM killed", func(t *testing.T) {
		for _, app := range appsToVerify {
			t.Run(app.GetName()+" resource", func(t *testing.T) {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: wftest.DefaultNamespace,
						Labels:    app.GetLabels(),
						Name:      fmt.Sprintf("%s-some-random-chars", app.GetName()),
					},
				}
				t.Run("unhealthy when it has been OOM killed in the last five minutes", func(t *testing.T) {
					ourPod := *pod
					ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode:   137, // OOMKilled
								FinishedAt: metav1.Time{Time: time.Now()},
							},
						},
					}}
					client := setup(append([]client.Object{&ourPod}, init...)...)

					status := GenerateWavefrontStatus(client, cr)

					require.Equal(t, Unhealthy, status.Status)
					require.Contains(t, status.Message, "OOMKilled in the last 5m")
					requireResourceStatusEqual(t, status, app.GetName(), Unhealthy)
				})

				t.Run("healthy when it has not been OOM killed in the last five minutes", func(t *testing.T) {
					ourPod := *pod
					ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode:   137, // OOMKilled
								FinishedAt: metav1.Time{Time: time.Now().Add(-OOMTimeout).Add(-10 * time.Second)},
							},
						},
					}}
					client := setup(append([]client.Object{&ourPod}, init...)...)

					status := GenerateWavefrontStatus(client, cr)

					require.Equal(t, Healthy, status.Status)
					requireResourceStatusEqual(t, status, app.GetName(), "Running (1/1)")
				})

				t.Run("handles when it has not been terminated", func(t *testing.T) {
					ourPod := *pod
					ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{}}
					client := setup(append([]client.Object{&ourPod}, init...)...)

					require.NotPanics(t, func() {
						GenerateWavefrontStatus(client, cr)
					})
				})
			})
		}
	})
}

func requireResourceStatusEqual(t *testing.T, status wf.WavefrontStatus, resourceName string, expectedStatus string) {
	t.Helper()
	for _, resourceStatus := range status.ResourceStatuses {
		if resourceStatus.Name == resourceName {
			require.Equal(t, expectedStatus, resourceStatus.Status)
		}
	}
}

func daemonSet(labels map[string]string, name string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wftest.DefaultNamespace,
			Name:      name,
			Labels:    labels,
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        1,
			DesiredNumberScheduled: 1,
		},
	}
}

func deployment(labels map[string]string, name string) *appsv1.Deployment {
	desired := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wftest.DefaultNamespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &desired,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
}

func statefulSet(labels map[string]string, name string) *appsv1.StatefulSet {
	desired := int32(1)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wftest.DefaultNamespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &desired,
		},
		Status: appsv1.StatefulSetStatus{
			AvailableReplicas: 1,
		},
	}
}

func RespondsToAppStatuses(t *testing.T, cr *wf.Wavefront, apps []client.Object) {
	t.Run("app statuses", func(t *testing.T) {
		t.Run("reports healthy when all components are running", func(t *testing.T) {
			fakeClient := setup(apps...)

			status := GenerateWavefrontStatus(fakeClient, cr)

			require.Equal(t, Healthy, status.Status)
			assert.Equal(t, "All components are healthy", status.Message)
			for _, app := range apps {
				requireResourceStatusEqual(t, status, app.GetName(), "Running (1/1)")
			}
		})

		for i, subject := range apps {
			initApps := make([]client.Object, len(apps)-1)
			copy(initApps[0:i], apps[0:i])
			if i+1 < len(apps) {
				copy(initApps[i:], apps[i+1:])
			}
			t.Run(subject.GetName()+" resource", func(t *testing.T) {
				t.Run("reports not running", func(t *testing.T) {
					fakeClient := setup(initApps...)

					status := GenerateWavefrontStatus(fakeClient, cr)

					require.Equal(t, Unhealthy, status.Status)
					require.Contains(t, status.Message, "")
					requireResourceStatusEqual(t, status, subject.GetName(), NotRunning)
				})

				t.Run("reports not enough running", func(t *testing.T) {
					fakeClient := setup(append([]client.Object{setReplicas(subject, 0, 1)}, initApps...)...)

					status := GenerateWavefrontStatus(fakeClient, cr)

					require.Equal(t, Unhealthy, status.Status)
					name := subject.GetName()
					require.Contains(t, status.Message, fmt.Sprintf("not enough instances of %s are running (0/1)", name))
					requireResourceStatusEqual(t, status, subject.GetName(), "Running (0/1)")
				})
			})
		}
	})
}

func setReplicas(app client.Object, available, desired int32) client.Object {
	switch newApp := app.DeepCopyObject().(type) {
	case *appsv1.DaemonSet:
		newApp.Status.NumberAvailable = available
		newApp.Status.DesiredNumberScheduled = desired
		return newApp
	case *appsv1.Deployment:
		newApp.Status.AvailableReplicas = available
		newApp.Spec.Replicas = &desired
		return newApp
	case *appsv1.StatefulSet:
		newApp.Status.AvailableReplicas = available
		newApp.Spec.Replicas = &desired
		return newApp
	default:
		panic(fmt.Sprintf("unhandled app kind %s", app.GetObjectKind().GroupVersionKind().Kind))
	}
}

func pastMaxInstallTime() time.Time {
	return time.Now().Add(-MaxInstallTime).Add(-time.Second * 10)
}

func setup(initObjs ...client.Object) client.Client {
	foundOperator := false
	for _, initObj := range initObjs {
		if initObj.GetName() == util.OperatorName {
			foundOperator = true
		}
	}
	if !foundOperator {
		initObjs = append(initObjs, operatorDeployment(1, 1))
	}
	return fake.NewClientBuilder().WithObjects(initObjs...).Build()
}

func operatorDeployment(ready, desired int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wftest.DefaultNamespace,
			Name:      util.OperatorName,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "controller-manager",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &desired,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: ready,
		},
	}
}

func getComponentStatusWithName(name string, componentStatuses []wf.ResourceStatus) wf.ResourceStatus {
	for _, componentStatus := range componentStatuses {
		if componentStatus.Name == name {
			return componentStatus
		}
	}
	return wf.ResourceStatus{}
}
