package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error in getting config: %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error in getting access to K8S: %s", err.Error())
	}
	return clientset, nil
}

func GetDynamicClient() (dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error in getting config: %s", err.Error())
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error in getting access to K8S: %s", err.Error())
	}
	return client, nil
}

func GetMaxMissedRowBatchSize(pemPods []corev1.Pod, sinceSeconds int64, client kubernetes.Interface) MiB {
	var maxMissedRowBatchSize Bytes
	for _, pemPod := range pemPods {
		rowBatchSizeErrors, err := ExtractFromPodLogs(client, pemPod, sinceSeconds, ExtractRowBatchSizeError)
		if err != nil {
			log.Fatalf("error extracting row batch size errors: %s", err.Error())
		}
		if len(rowBatchSizeErrors) <= 1 { // you get one missed row batch at startup, but nothing else
			continue
		}
		for _, rowBatchSizeError := range rowBatchSizeErrors[1:] {
			maxMissedRowBatchSize = MaxNumber(rowBatchSizeError.RowBatchSize, maxMissedRowBatchSize)
		}
	}
	return MiB(math.Ceil(float64(maxMissedRowBatchSize) / (1024.0 * 1024.0)))
}

func ExtractFromPodLogs[T any](client kubernetes.Interface, pod corev1.Pod, sinceSeconds int64, extract func(string) (T, bool)) ([]T, error) {
	podLogOpts := corev1.PodLogOptions{SinceSeconds: &sinceSeconds}
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error in opening stream: %s", err.Error())
	}
	defer podLogs.Close()
	var matched []T
	lines := bufio.NewScanner(podLogs)
	for lines.Scan() {
		line := lines.Text()
		data, matches := extract(line)
		if matches {
			matched = append(matched, data)
		}
	}
	if lines.Err() != nil {
		return nil, fmt.Errorf("error reading lines: %s", err.Error())
	}
	return matched, nil
}

var rowBatchSizeRegex = regexp.MustCompile(`RowBatch size \((?P<RowBatchSize>\d+)\).+\((?P<MaxTableSize>\d+)\).$`)

func ExtractRowBatchSizeError(line string) (RowBatchSizeError, bool) {
	match := rowBatchSizeRegex.FindStringSubmatch(line)
	if len(match) == 0 {
		return RowBatchSizeError{}, false
	}
	rowBatchSize, err := ParseInt(match[1])
	if err != nil {
		panic(err)
	}
	maxTableSize, err := ParseInt(match[2])
	if err != nil {
		panic(err)
	}
	return RowBatchSizeError{RowBatchSize: rowBatchSize, MaxTableSize: maxTableSize}, true
}

func GetPodsByLabel(client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.Pod, error) {
	podList, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pods by label (%s) in ns %s: %s", labelSelector, namespace, err.Error())
	}
	return podList.Items, nil
}

func GetConfigMapsByLabel(client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.ConfigMap, error) {
	cmList, err := client.CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pods by label (%s) in ns %s: %s", labelSelector, namespace, err.Error())
	}
	return cmList.Items, nil
}

var cronYAMLFrequencyRegex = regexp.MustCompile(`(?m)frequency_s:\s*(\d+)$`)

func GetMaxFrequencyFromCronScripts(configMaps []corev1.ConfigMap) time.Duration {
	maxFrequencyS := 0
	for _, configMap := range configMaps {
		cronYAML := configMap.Data["cron.yaml"]
		match := cronYAMLFrequencyRegex.FindStringSubmatch(cronYAML)
		if len(match) == 0 {
			continue
		}
		cronFrequencyS, err := ParseInt(match[1])
		if err != nil {
			panic(err)
		}
		maxFrequencyS = MaxNumber(maxFrequencyS, cronFrequencyS)
	}
	return time.Duration(maxFrequencyS) * time.Second
}

func GetPrefixForPEMSettings(client dynamic.Interface, namespace string) (string, error) {
	obj, err := client.Resource(schema.GroupVersionResource{
		Group:    "wavefront.com",
		Version:  "v1alpha1",
		Resource: "wavefronts",
	}).Namespace(namespace).Get(context.Background(), "wavefront", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error fetching wavefront CR: %s", err)
	}
	hubPixieEnabled, _, err := unstructured.NestedBool(obj.Object, "spec", "experimental", "hub", "pixie", "enable")
	if err != nil {
		return "", fmt.Errorf("error retrieving spec.experimental.hub.pixie.enable: %s", err)
	}
	if hubPixieEnabled {
		return "spec.experimental.hub.pixie.pem", nil
	}
	return "spec.experimental.autotracing.pem", nil
}

type RowBatchSizeError struct {
	RowBatchSize Bytes
	MaxTableSize Bytes
}
