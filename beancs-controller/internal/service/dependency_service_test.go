package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestRabbitMQDependencyOutputsIncludeUsername(t *testing.T) {
	registry, err := NewDependencyDefinitionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Get("rabbitmq")
	if !ok {
		t.Fatal("rabbitmq definition not found")
	}
	config := applyDependencyConfigDefaults(def, model.JSONMap{"username": "araneae"})
	secretData := dependencySecretData(def, config)
	outputs := flattenDependencyOutputs(resolveDependencyOutputs(def, "rabbitmq", config, secretData))

	if outputs["username"] != "araneae" {
		t.Fatalf("expected username output to be araneae, got %q", outputs["username"])
	}
	if outputs["password"] == "" {
		t.Fatal("expected generated password output")
	}
	if outputs["url"] == "" || outputs["url"] == "amqp://:@rabbitmq:5672/" {
		t.Fatalf("expected populated rabbitmq url, got %q", outputs["url"])
	}
}
