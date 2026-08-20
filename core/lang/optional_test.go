package lang_test

import (
	"testing"

	"github.com/goark-projects/goark/core/lang"
)

func TestOptional_whenValueIsPresent_shouldReturnValue(t *testing.T) {
	optional := lang.Some("goark")

	value, ok := optional.Value()
	if !ok || value != "goark" {
		t.Fatalf("unexpected optional value: %q, %v", value, ok)
	}
	if optional.OrElse("fallback") != "goark" {
		t.Fatal("expected present value")
	}
	if optional.Ptr() == nil || *optional.Ptr() != "goark" {
		t.Fatal("expected pointer to present value")
	}
}

func TestOptional_whenValueIsMissing_shouldReturnFallback(t *testing.T) {
	optional := lang.None[string]()

	if !optional.Empty() || optional.Present() {
		t.Fatal("expected empty optional")
	}
	if optional.OrElse("fallback") != "fallback" {
		t.Fatal("expected fallback value")
	}
	if optional.OrElseGet(func() string { return "computed" }) != "computed" {
		t.Fatal("expected computed fallback value")
	}
	if optional.Ptr() != nil {
		t.Fatal("empty optional pointer should be nil")
	}
}
