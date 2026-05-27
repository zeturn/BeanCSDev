package service

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed dependency-definitions/*.yaml
var embeddedDependencyDefinitions embed.FS

type DependencyDefinitionRegistry struct {
	definitions map[string]DependencyDefinition
}

type DependencyDefinition struct {
	APIVersion string                       `json:"api_version" yaml:"apiVersion"`
	Kind       string                       `json:"kind" yaml:"kind"`
	Metadata   DependencyDefinitionMetadata `json:"metadata" yaml:"metadata"`
	Spec       DependencyDefinitionSpec     `json:"spec" yaml:"spec"`
}

type DependencyDefinitionMetadata struct {
	Name        string `json:"name" yaml:"name"`
	DisplayName string `json:"display_name" yaml:"displayName"`
	Category    string `json:"category" yaml:"category"`
}

type DependencyDefinitionSpec struct {
	Type                   string                         `json:"type" yaml:"type"`
	SupportedDeployMethods []string                       `json:"supported_deploy_methods" yaml:"supportedDeployMethods"`
	DefaultDeployMethod    string                         `json:"default_deploy_method" yaml:"defaultDeployMethod"`
	Helm                   DependencyDefinitionHelm       `json:"helm,omitempty" yaml:"helm"`
	ConfigSchema           map[string]any                 `json:"config_schema" yaml:"configSchema"`
	Services               []DependencyDefinitionService  `json:"services" yaml:"services"`
	Outputs                map[string]DependencyOutput    `json:"outputs" yaml:"outputs"`
	EnvPresets             map[string]DependencyEnvPreset `json:"env_presets" yaml:"envPresets"`
	Persistence            map[string]any                 `json:"persistence,omitempty" yaml:"persistence"`
}

type DependencyDefinitionHelm struct {
	Chart          DependencyHelmChart `json:"chart" yaml:"chart"`
	ValuesTemplate string              `json:"values_template" yaml:"valuesTemplate"`
}

type DependencyHelmChart struct {
	Repo    string `json:"repo" yaml:"repo"`
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type DependencyDefinitionService struct {
	Name     string `json:"name" yaml:"name"`
	Port     int    `json:"port" yaml:"port"`
	Protocol string `json:"protocol" yaml:"protocol"`
	Default  bool   `json:"default,omitempty" yaml:"default"`
	Optional bool   `json:"optional,omitempty" yaml:"optional"`
}

type DependencyOutput struct {
	Value          string            `json:"value,omitempty" yaml:"value"`
	ValueFrom      map[string]string `json:"value_from,omitempty" yaml:"valueFrom"`
	Template       string            `json:"template,omitempty" yaml:"template"`
	SecretTemplate string            `json:"secret_template,omitempty" yaml:"secretTemplate"`
}

type DependencyEnvPreset struct {
	Env map[string]DependencyEnvValue `json:"env" yaml:"env"`
}

type DependencyEnvValue struct {
	Output         string `json:"output,omitempty" yaml:"output"`
	Secret         bool   `json:"secret,omitempty" yaml:"secret"`
	Value          string `json:"value,omitempty" yaml:"value"`
	SecretOutput   string `json:"secret_output,omitempty" yaml:"secretOutput"`
	SecretKeyRef   string `json:"secret_key_ref,omitempty" yaml:"secretKeyRef"`
	SecretTemplate string `json:"secret_template,omitempty" yaml:"secretTemplate"`
}

func NewDependencyDefinitionRegistry() (*DependencyDefinitionRegistry, error) {
	entries, err := embeddedDependencyDefinitions.ReadDir("dependency-definitions")
	if err != nil {
		return nil, err
	}
	reg := &DependencyDefinitionRegistry{definitions: map[string]DependencyDefinition{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		body, err := embeddedDependencyDefinitions.ReadFile("dependency-definitions/" + entry.Name())
		if err != nil {
			return nil, err
		}
		var def DependencyDefinition
		if err := yaml.Unmarshal(body, &def); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if err := validateDependencyDefinition(def); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		reg.definitions[def.Metadata.Name] = def
	}
	return reg, nil
}

func (r *DependencyDefinitionRegistry) List() []DependencyDefinition {
	out := make([]DependencyDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Metadata.Name < out[j].Metadata.Name
	})
	return out
}

func (r *DependencyDefinitionRegistry) Get(name string) (DependencyDefinition, bool) {
	name = strings.TrimSpace(name)
	if name == "pgsql" || name == "postgres" {
		name = "postgresql"
	}
	def, ok := r.definitions[name]
	return def, ok
}

func validateDependencyDefinition(def DependencyDefinition) error {
	if def.APIVersion == "" || def.Kind != "DependencyDefinition" {
		return fmt.Errorf("invalid apiVersion or kind")
	}
	if def.Metadata.Name == "" || def.Spec.Type == "" {
		return fmt.Errorf("metadata.name and spec.type are required")
	}
	if def.Spec.DefaultDeployMethod == "" {
		return fmt.Errorf("spec.defaultDeployMethod is required")
	}
	if len(def.Spec.Outputs) == 0 {
		return fmt.Errorf("spec.outputs is required")
	}
	if len(def.Spec.EnvPresets) == 0 {
		return fmt.Errorf("spec.envPresets is required")
	}
	return nil
}
