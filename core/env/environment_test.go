package env_test

import (
	"reflect"
	"sync"
	"testing"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
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

func TestMapPropertySource_whenMutableValuesAreExposed_shouldReturnDefensiveCopies(t *testing.T) {
	pointerTags := []any{"pointer-core", "pointer-env"}
	sourceValues := map[string]any{
		"tags":        []any{"core", "env"},
		"nested":      map[string]any{"name": "goark"},
		"pointerTags": &pointerTags,
	}
	source, err := env.NewMapPropertySource("test", sourceValues)
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	sourceValues["tags"].([]any)[0] = "mutated-input"
	sourceValues["nested"].(map[string]any)["name"] = "mutated-input"
	pointerTags[0] = "mutated-input"

	tags, ok := source.GetProperty("tags")
	if !ok {
		t.Fatal("tags property not found")
	}
	if tags.([]any)[0] != "core" {
		t.Fatalf("input mutation should not affect source, got %#v", tags)
	}
	pointerValue, ok := source.GetProperty("pointerTags")
	if !ok {
		t.Fatal("pointerTags property not found")
	}
	if (*(pointerValue.(*[]any)))[0] != "pointer-core" {
		t.Fatalf("input pointer mutation should not affect source, got %#v", pointerValue)
	}

	sourceSnapshot := source.Source().(map[string]any)
	sourceSnapshot["tags"].([]any)[0] = "mutated-source"
	sourceSnapshot["nested"].(map[string]any)["name"] = "mutated-source"
	(*sourceSnapshot["pointerTags"].(*[]any))[0] = "mutated-source"
	tags, _ = source.GetProperty("tags")
	nested, _ := source.GetProperty("nested")
	pointerValue, _ = source.GetProperty("pointerTags")
	if tags.([]any)[0] != "core" || nested.(map[string]any)["name"] != "goark" || (*(pointerValue.(*[]any)))[0] != "pointer-core" {
		t.Fatalf("source snapshot mutation should not affect source, got tags=%#v nested=%#v pointer=%#v", tags, nested, pointerValue)
	}

	tags.([]any)[0] = "mutated-get"
	nested.(map[string]any)["name"] = "mutated-get"
	(*(pointerValue.(*[]any)))[0] = "mutated-get"
	tags, _ = source.GetProperty("tags")
	nested, _ = source.GetProperty("nested")
	pointerValue, _ = source.GetProperty("pointerTags")
	if tags.([]any)[0] != "core" || nested.(map[string]any)["name"] != "goark" || (*(pointerValue.(*[]any)))[0] != "pointer-core" {
		t.Fatalf("property value mutation should not affect source, got tags=%#v nested=%#v pointer=%#v", tags, nested, pointerValue)
	}
}

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

func TestStandardEnvironmentProfiles_whenNegatedProfileIsEmpty_shouldNotMatch(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	if environment.AcceptsProfiles("!") {
		t.Fatal("empty negated profile should not match")
	}
}

func TestMatchProfileExpression_whenExpressionUsesBooleanOperators_shouldEvaluate(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	if err := environment.SetActiveProfiles("prod", "mysql"); err != nil {
		t.Fatalf("set active profiles failed: %v", err)
	}

	cases := []struct {
		expression string
		want       bool
	}{
		{expression: "prod", want: true},
		{expression: "dev | prod", want: true},
		{expression: "prod & mysql", want: true},
		{expression: "prod & !mysql", want: false},
		{expression: "prod & (mysql | postgres)", want: true},
		{expression: "!dev", want: true},
	}
	for _, item := range cases {
		got, err := env.MatchProfileExpression(environment, item.expression)
		if err != nil {
			t.Fatalf("match %q failed: %v", item.expression, err)
		}
		if got != item.want {
			t.Fatalf("expression %q expected %v, got %v", item.expression, item.want, got)
		}
	}
}

func TestMatchProfileExpression_whenExpressionInvalid_shouldReturnError(t *testing.T) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	_, err = env.MatchProfileExpression(environment, "dev | | test")
	if err == nil {
		t.Fatal("expected invalid expression error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
