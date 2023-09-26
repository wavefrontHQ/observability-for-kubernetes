package wftest

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CROption func(w *wf.Wavefront)

func CR(options ...CROption) *wf.Wavefront {
	cr := &wf.Wavefront{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront",
			Namespace: DefaultNamespace,
		},
		Spec: wf.WavefrontSpec{
			ClusterName:          DefaultClusterName,
			WavefrontUrl:         "testWavefrontUrl",
			WavefrontTokenSecret: "testToken",
			Namespace:            DefaultNamespace,
			ControllerManagerUID: "controller-manager-uid",
			ClusterUUID:          "cluster-uuid",
			ImageRegistry:        DefaultImageRegistry,
			DataCollection: wf.DataCollection{
				Metrics: wf.Metrics{
					Enable: true,
					ControlPlane: wf.ControlPlane{
						Enable: true,
					},
				},
				Logging: wf.Logging{
					Enable:         true,
					LoggingVersion: "2.1.6",
					Resources: wf.Resources{
						Requests: wf.Resource{},
						Limits: wf.Resource{
							CPU:    "100Mi",
							Memory: "50Mi",
						},
					},
				},
			},
			DataExport: wf.DataExport{
				WavefrontProxy: wf.WavefrontProxy{
					Enable:     true,
					MetricPort: 2878,
					Resources: wf.Resources{
						Requests: wf.Resource{},
						Limits: wf.Resource{
							CPU:    "100Mi",
							Memory: "50Mi",
						},
					},
				},
			},
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}

func NothingEnabledCR(options ...CROption) *wf.Wavefront {
	cr := &wf.Wavefront{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront",
			Namespace: DefaultNamespace,
		},
		Spec: wf.WavefrontSpec{
			ClusterName:          "testClusterName",
			ControllerManagerUID: "controller-manager-uid",
			ClusterUUID:          "cluster-uuid",
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}
