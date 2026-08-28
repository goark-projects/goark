package resource_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"goark.dev/goark/core/resource"
)

func TestMemoryResource_whenReadAllCalled_shouldReturnCopiedBytes(t *testing.T) {
	ctx := context.Background()
	res, err := resource.NewMemoryResource("config.yaml", []byte("name: goark"))
	if err != nil {
		t.Fatalf("create memory resource failed: %v", err)
	}

	first, err := res.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read memory resource failed: %v", err)
	}
	first[0] = 'N'
	second, err := res.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read memory resource again failed: %v", err)
	}
	if string(second) != "name: goark" {
		t.Fatalf("resource data should be immutable, got %q", string(second))
	}
}

func TestDefaultLoader_whenLocationHasKnownSchemes_shouldLoadResources(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "app.properties")
	if err := os.WriteFile(filePath, []byte("app.name=goark"), 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	filesystem := fstest.MapFS{
		"config/app.yaml": &fstest.MapFile{Data: []byte("app:\n  name: goark\n")},
	}
	loader, err := resource.NewLoader(
		resource.WithFileBase(tempDir),
		resource.WithFS("embedded", filesystem),
		resource.WithMemory("defaults", []byte("debug=false")),
	)
	if err != nil {
		t.Fatalf("create loader failed: %v", err)
	}

	cases := []struct {
		location string
		contains string
	}{
		{location: "app.properties", contains: "app.name=goark"},
		{location: "fs:embedded/config/app.yaml", contains: "name: goark"},
		{location: "memory:defaults", contains: "debug=false"},
	}
	for _, item := range cases {
		res, err := loader.Load(item.location)
		if err != nil {
			t.Fatalf("load %s failed: %v", item.location, err)
		}
		data, err := res.ReadAll(ctx)
		if err != nil {
			t.Fatalf("read %s failed: %v", item.location, err)
		}
		if !strings.Contains(string(data), item.contains) {
			t.Fatalf("expected %s to contain %q, got %q", item.location, item.contains, string(data))
		}
	}
}

func TestURLResource_whenServerReturnsSuccess_shouldReadResponse(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("remote-config"))
	}))
	defer server.Close()

	res, err := resource.NewURLResource(server.URL+"/config", server.Client())
	if err != nil {
		t.Fatalf("create url resource failed: %v", err)
	}
	data, err := res.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read url resource failed: %v", err)
	}
	if string(data) != "remote-config" {
		t.Fatalf("unexpected url resource data: %q", string(data))
	}
}
