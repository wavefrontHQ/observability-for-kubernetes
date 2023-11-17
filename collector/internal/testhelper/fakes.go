package testhelper

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	v1 "k8s.io/api/core/v1"
)

type testWorkloadCache struct {
	workloadName string
	workloadKind string
	nodeName     string
	pod          *v1.Pod
}

func NewFakeWorkloadCache(workloadName, workloadKind, nodeName string, pod *v1.Pod) util.WorkloadCache {
	return testWorkloadCache{
		workloadName: workloadName,
		workloadKind: workloadKind,
		nodeName:     nodeName,
		pod:          pod,
	}
}

func NewEmptyFakeWorkloadCache() util.WorkloadCache {
	return testWorkloadCache{}
}

func (wc testWorkloadCache) GetPod(podName, ns string) (pod *v1.Pod, err error) {
	return wc.pod, err
}

func (wc testWorkloadCache) GetWorkloadForPodName(podName, ns string) (name, kind, nodeName string) {
	return wc.workloadName, wc.workloadKind, wc.nodeName
}

func (wc testWorkloadCache) GetWorkloadForPod(pod *v1.Pod) (string, string) {
	return wc.workloadName, wc.workloadKind
}
