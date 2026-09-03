package env_test

import (
	"reflect"
	"testing"

	"goark.dev/goark/core/env"
)

func TestResolveValue_whenPlaceholderExists_shouldResolveAndConvert(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	source, err := env.NewMapPropertySource("test", map[string]any{
		"server.port": "8080",
		"app.name":    "goark",
	})
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("add source failed: %v", err)
	}

	port, err := env.ResolveValue(environment, "${server.port}", reflect.TypeOf(0))
	if err != nil {
		t.Fatalf("resolve port failed: %v", err)
	}
	if port.(int) != 8080 {
		t.Fatalf("expected port 8080, got %#v", port)
	}

	title, err := env.ResolveValueAs[string](environment, "service-${app.name}")
	if err != nil {
		t.Fatalf("resolve title failed: %v", err)
	}
	if title != "service-goark" {
		t.Fatalf("unexpected title: %q", title)
	}
}

func TestResolveValue_whenPlaceholderMissingHasDefault_shouldUseDefault(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	enabled, err := env.ResolveValueAs[bool](environment, "${feature.enabled:true}")
	if err != nil {
		t.Fatalf("resolve default failed: %v", err)
	}
	if !enabled {
		t.Fatal("expected default true")
	}
}

func TestResolveValue_whenLiteral_shouldConvertLiteral(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	value, err := env.ResolveValueAs[int](environment, "42")
	if err != nil {
		t.Fatalf("resolve literal failed: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

func TestResolveValue_whenStringLiteralHasWhitespace_shouldPreserveWhitespace(t *testing.T) {
	environment := env.MustNewStandardEnvironment()
	value, err := env.ResolveValueAs[string](environment, "  value  ")
	if err != nil {
		t.Fatalf("resolve literal failed: %v", err)
	}
	if value != "  value  " {
		t.Fatalf("expected preserved whitespace, got %q", value)
	}
}

func TestResolveValue_whenGaELExpression_shouldEvaluateAndConvert(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	source, err := env.NewMapPropertySource("test", map[string]any{"feature.enabled": "true"})
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("add source failed: %v", err)
	}
	value, err := env.ResolveValueAs[bool](environment, "#{environment['feature.enabled'] == 'true'}")
	if err != nil {
		t.Fatalf("resolve GaEL failed: %v", err)
	}
	if !value {
		t.Fatal("expected GaEL result true")
	}
}

func TestResolveValue_whenGaELIsMixedWithLiteral_shouldReturnError(t *testing.T) {
	environment := env.MustNewStandardEnvironment()
	if _, err := env.ResolveValueAs[string](environment, "prefix-#{true}"); err == nil {
		t.Fatal("expected complete GaEL expression error")
	}
}
