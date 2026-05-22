package spec

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var dnsSafePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`)

type DependencyDefinitionView struct {
	Type       string
	Outputs    map[string]bool
	EnvPresets map[string][]string
}

type ValidateOptions struct {
	RepoFiles    map[string]bool
	Dependencies map[string]DependencyDefinitionView
}

func Validate(doc *ApplicationSpecDocument, opts ValidateOptions) ValidationResult {
	var result ValidationResult
	addError := func(field, msg string) {
		result.Errors = append(result.Errors, ValidationIssue{Field: field, Message: msg, Severity: "error"})
	}
	addWarning := func(field, msg string) {
		result.Warnings = append(result.Warnings, ValidationIssue{Field: field, Message: msg, Severity: "warning"})
	}
	if doc == nil {
		addError("", "application spec is required")
		result.Valid = false
		return result
	}
	if doc.APIVersion != APIVersionV1Alpha1 {
		addError("apiVersion", fmt.Sprintf("must be %s", APIVersionV1Alpha1))
	}
	if doc.Kind != KindApplication {
		addError("kind", "must be Application")
	}
	if doc.Metadata.Name == "" {
		addError("metadata.name", "is required")
	} else if !dnsSafePattern.MatchString(doc.Metadata.Name) {
		addError("metadata.name", "must be a DNS-safe name")
	}
	if !inSet(doc.Spec.Type, "single", "monorepo") {
		addError("spec.type", "must be single or monorepo")
	}
	if doc.Spec.Repo.Name == "" {
		addError("spec.repo.name", "is required")
	}
	if doc.Spec.Repo.Provider != "" && doc.Spec.Repo.Provider != "github" {
		addError("spec.repo.provider", "only github is supported")
	}
	if doc.Spec.Namespace != nil && !inSet(doc.Spec.Namespace.Strategy, "shared", "per-project") {
		addError("spec.namespace.strategy", "must be shared or per-project")
	}
	if doc.Spec.Namespace != nil && doc.Spec.Namespace.Strategy == "shared" && doc.Spec.Namespace.Name == "" {
		addError("spec.namespace.name", "is required when namespace strategy is shared")
	}
	if doc.Spec.AutoDeploy != nil && !inSet(doc.Spec.AutoDeploy.Mode, "all", "affected-components", "disabled") {
		addError("spec.autoDeploy.mode", "must be all, affected-components, or disabled")
	}

	dependencyNames := map[string]bool{}
	dependencyDefs := map[string]DependencyDefinitionView{}
	for i, dep := range doc.Spec.Dependencies {
		field := fmt.Sprintf("spec.dependencies[%d]", i)
		if dep.Name == "" {
			addError(field+".name", "is required")
		}
		if dependencyNames[dep.Name] {
			addError(field+".name", "must be unique")
		}
		dependencyNames[dep.Name] = true
		def, ok := opts.Dependencies[dep.Type]
		if dep.Type == "" {
			addError(field+".type", "is required")
		} else if !ok {
			addError(field+".type", "unknown dependency type")
		} else if def.Type != "" && def.Type != dep.Type {
			addError(field+".type", "does not match dependency definition")
		} else {
			dependencyDefs[dep.Name] = def
		}
	}

	componentNames := map[string]bool{}
	projectNames := map[string]bool{}
	for i, component := range doc.Spec.Components {
		field := fmt.Sprintf("spec.components[%d]", i)
		if component.Name == "" {
			addError(field+".name", "is required")
		}
		if componentNames[component.Name] {
			addError(field+".name", "must be unique")
		}
		componentNames[component.Name] = true
		if component.ProjectName == "" {
			addError(field+".projectName", "is required")
		} else if !dnsSafePattern.MatchString(component.ProjectName) {
			addError(field+".projectName", "must be DNS-safe")
		}
		if projectNames[component.ProjectName] {
			addError(field+".projectName", "must be unique")
		}
		projectNames[component.ProjectName] = true
		if !inSet(component.Kind, "service", "worker", "frontend", "job") {
			addError(field+".kind", "must be service, worker, frontend, or job")
		}
		if component.Build != nil {
			if component.Build.Context == "" {
				addError(field+".build.context", "is required")
			} else if len(opts.RepoFiles) > 0 && !repoPathExists(opts.RepoFiles, component.Build.Context) {
				addError(field+".build.context", "path was not found in repository")
			}
			if component.Build.Dockerfile == "" {
				addError(field+".build.dockerfile", "is required")
			} else if len(opts.RepoFiles) > 0 && !opts.RepoFiles[strings.Trim(component.Build.Dockerfile, "/")] {
				addError(field+".build.dockerfile", "file was not found in repository")
			}
		}
		validatePorts(component, field, addError)
		validateHealth(component, field, addError, addWarning)
		validateVolumes(component, field, addError)
		validateWatchPaths(component, field, addError, addWarning)
	}
	for i, component := range doc.Spec.Components {
		field := fmt.Sprintf("spec.components[%d]", i)
		validateDependencies(component, field, dependencyNames, componentNames, dependencyDefs, addError)
	}
	if len(doc.Spec.Components) == 0 {
		addError("spec.components", "at least one component is required")
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func validatePorts(component ComponentSpec, field string, addError func(string, string)) {
	portNames := map[string]bool{}
	for i, port := range component.Ports {
		pf := fmt.Sprintf("%s.ports[%d]", field, i)
		if port.Name == "" {
			addError(pf+".name", "is required")
		}
		if portNames[port.Name] {
			addError(pf+".name", "must be unique within component")
		}
		portNames[port.Name] = true
		if port.Port <= 0 || port.Port > 65535 {
			addError(pf+".port", "must be between 1 and 65535")
		}
		if !inSet(port.Protocol, "tcp", "udp", "http", "grpc") {
			addError(pf+".protocol", "must be tcp, udp, http, or grpc")
		}
		if !inSet(port.Exposure, "public", "private", "internal", "internal-only") {
			addError(pf+".exposure", "must be public, private, internal, or internal-only")
		}
	}
}

func validateHealth(component ComponentSpec, field string, addError func(string, string), addWarning func(string, string)) {
	if component.HealthCheck == nil {
		if component.Kind == "frontend" {
			addWarning(field+".healthCheck", "frontend component has no healthCheck; tcp probe on first port will be used")
		}
		return
	}
	h := component.HealthCheck
	if !inSet(h.Type, "http", "tcp", "disabled") {
		addError(field+".healthCheck.type", "must be http, tcp, or disabled")
	}
	if h.Type == "http" && h.Path == "" {
		addError(field+".healthCheck.path", "is required for http health checks")
	}
	if h.Type != "disabled" && h.Port != nil && !healthPortExists(component.Ports, h.Port) {
		addError(field+".healthCheck.port", "must reference an existing port name or number")
	}
}

func validateDependencies(component ComponentSpec, field string, dependencyNames, componentNames map[string]bool, definitions map[string]DependencyDefinitionView, addError func(string, string)) {
	for i, dep := range component.DependsOn {
		if !dependencyNames[dep] && !componentNames[dep] {
			addError(fmt.Sprintf("%s.dependsOn[%d]", field, i), "references an unknown component or dependency")
		}
	}
	for i, ref := range component.EnvFromDependencies {
		rf := fmt.Sprintf("%s.envFromDependencies[%d]", field, i)
		if !dependencyNames[ref.Dependency] {
			addError(rf+".dependency", "references an unknown dependency")
			continue
		}
		if ref.Preset == "" && len(ref.Mappings) == 0 {
			addError(rf, "preset or mappings is required")
		}
		def := definitions[ref.Dependency]
		if ref.Preset != "" && len(def.EnvPresets) > 0 {
			if _, ok := def.EnvPresets[ref.Preset]; !ok {
				addError(rf+".preset", "unknown dependency env preset")
			}
		}
		for key, mapping := range ref.Mappings {
			if mapping.Output == "" {
				addError(rf+".mappings."+key+".output", "is required")
			}
		}
	}
}

func validateVolumes(component ComponentSpec, field string, addError func(string, string)) {
	for i, volume := range component.Volumes {
		vf := fmt.Sprintf("%s.volumes[%d]", field, i)
		if volume.Name == "" {
			addError(vf+".name", "is required")
		}
		if !inSet(volume.Type, "pvc", "emptyDir") {
			addError(vf+".type", "must be pvc or emptyDir")
		}
		if volume.MountPath == "" {
			addError(vf+".mountPath", "is required")
		}
		if volume.Type == "pvc" && volume.Size == "" {
			addError(vf+".size", "is required for pvc volumes")
		}
	}
}

func validateWatchPaths(component ComponentSpec, field string, addError func(string, string), addWarning func(string, string)) {
	if len(component.WatchPaths) == 0 && component.Build != nil {
		addWarning(field+".watchPaths", "watchPaths is empty; build.context/** will be used as fallback")
		return
	}
	for i, pattern := range component.WatchPaths {
		if _, err := filepath.Match(pattern, "placeholder"); err != nil && !strings.Contains(pattern, "**") {
			addError(fmt.Sprintf("%s.watchPaths[%d]", field, i), "invalid glob pattern")
		}
	}
}

func repoPathExists(files map[string]bool, p string) bool {
	p = strings.Trim(strings.TrimSpace(p), "/")
	if p == "." || p == "" {
		return true
	}
	if files[p] {
		return true
	}
	prefix := p + "/"
	for file := range files {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}

func healthPortExists(ports []PortSpec, ref any) bool {
	switch v := ref.(type) {
	case string:
		for _, port := range ports {
			if port.Name == v {
				return true
			}
		}
	case int:
		for _, port := range ports {
			if port.Port == v {
				return true
			}
		}
	case int64:
		for _, port := range ports {
			if port.Port == int(v) {
				return true
			}
		}
	case float64:
		for _, port := range ports {
			if port.Port == int(v) {
				return true
			}
		}
	default:
		value := fmt.Sprint(ref)
		if n, err := strconv.Atoi(value); err == nil {
			return healthPortExists(ports, n)
		}
		return healthPortExists(ports, value)
	}
	return false
}

func inSet(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
