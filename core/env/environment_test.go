package env_test

import (
	"reflect"
	"testing"

	"github.com/goark-projects/goark/core/env"
	arkerrors "github.com/goark-projects/goark/errors"
)

func TestStandardEnvironment_whenSourcesHaveSameKey_shouldUseHigherPriority(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	low, err := env.NewMapPropertySource("low", map[string]any{
		"app.name": "low",
		"app.port": "8080",
	})
	if err != nil {
		t.Fatalf("create low source failed: %v", err)
	}
	high, err := env.NewMapPropertySource("high", map[string]any{
		"app.name": "high",
	})
	if err != nil {
		t.Fatalf("create high source failed: %v", err)
	}
	if err := environment.PropertySources().AddLast(low); err != nil {
		t.Fatalf("add low source failed: %v", err)
	}
	if err := environment.PropertySources().AddFirst(high); err != nil {
		t.Fatalf("add high source failed: %v", err)
	}

	name, ok := environment.GetProperty("app.name")
	if !ok || name != "high" {
		t.Fatalf("expected high priority value, got %q, %v", name, ok)
	}
	port, ok := environment.GetProperty("app.port")
	if !ok || port != "8080" {
		t.Fatalf("expected fallback value, got %q, %v", port, ok)
	}
	if got := environment.GetPropertyOrDefault("missing", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestPropertyResolver_whenPlaceholdersHaveDefaults_shouldResolve(t *testing.T) {
	source, err := env.NewMapPropertySource("test", map[string]any{
		"app.name":  "goark",
		"app.title": "${app.name}-${missing:core}",
	})
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	sources, err := env.NewMutablePropertySources(source)
	if err != nil {
		t.Fatalf("create property sources failed: %v", err)
	}
	resolver, err := env.NewPropertySourcesPropertyResolver(sources)
	if err != nil {
		t.Fatalf("create resolver failed: %v", err)
	}

	title, ok := resolver.GetProperty("app.title")
	if !ok || title != "goark-core" {
		t.Fatalf("unexpected title: %q, %v", title, ok)
	}
	unresolved, err := resolver.ResolvePlaceholders("x-${unknown}")
	if err != nil {
		t.Fatalf("resolve optional placeholder failed: %v", err)
	}
	if unresolved != "x-${unknown}" {
		t.Fatalf("unexpected unresolved placeholder: %q", unresolved)
	}
	_, err = resolver.ResolveRequiredPlaceholders("x-${unknown}")
	if err == nil || !arkerrors.Is(err, arkerrors.CodeNotFound) {
		t.Fatalf("expected required placeholder not found, got %v", err)
	}
}

func TestPropertyResolver_whenTargetTypeIsRequested_shouldConvertValue(t *testing.T) {
	source, err := env.NewMapPropertySource("test", map[string]any{
		"server.port": "8080",
	})
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	sources, err := env.NewMutablePropertySources(source)
	if err != nil {
		t.Fatalf("create property sources failed: %v", err)
	}
	resolver, err := env.NewPropertySourcesPropertyResolver(sources)
	if err != nil {
		t.Fatalf("create resolver failed: %v", err)
	}

	value, ok, err := resolver.GetPropertyAs("server.port", reflect.TypeOf(0))
	if err != nil {
		t.Fatalf("convert property failed: %v", err)
	}
	if !ok || value.(int) != 8080 {
		t.Fatalf("unexpected converted value: %#v, %v", value, ok)
	}
}

func TestStandardEnvironmentProfiles_whenActiveProfilesAreEmpty_shouldUseDefaultProfiles(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	if !environment.AcceptsProfiles("default") {
		t.Fatal("expected default profile to be accepted")
	}
	if err := environment.SetActiveProfiles("prod", "api"); err != nil {
		t.Fatalf("set active profiles failed: %v", err)
	}
	if !environment.AcceptsProfiles("prod") || environment.AcceptsProfiles("default") {
		t.Fatalf("unexpected active profile state: active=%#v default=%#v", environment.ActiveProfiles(), environment.DefaultProfiles())
	}
}
