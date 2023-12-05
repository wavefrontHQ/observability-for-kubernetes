package wftest

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CROption func(w *wf.Wavefront)

func CR(options ...CROption) *wf.Wavefront {
	defaults := func(w *wf.Wavefront) {
		w.Spec.WavefrontUrl = "testWavefrontUrl"
		w.Spec.WavefrontTokenSecret = "testToken"
		w.Spec.Namespace = DefaultNamespace
		w.Spec.ImageRegistry = DefaultImageRegistry
		w.Spec.DataCollection = wf.DataCollection{
			Metrics: wf.Metrics{
				Enable: true,
				ControlPlane: wf.ControlPlane{
					Enable: true,
				},
				ClusterCollector: wf.Collector{Resources: common.ContainerResources{
					Requests: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
					Limits: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
				}},
				NodeCollector: wf.Collector{Resources: common.ContainerResources{
					Requests: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
					Limits: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
				}},
				CollectorVersion: "1.28.0",
			},
			Logging: wf.Logging{
				Enable:         true,
				LoggingVersion: "2.1.6",
				Resources: common.ContainerResources{
					Requests: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
					Limits: common.ContainerResource{
						CPU:    "100m",
						Memory: "50Mi",
					},
				},
			},
		}
		w.Spec.DataExport = wf.DataExport{
			WavefrontProxy: wf.WavefrontProxy{
				Enable:     true,
				MetricPort: 2878,
				Resources: common.ContainerResources{
					Requests: common.ContainerResource{
						CPU:              "100m",
						Memory:           "1Gi",
						EphemeralStorage: "2Gi",
					},
					Limits: common.ContainerResource{
						CPU:              "1",
						Memory:           "4Gi",
						EphemeralStorage: "8Gi",
					},
				},
				ProxyVersion: "13.1",
			},
		}
	}
	return NothingEnabledCR(append([]CROption{defaults}, options...)...)
}

func NothingEnabledCR(options ...CROption) *wf.Wavefront {
	cr := &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Wavefront",
			APIVersion: wf.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront",
			Namespace: DefaultNamespace,
		},
		Spec: wf.WavefrontSpec{
			ClusterSize:          wf.ClusterSizeMedium,
			ClusterName:          "testClusterName",
			ControllerManagerUID: "controller-manager-uid",
			ClusterUUID:          "cluster-uuid",
			Namespace:            DefaultNamespace,
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}
