package proxyPreprocessor

import (
	"context"
	"fmt"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	baseYaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PreprocessorRule struct {
	Rule   string
	Action string
	Key    string
	Tag    string
	Value  string
}

func ValidateRules(namespace string, client client.Client, wavefront *wf.Wavefront) error {
	if len(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor) != 0 {
		preprocessorConfigMap, err := findConfigMap(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor, namespace, client)
		if err != nil {
			return err
		}
		//println(fmt.Sprintf("Configmap:%v", preprocessorConfigMap.Data))
		//
		//println(fmt.Sprintf("String output:%s", preprocessorConfigMap.Data["rules.yaml"]))

		out := make(map[string][]PreprocessorRule)
		if err := baseYaml.Unmarshal([]byte(preprocessorConfigMap.Data["rules.yaml"]), &out); err != nil {
			return err
		}

		for port, rules := range out {
			fmt.Printf("port:%s rules:%+v\n", port, rules)
			for _, rule := range rules {
				fmt.Printf("value:%+v\n", rule)
				//for key, value := range rule {
				//	fmt.Printf("key:%s value:%+v\n", key, value)
				//}
				if rule.Tag == "cluster" {
					return fmt.Errorf("Invalid rule configured in dataExport.wavefrontProxy.preprocessor, overriding metric tag 'cluster' is disallowed.")
				}
				if rule.Tag == "cluster_uuid" {
					return fmt.Errorf("Invalid rule configured in dataExport.wavefrontProxy.preprocessor, overriding metric tag 'cluster_uuid' is disallowed.")
				}
				if rule.Key == "cluster" {
					return fmt.Errorf("Invalid rule configured in dataExport.wavefrontProxy.preprocessor, overriding span key 'cluster' is disallowed.")
				}
				if rule.Key == "cluster_uuid" {
					return fmt.Errorf("Invalid rule configured in dataExport.wavefrontProxy.preprocessor, overriding span key 'cluster_uuid' is disallowed.")
				}
			}
		}

		println(fmt.Sprintf("%+v", out))
	}
	return nil
}

func findConfigMap(name, namespace string, client client.Client) (*corev1.ConfigMap, error) {
	objectKey := util.ObjKey(namespace, name)
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.Background(), objectKey, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

// retrieve config map and look for port.
//func (r *WavefrontReconciler) postProcess(wavefront *wf.Wavefront, ctx context.Context) error {
//	if len(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor) != 0 {
//		preprocessorConfigMap, err := r.findConfigMap(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor, ctx)
//		if err != nil {
//			return err
//		}
//		out := make(map[string][]PreprocessorRule)
//		if err := baseYaml.Unmarshal([]byte(preprocessorConfigMap.Data["rules.yaml"]), &out); err != nil {
//			return err
//		}
//
//		for port, rules := range out {
//			println(fmt.Printf("port:%s rules:%+v\n", port, rules))
//			if port.
//		}
//
//		println(fmt.Sprintf("%+v", out))
//	}
//	return nil
//}
