// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// FromFile loads the configuration from a given file
func FromFile(filename string) (*Config, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to load configuration file: %v", err)
	}
	return FromYAML(contents)
}

// FromYAML loads the configuration from a blob of YAML.
func FromYAML(contents []byte) (*Config, error) {
	return New(func(cfg *Config) error {
		if err := yaml.UnmarshalStrict(contents, cfg); err != nil {
			return fmt.Errorf("unable to parse configuration: %v", err)
		}
		return nil
	})
}
