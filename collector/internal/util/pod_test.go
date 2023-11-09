package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const podStuckInTerminating = `
Version: v1
kind: Pod
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{},"finalizers":["kubernetes"],"labels":{"exclude-me":"true","name":"pod-stuck-in-terminating"},"name":"pod-stuck-in-terminating","namespace":"collector-targets"},"spec":{"containers":[{"args":["/bin/sh","-c","i=0; while true; do echo \"$i: $(date)\\n\"; echo \"nextline\"; i=$((i+1)); sleep 1; done"],"image":"projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest","name":"pod-stuck-in-terminating","resources":{"limits":{"cpu":"75m","ephemeral-storage":"512Mi","memory":"75Mi"},"requests":{"cpu":"50m","ephemeral-storage":"256Mi","memory":"50Mi"}}}]}}
  creationTimestamp: "2023-11-09T15:16:48Z"
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-11-09T15:16:59Z"
  finalizers:
  - kubernetes
  labels:
    exclude-me: "true"
    name: pod-stuck-in-terminating
  name: pod-stuck-in-terminating
  namespace: collector-targets
  resourceVersion: "1721"
  uid: 6fe211cf-a823-42fe-b574-48eb94323b9a
spec:
  containers:
  - args:
    - /bin/sh
    - -c
    - 'i=0; while true; do echo "$i: $(date)\n"; echo "nextline"; i=$((i+1)); sleep
      1; done'
    image: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest
    imagePullPolicy: Always
    name: pod-stuck-in-terminating
    resources:
      limits:
        cpu: 75m
        ephemeral-storage: 512Mi
        memory: 75Mi
      requests:
        cpu: 50m
        ephemeral-storage: 256Mi
        memory: 50Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: kube-api-access-bmkfv
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: kube-api-access-bmkfv
    projected:
      defaultMode: 420
      sources:
      - serviceAccountToken:
          expirationSeconds: 3607
          path: token
      - configMap:
          items:
          - key: ca.crt
            path: ca.crt
          name: kube-root-ca.crt
      - downwardAPI:
          items:
          - fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
            path: namespace
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:16:48Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:17:31Z"
    message: 'containers with unready status: [pod-stuck-in-terminating]'
    reason: ContainersNotReady
    status: "False"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:17:31Z"
    message: 'containers with unready status: [pod-stuck-in-terminating]'
    reason: ContainersNotReady
    status: "False"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2023-11-09T15:16:48Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://65ee5fd0328132157ff97107b5fbc8ca2a9f7257b0b3c4ebf322eb956f44ac11
    image: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox:latest
    imageID: projects.registry.vmware.com/tanzu_observability_keights_saas/busybox@sha256:2376a0c12759aa1214ba83e771ff252c7b1663216b192fbe5e0fb364e952f85c
    lastState: {}
    name: pod-stuck-in-terminating
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://65ee5fd0328132157ff97107b5fbc8ca2a9f7257b0b3c4ebf322eb956f44ac11
        exitCode: 137
        finishedAt: "2023-11-09T15:17:30Z"
        reason: Error
        startedAt: "2023-11-09T15:16:58Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.13
  podIPs:
  - ip: 10.244.0.13
  qosClass: Burstable
  startTime: "2023-11-09T15:16:48Z"
`

func TestIsStuckInTerminating(t *testing.T) {
	t.Run("Pod not stuck in terminating", func(t *testing.T) {
		assert.False(t, IsStuckInTerminating(fakePod()))
	})

	t.Run("Pod not stuck in terminating", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := yaml.Unmarshal([]byte(podStuckInTerminating), pod)
		require.NoError(t, err)
		assert.True(t, IsStuckInTerminating(pod))
	})

}

func fakePod() *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-pod",
			Namespace: "a-ns",
		},
		Spec: corev1.PodSpec{NodeName: "some-node-name"},
	}
	return pod
}
