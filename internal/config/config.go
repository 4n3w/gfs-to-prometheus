package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MetricPrefix   string                       `yaml:"metric_prefix"`
	MetricMappings map[string]MetricMapping     `yaml:"metric_mappings"`
	LabelMappings  map[string]string            `yaml:"label_mappings"`
	Filters        Filters                      `yaml:"filters"`
}

type MetricMapping struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
	Drop   bool              `yaml:"drop"`
}

type Filters struct {
	IncludeResourceTypes []string `yaml:"include_resource_types"`
	ExcludeResourceTypes []string `yaml:"exclude_resource_types"`
	IncludeStats         []string `yaml:"include_stats"`
	ExcludeStats         []string `yaml:"exclude_stats"`
}

func Default() *Config {
	return &Config{
		MetricPrefix:   "gemfire",
		MetricMappings: make(map[string]MetricMapping),
		LabelMappings:  make(map[string]string),
		Filters:        Filters{},
	}
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}