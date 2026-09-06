package env_test

import (
	"reflect"
	"sync"
	"testing"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

func TestPropertyResolver_whenConfiguredConcurrently_shouldRemainRaceFree(t *testing.T) {
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

	const goroutines = 16
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*2)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, _, err := resolver.GetPropertyAs("server.port", reflect.TypeOf(0)); err != nil {
					errs <- err
					return
				}
				if err := resolver.ValidateRequiredProperties(); err != nil {
					errs <- err
					return
				}
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if err := resolver.SetConversionService(convert.DefaultService()); err != nil {
					errs <- err
					return
				}
				resolver.SetRequiredProperties("server.port")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent resolver access failed: %v", err)
	}
}

func TestPropertyResolver_whenPlaceholdersHaveDefaults_shouldResolve(t *testing.T) {
	source, err := env.NewMapPropertySource("test", map[string]any{
		"app.name":  "goark",
		"app.title": "${app.name}-${missing:core}",
		"fallback":  "core",
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
	nested, err := resolver.ResolveRequiredPlaceholders("${missing:${fallback}}")
	if err != nil {
		t.Fatalf("resolve nested fallback failed: %v", err)
	}
	if nested != "core" {
		t.Fatalf("unexpected nested fallback: %q", nested)
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

func TestStandardEnvironmentProfiles_whenNegatedProfileIsEmpty_shouldNotMatch(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	if environment.AcceptsProfiles("!") {
		t.Fatal("empty negated profile should not match")
	}
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
