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

	source, err := env.LoadPropertiesPropertySource(context.Background(), loader, "app.properties", env.WithPropertySourceName("app"))
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

	source, err := env.LoadConfigPropertySource(context.Background(), loader, "app.yml", env.WithPropertySourceName("app"))
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

func TestLoadConfigPropertySource_whenTOMLExists_shouldParseAndFlatten(t *testing.T) {
	tempDir := t.TempDir()
	data := []byte(`
[app]
name = "goark"
tags = ["core", "toml"]

[server]
port = 9090

[feature]
enabled = true
`)
	if err := os.WriteFile(filepath.Join(tempDir, "app.toml"), data, 0o644); err != nil {
		t.Fatalf("write toml failed: %v", err)
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadConfigPropertySource(context.Background(), loader, "app.toml")
	if err != nil {
		t.Fatalf("load config property source failed: %v", err)
	}
	assertAnyProperty(t, source, "app.name", "goark")
	assertIntProperty(t, source, "server.port", 9090)
	assertAnyProperty(t, source, "feature.enabled", true)
	assertAnyProperty(t, source, "app.tags[1]", "toml")
}

func TestLoadConfigPropertySource_whenLocationHasNoExtension_shouldUsePriority(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantSource string
		wantValue  string
	}{
		{
			name: "yml first",
			files: map[string]string{
				"app.yml":        "selected: yml\n",
				"app.properties": "selected=properties\n",
				"app.toml":       "selected = \"toml\"\n",
			},
			wantSource: "app.yml",
			wantValue:  "yml",
		},
		{
			name: "properties before toml",
			files: map[string]string{
				"app.properties": "selected=properties\n",
				"app.toml":       "selected = \"toml\"\n",
			},
			wantSource: "app.properties",
			wantValue:  "properties",
		},
		{
			name: "toml last",
			files: map[string]string{
				"app.toml": "selected = \"toml\"\n",
			},
			wantSource: "app.toml",
			wantValue:  "toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			for name, data := range tt.files {
				if err := os.WriteFile(filepath.Join(tempDir, name), []byte(data), 0o644); err != nil {
					t.Fatalf("write %s failed: %v", name, err)
				}
			}
			loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
			if err != nil {
				t.Fatalf("create loader failed: %v", err)
			}

			source, err := env.LoadConfigPropertySource(context.Background(), loader, "app")
			if err != nil {
				t.Fatalf("load config property source failed: %v", err)
			}
			if source.Name() != tt.wantSource {
				t.Fatalf("expected source %q, got %q", tt.wantSource, source.Name())
			}
			assertAnyProperty(t, source, "selected", tt.wantValue)
		})
	}
}

func TestLoadDefaultConfigPropertySource_whenOnlyAppYAMLExists_shouldNotDiscoverIt(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "app.yaml"), []byte("selected: yaml\n"), 0o644); err != nil {
		t.Fatalf("write app.yaml failed: %v", err)
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}
	if _, err := env.LoadDefaultConfigPropertySource(context.Background(), loader); !arkerrors.Is(err, arkerrors.CodeNotFound) {
		t.Fatalf("LoadDefaultConfigPropertySource() error = %v, want not found", err)
	}
}

func TestLoadDefaultConfigPropertySource_shouldUseAppBaseName(t *testing.T) {
	tempDir := t.TempDir()
	files := map[string]string{
		"app.yaml":       "selected: yaml\n",
		"app.properties": "selected=properties\n",
		"app.toml":       "selected = \"toml\"\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadDefaultConfigPropertySource(context.Background(), loader)
	if err != nil {
		t.Fatalf("load default config property source failed: %v", err)
	}
	if source.Name() != "app.properties" {
		t.Fatalf("expected app.properties source, got %q", source.Name())
	}
	assertAnyProperty(t, source, "selected", "properties")
}

func TestLoadConfigPropertySource_whenExplicitNameIsBlank_shouldReturnError(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	_, err = env.LoadConfigPropertySource(context.Background(), loader, "app.yml", env.WithPropertySourceName(" "))
	if err == nil {
		t.Fatal("expected blank property source name error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestLoadPropertiesPropertySource_whenMissingIgnored_shouldReturnNil(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadPropertiesPropertySource(context.Background(), loader, "missing.properties", env.WithIgnoreResourceNotFound(true))
	if err != nil {
		t.Fatalf("missing ignored should not fail: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil property source, got %#v", source)
	}
}

func TestLoadConfigPropertySource_whenMissingIgnored_shouldReturnNil(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadConfigPropertySource(context.Background(), loader, "missing", env.WithIgnoreResourceNotFound(true))
	if err != nil {
		t.Fatalf("missing ignored should not fail: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil property source, got %#v", source)
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

func TestLoadProperties_whenUnicodeSurrogatePairInvalid_shouldReturnError(t *testing.T) {
	_, err := env.ParseProperties([]byte("bad=\\uD83D\\u0041"))
	if err == nil {
		t.Fatal("expected surrogate pair parse error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
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

func assertAnyProperty(t *testing.T, source env.PropertySource, key string, want any) {
	t.Helper()
	got, ok := source.GetProperty(key)
	if !ok {
		t.Fatalf("property %q not found", key)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("property %q expected %#v, got %#v", key, want, got)
	}
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
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if value.Uint() != uint64(want) {
			t.Fatalf("property %q expected %d, got %#v", key, want, got)
		}
	default:
		t.Fatalf("property %q expected integer %d, got %#v", key, want, got)
	}
}
