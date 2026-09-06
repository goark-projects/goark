package env_test

import (
	"sort"
	"testing"

	"goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

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
	if tags.([]any)[0] != "core" || nested.(map[string]any)["name"] != "goark" ||
		(*(pointerValue.(*[]any)))[0] != "pointer-core" {
		t.Fatalf(
			"source snapshot mutation should not affect source, got tags=%#v nested=%#v pointer=%#v",
			tags,
			nested,
			pointerValue,
		)
	}

	tags.([]any)[0] = "mutated-get"
	nested.(map[string]any)["name"] = "mutated-get"
	(*(pointerValue.(*[]any)))[0] = "mutated-get"
	tags, _ = source.GetProperty("tags")
	nested, _ = source.GetProperty("nested")
	pointerValue, _ = source.GetProperty("pointerTags")
	if tags.([]any)[0] != "core" || nested.(map[string]any)["name"] != "goark" ||
		(*(pointerValue.(*[]any)))[0] != "pointer-core" {
		t.Fatalf(
			"property value mutation should not affect source, got tags=%#v nested=%#v pointer=%#v",
			tags,
			nested,
			pointerValue,
		)
	}
}

func TestStandardEnvironmentPropertyNames_whenSourcesOverlap_shouldReturnSortedUniqueNames(
	t *testing.T,
) {
	environment, err := env.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("NewStandardEnvironment() error = %v", err)
	}
	first, err := env.NewMapPropertySource("first", map[string]any{
		"logging.level.root": "info",
		"server.port":        8080,
	})
	if err != nil {
		t.Fatalf("NewMapPropertySource(first) error = %v", err)
	}
	second, err := env.NewMapPropertySource("second", map[string]any{
		"logging.level.root":          "debug",
		"logging.level.goark.dev.web": "warn",
	})
	if err != nil {
		t.Fatalf("NewMapPropertySource(second) error = %v", err)
	}
	if err := environment.PropertySources().AddFirst(first); err != nil {
		t.Fatalf("AddFirst(first) error = %v", err)
	}
	if err := environment.PropertySources().AddLast(second); err != nil {
		t.Fatalf("AddLast(second) error = %v", err)
	}

	got := environment.PropertyNames()
	if !sort.StringsAreSorted(got) {
		t.Fatalf("PropertyNames() is not sorted: %#v", got)
	}
	for _, want := range []string{"logging.level.goark.dev.web", "logging.level.root", "server.port"} {
		if count := countString(got, want); count != 1 {
			t.Fatalf("PropertyNames() contains %q %d times, want once", want, count)
		}
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

func TestStandardEnvironmentProfiles_whenActiveProfilesAreEmpty_shouldUseDefaultProfiles(
	t *testing.T,
) {
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
		t.Fatalf(
			"unexpected active profile state: active=%#v default=%#v",
			environment.ActiveProfiles(),
			environment.DefaultProfiles(),
		)
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
