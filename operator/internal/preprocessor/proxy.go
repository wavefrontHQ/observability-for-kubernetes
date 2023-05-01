package preprocessor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	baseYaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PreprocessorRule struct {
	Rule   string
	Action string
	Key    string `yaml:",omitempty"`
	Tag    string `yaml:",omitempty"`
	Value  string `yaml:",omitempty"`
}

func SetEnabledPorts(wavefront *wf.Wavefront) {
	allPorts := []int{wavefront.Spec.DataExport.WavefrontProxy.MetricPort,
		wavefront.Spec.DataExport.WavefrontProxy.DeltaCounterPort,
		wavefront.Spec.DataExport.WavefrontProxy.OTLP.GrpcPort,
		wavefront.Spec.DataExport.WavefrontProxy.OTLP.HttpPort,
		wavefront.Spec.DataExport.WavefrontProxy.Tracing.Wavefront.Port,
		wavefront.Spec.DataExport.WavefrontProxy.Tracing.Jaeger.Port,
		wavefront.Spec.DataExport.WavefrontProxy.Tracing.Jaeger.GrpcPort,
		wavefront.Spec.DataExport.WavefrontProxy.Tracing.Jaeger.HttpPort,
		wavefront.Spec.DataExport.WavefrontProxy.Tracing.Zipkin.Port,
		wavefront.Spec.DataExport.WavefrontProxy.Histogram.Port,
		wavefront.Spec.DataExport.WavefrontProxy.Histogram.MinutePort,
		wavefront.Spec.DataExport.WavefrontProxy.Histogram.HourPort,
		wavefront.Spec.DataExport.WavefrontProxy.Histogram.DayPort,
	}

	var enabledPorts []int
	for _, value := range allPorts {
		if value != 0 {
			enabledPorts = append(enabledPorts, value)
		}
	}

	wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.EnabledPorts = strings.Trim(strings.Join(strings.Fields(fmt.Sprint(enabledPorts)), ","), "[]")
}

func SetUserDefinedRules(client client.Client, wavefront *wf.Wavefront) error {
	if len(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor) == 0 {
		return nil
	}

	preprocessorConfigMap, err := findConfigMap(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor, wavefront.Spec.Namespace, client)
	if err != nil {
		return err
	}

	rules := make(map[string][]PreprocessorRule)
	if err := baseYaml.Unmarshal([]byte(preprocessorConfigMap.Data["rules.yaml"]), &rules); err != nil {
		return err
	}

	var globalRulesYAML []byte
	if len(rules["global"]) > 0 {
		globalRulesYAML, err = baseYaml.Marshal(rules["global"])
		if err != nil {
			return err
		}
		delete(rules, "global")
	}

	userDefinedRulesYAML, err := baseYaml.Marshal(rules)

	wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.GlobalUserDefinedRules = string(globalRulesYAML)
	wavefront.Spec.DataExport.WavefrontProxy.PreprocessorRules.UserDefinedRules = string(userDefinedRulesYAML)
	return nil
}

func ValidateRules(namespace string, client client.Client, wavefront *wf.Wavefront) error {
	if len(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor) == 0 {
		return nil
	}
	preprocessorConfigMap, err := findConfigMap(wavefront.Spec.DataExport.WavefrontProxy.Preprocessor, namespace, client)
	if err != nil {
		return err
	}

	out := make(map[string][]PreprocessorRule)
	if err := baseYaml.Unmarshal([]byte(preprocessorConfigMap.Data["rules.yaml"]), &out); err != nil {
		return errors.New("cannot parse rules YAML configured in dataExport.wavefrontProxy.preprocessor")
	}

	for port, rules := range out {
		fmt.Printf("port:%s rules:%+v\n", port, rules)
		for _, rule := range rules {
			fmt.Printf("value:%+v\n", rule)
			errMsg := "invalid rule configured in dataExport.wavefrontProxy.preprocessor, overriding %s tag '%s' is disallowed"
			if rule.Tag == "cluster" {
				return fmt.Errorf(errMsg, "metric", "cluster")
			}
			if rule.Tag == "cluster_uuid" {
				return fmt.Errorf(errMsg, "metric", "cluster_uuid")
			}
			if rule.Key == "cluster" {
				return fmt.Errorf(errMsg, "span", "cluster")
			}
			if rule.Key == "cluster_uuid" {
				return fmt.Errorf(errMsg, "span", "cluster_uuid")
			}
		}
	}

	println(fmt.Sprintf("%+v", out))
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
