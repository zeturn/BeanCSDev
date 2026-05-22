package spec

import "testing"

func TestValidateAndPlanAraneaeSpec(t *testing.T) {
	doc, err := Parse([]byte(araneaeSpecYAML))
	if err != nil {
		t.Fatal(err)
	}
	opts := ValidateOptions{
		RepoFiles: map[string]bool{
			"Backend/Dockerfile":  true,
			"Frontend/Dockerfile": true,
		},
		Dependencies: map[string]DependencyDefinitionView{
			"rabbitmq": {
				Type:    "rabbitmq",
				Outputs: map[string]bool{"url": true},
				EnvPresets: map[string][]string{
					"rabbitmq_default": {"RABBITMQ_URL"},
				},
			},
		},
	}
	result := Validate(doc, opts)
	if !result.Valid {
		t.Fatalf("expected valid spec, got errors: %#v", result.Errors)
	}
	plan := Plan(doc, result, opts)
	if plan.Application.Name != "araneae" {
		t.Fatalf("application = %q", plan.Application.Name)
	}
	if len(plan.WillCreateDependencies) != 1 || plan.WillCreateDependencies[0].Name != "rabbitmq" {
		t.Fatalf("dependencies = %#v", plan.WillCreateDependencies)
	}
	if len(plan.WillCreateProjects) != 3 {
		t.Fatalf("projects = %#v", plan.WillCreateProjects)
	}
	executor := plan.WillCreateProjects[1]
	if executor.BuildArgs["TARGET"] != "executor" {
		t.Fatalf("executor TARGET = %q", executor.BuildArgs["TARGET"])
	}
	if executor.HealthCheck == nil || executor.HealthCheck.Path != "/healthz" {
		t.Fatalf("executor health = %#v", executor.HealthCheck)
	}
	if len(plan.WillInjectEnv) != 2 {
		t.Fatalf("env injections = %#v", plan.WillInjectEnv)
	}
}

func TestValidateDuplicateComponentNameFails(t *testing.T) {
	doc, err := Parse([]byte(araneaeSpecYAML))
	if err != nil {
		t.Fatal(err)
	}
	doc.Spec.Components[1].Name = "control"
	result := Validate(doc, ValidateOptions{Dependencies: map[string]DependencyDefinitionView{"rabbitmq": {Type: "rabbitmq"}}})
	if result.Valid {
		t.Fatal("expected duplicate component name to fail")
	}
}

func TestValidateUnknownDependsOnFails(t *testing.T) {
	doc, err := Parse([]byte(araneaeSpecYAML))
	if err != nil {
		t.Fatal(err)
	}
	doc.Spec.Components[0].DependsOn = []string{"missing"}
	result := Validate(doc, ValidateOptions{Dependencies: map[string]DependencyDefinitionView{"rabbitmq": {Type: "rabbitmq"}}})
	if result.Valid {
		t.Fatal("expected unknown dependsOn to fail")
	}
}
