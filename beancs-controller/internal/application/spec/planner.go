package spec

import "sort"

func Plan(doc *ApplicationSpecDocument, validation ValidationResult, opts ValidateOptions) ApplicationPlan {
	if doc == nil {
		return ApplicationPlan{}
	}
	plan := ApplicationPlan{
		Application: PlannedApplication{Name: doc.Metadata.Name, Type: doc.Spec.Type},
		Warnings:    validation.Warnings,
	}
	dependencyDefs := map[string]DependencyDefinitionView{}
	for _, dep := range doc.Spec.Dependencies {
		if def, ok := opts.Dependencies[dep.Type]; ok {
			dependencyDefs[dep.Name] = def
		}
		deployMethod := dep.DeployMethod
		if deployMethod == "" {
			if def, ok := opts.Dependencies[dep.Type]; ok && def.Type != "" {
				deployMethod = "helm"
			}
		}
		if deployMethod == "" {
			deployMethod = "helm"
		}
		plan.WillCreateDependencies = append(plan.WillCreateDependencies, PlannedDependency{
			Name:         dep.Name,
			Type:         dep.Type,
			DeployMethod: deployMethod,
			Version:      dep.Version,
			Config:       dep.Config,
		})
	}
	for _, component := range doc.Spec.Components {
		project := PlannedProject{
			Name:        component.ProjectName,
			Component:   component.Name,
			Kind:        component.Kind,
			Ports:       component.Ports,
			HealthCheck: component.HealthCheck,
			Volumes:     component.Volumes,
			WatchPaths:  effectiveWatchPaths(component),
		}
		if component.Build != nil {
			project.Dockerfile = component.Build.Dockerfile
			project.BuildContext = component.Build.Context
			project.BuildArgs = component.Build.Args
		}
		plan.WillCreateProjects = append(plan.WillCreateProjects, project)
		for _, envRef := range component.EnvFromDependencies {
			names := envNamesForDependency(envRef, dependencyDefs)
			plan.WillInjectEnv = append(plan.WillInjectEnv, PlannedEnvInjection{
				Component:  component.Name,
				Dependency: envRef.Dependency,
				Env:        names,
			})
		}
	}
	return plan
}

func effectiveWatchPaths(component ComponentSpec) []string {
	if len(component.WatchPaths) > 0 {
		return component.WatchPaths
	}
	if component.Build != nil && component.Build.Context != "" && component.Build.Context != "." {
		return []string{component.Build.Context + "/**"}
	}
	return nil
}

func envNamesForDependency(ref EnvFromDependencySpec, deps map[string]DependencyDefinitionView) []string {
	seen := map[string]bool{}
	var out []string
	if ref.Preset != "" {
		def := deps[ref.Dependency]
		for _, name := range def.EnvPresets[ref.Preset] {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	for name := range ref.Mappings {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
