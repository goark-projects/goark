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

func TestLoadPropertiesPropertySource_whenMissingIgnored_shouldReturnNil(t *testing.T) {
	loader, err := resource.NewLoader(resource.WithFileBase(t.TempDir()))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	source, err := env.LoadPropertiesPropertySource(
		context.Background(),
		loader,
		"missing.properties",
		env.WithIgnoreResourceNotFound(true),
	)
	if err != nil {
		t.Fatalf("missing ignored should not fail: %v", err)
	}
	if source != nil {
		t.Fatalf("expected nil property source, got %#v", source)
	}
}

func TestLoadDefaultConfigPropertySource_whenOnlyAppYAMLExists_shouldNotDiscoverIt(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "app.yaml"), []byte("selected: yaml\n"),
		0o644); err != nil {
		t.Fatalf("write app.yaml failed: %v", err)
	}
	loader, err := resource.NewLoader(resource.WithFileBase(tempDir))
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}
	if _, err := env.LoadDefaultConfigPropertySource(context.Background(), loader); !arkerrors.Is(
		err,
		arkerrors.CodeNotFound,
	) {
		t.Fatalf("LoadDefaultConfigPropertySource() error = %v, want not found", err)
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

func TestLoadProperties_whenUnicodeSurrogatePairInvalid_shouldReturnError(t *testing.T) {
	_, err := env.ParseProperties([]byte("bad=\\uD83D\\u0041"))
	if err == nil {
		t.Fatal("expected surrogate pair parse error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
