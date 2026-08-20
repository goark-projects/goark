package env_test

import (
	"reflect"
	"testing"

	"github.com/goark-projects/goark/core/env"
	arkerrors "github.com/goark-projects/goark/errors"
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

func TestResolveValue_whenSpELExpression_shouldReturnError(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	_, err = env.ResolveValueAs[string](environment, "#{systemProperties['user.home']}")
	if err == nil {
		t.Fatal("expected SpEL error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
