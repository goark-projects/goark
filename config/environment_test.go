package config_test

import (
	"reflect"
	"testing"

	"github.com/goark-projects/goark/config"
)

func TestEnvironment_whenSourcesHaveSameKey_shouldUseHigherPriority(t *testing.T) {
	low, err := config.NewMapSource("low", map[string]string{
		"app.name": "low",
		"app.port": "8080",
	})
	if err != nil {
		t.Fatalf("create low source failed: %v", err)
	}
	high, err := config.NewMapSource("high", map[string]string{
		"app.name": "high",
	})
	if err != nil {
		t.Fatalf("create high source failed: %v", err)
	}
	env, err := config.NewEnvironment(low)
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	if err := env.AddFirst(high); err != nil {
		t.Fatalf("add source failed: %v", err)
	}

	name, ok := env.Get("app.name")
	if !ok || name != "high" {
		t.Fatalf("expected high priority value, got %q, %v", name, ok)
	}
	port, ok := env.Get("app.port")
	if !ok || port != "8080" {
		t.Fatalf("expected fallback value, got %q, %v", port, ok)
	}
	if got := env.GetOrDefault("missing", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestEnvironmentSnapshot_whenSourcesHaveOverlap_shouldFoldByPriority(t *testing.T) {
	low, err := config.NewMapSource("low", map[string]string{
		"a": "1",
		"b": "1",
	})
	if err != nil {
		t.Fatalf("create low source failed: %v", err)
	}
	high, err := config.NewMapSource("high", map[string]string{
		"b": "2",
	})
	if err != nil {
		t.Fatalf("create high source failed: %v", err)
	}
	env, err := config.NewEnvironment(high, low)
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}

	snapshot := env.Snapshot()
	expected := map[string]string{"a": "1", "b": "2"}
	if !reflect.DeepEqual(snapshot, expected) {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}
