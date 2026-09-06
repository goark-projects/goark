package env_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"goark.dev/goark/core/env"
	"goark.dev/goark/core/resource"
	arkerrors "goark.dev/goark/errors"
)

func TestLoadConfigPropertySource_whenYAMLExists_shouldParseAndFlatten(t *testing.T) {
	tempDir := t.TempDir()
	data := []byte(`
app:
  name: goark
  tags:
    - core
    - boot
server:
  port: 8080
feature:
  enabled: true
items:
  - name: first
`)
	if err := os.WriteFile(filepath.Join(tempDir, "app.yml"), data, 0o644); err != nil {
		t.Fatalf("write yml failed: %v", err)
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadConfigPropertySource(
		context.Background(),
		loader,
		"app.yml",
		env.WithPropertySourceName("app"),
	)
	if err != nil {
		t.Fatalf("load config property source failed: %v", err)
	}
	if source.Name() != "app" {
		t.Fatalf("unexpected property source name: %q", source.Name())
	}
	assertAnyProperty(t, source, "app.name", "goark")
	assertIntProperty(t, source, "server.port", 8080)
	assertAnyProperty(t, source, "feature.enabled", true)
	assertAnyProperty(t, source, "app.tags", []any{"core", "boot"})
	assertAnyProperty(t, source, "app.tags[0]", "core")
	assertAnyProperty(t, source, "items[0].name", "first")
}

func TestLoadPropertiesPropertySource_whenResourceExists_shouldParseProperties(t *testing.T) {
	tempDir := t.TempDir()
	data := []byte(`
# 应用配置
app.name = goark
server.port: 8080
feature.enabled true
escaped.key = hello\:world
multi.line = hello\
  world
unicode = Go\u0061rk
`)
	if err := os.WriteFile(filepath.Join(tempDir, "app.properties"), data, 0o644); err != nil {
		t.Fatalf("write properties failed: %v", err)
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadPropertiesPropertySource(
		context.Background(),
		loader,
		"app.properties",
		env.WithPropertySourceName("app"),
	)
	if err != nil {
		t.Fatalf("load property source failed: %v", err)
	}
	if source.Name() != "app" {
		t.Fatalf("unexpected property source name: %q", source.Name())
	}
	assertProperty(t, source, "app.name", "goark")
	assertProperty(t, source, "server.port", "8080")
	assertProperty(t, source, "feature.enabled", "true")
	assertProperty(t, source, "escaped.key", "hello:world")
	assertProperty(t, source, "multi.line", "helloworld")
	assertProperty(t, source, "unicode", "Goark")
}

func assertIntProperty(t *testing.T, source env.PropertySource, key string, want int64) {
	t.Helper()
	got, ok := source.GetProperty(key)
	if !ok {
		t.Fatalf("property %q not found", key)
	}
	value := reflect.ValueOf(got)
	if !value.IsValid() {
		t.Fatalf("property %q expected integer %d, got nil", key, want)
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() != want {
			t.Fatalf("property %q expected %d, got %#v", key, want, got)
		}
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		if value.Uint() != uint64(want) {
			t.Fatalf("property %q expected %d, got %#v", key, want, got)
		}
	default:
		t.Fatalf("property %q expected integer %d, got %#v", key, want, got)
	}
}
func TestLoadConfigPropertySource_whenExplicitNameIsBlank_shouldReturnError(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	_, err = env.LoadConfigPropertySource(
		context.Background(),
		loader,
		"app.yml",
		env.WithPropertySourceName(" "),
	)
	if err == nil {
		t.Fatal("expected blank property source name error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestLoadConfigPropertySource_whenMissingIgnored_shouldReturnNil(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadConfigPropertySource(
		context.Background(),
		loader,
		"missing",
		env.WithIgnoreResourceNotFound(true),
	)
	if err != nil {
		t.Fatalf("missing ignored should not fail: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil property source, got %#v", source)
	}
}

func assertProperty(t *testing.T, source env.PropertySource, key string, want string) {
	t.Helper()
	got, ok := source.GetProperty(key)
	if !ok {
		t.Fatalf("property %q not found", key)
	}
	if got != want {
		t.Fatalf("property %q expected %q, got %#v", key, want, got)
	}
}

func TestLoadProperties_whenUnicodeEscapeInvalid_shouldReturnError(t *testing.T) {
	_, err := env.ParseProperties([]byte("bad=\\u12xx"))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
