package static_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	servletresource "goark.dev/arkarta/servlet/resource"
	"goark.dev/goark/web"
	"goark.dev/goark/web/static"
)

func TestConfigurerServesContentVersionedResource(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('goark')"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}
	versioned, err := static.ContentVersionPath(t.Context(), root, "app.js")
	if err != nil {
		t.Fatalf("ContentVersionPath failed: %v", err)
	}
	configurer, err := static.New(
		"/assets/*",
		root,
		static.WithContentVersioning(),
		static.WithCacheMaxAge(time.Hour),
	)
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := web.NewRegistry()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveStaticRegistry(t, registry, http.MethodGet, "/assets/"+versioned)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "console.log('goark')" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("Cache-Control = %q, want public max-age", got)
	}

	missing := serveStaticRegistry(t, registry, http.MethodGet, "/assets/app-deadbeef.js")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}
}

func TestConfigurerServesStaticResource(t *testing.T) {
	t.Parallel()

	configurer, err := static.New("/assets/*", fstest.MapFS{
		"app.txt": &fstest.MapFile{
			Data:    []byte("hello static"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	})
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := web.NewRegistry()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveStaticRegistry(t, registry, http.MethodGet, "/assets/app.txt")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "hello static" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(
		contentType,
		"text/plain",
	) {
		t.Fatalf("Content-Type = %q, want text/plain", contentType)
	}
}

func TestResourceURLProviderBuildsVersionedURL(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('goark')"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}
	provider, err := static.NewURLProvider(root,
		static.WithURLPathPrefix("/assets"),
		static.WithURLContentVersioning(),
		static.WithURLFixedVersion("v1"),
	)
	if err != nil {
		t.Fatalf("NewURLProvider failed: %v", err)
	}

	got, err := provider.URL(t.Context(), "app.js")
	if err != nil {
		t.Fatalf("URL failed: %v", err)
	}
	if !strings.HasPrefix(got, "/assets/v1/app-") || !strings.HasSuffix(got, ".js") {
		t.Fatalf("url = %q, want versioned assets URL", got)
	}
}

func TestConfigurerServesWelcomeFile(t *testing.T) {
	t.Parallel()

	configurer, err := static.New("/docs/*", fstest.MapFS{
		"home.html": &fstest.MapFile{
			Data:    []byte("<h1>docs</h1>"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}, static.WithWelcomeFiles("home.html"), static.WithServletName("docsStatic"))
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := web.NewRegistry()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveStaticRegistry(t, registry, http.MethodGet, "/docs/")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "<h1>docs</h1>" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestContentVersionPathRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{"app.js": &fstest.MapFile{Data: []byte("ok")}}
	if _, err := static.ContentVersionPath(t.Context(), root, "../app.js"); !errors.Is(
		err,
		static.ErrInvalidResourcePath,
	) {
		t.Fatalf("err = %v, want ErrInvalidResourcePath", err)
	}
	if _, err := static.ContentVersionPath(t.Context(), nil, "app.js"); !errors.Is(
		err,
		servletresource.ErrNilFileSystem,
	) {
		t.Fatalf("err = %v, want ErrNilFileSystem", err)
	}
	if _, err := static.FixedVersionPath("../v1", "app.js"); !errors.Is(
		err,
		static.ErrInvalidResourceVersion,
	) {
		t.Fatalf("err = %v, want ErrInvalidResourceVersion", err)
	}
}

func TestResourceURLProviderRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	provider, err := static.NewURLProvider(nil, static.WithURLPathPrefix("/assets"))
	if err != nil {
		t.Fatalf("NewURLProvider failed: %v", err)
	}
	if _, err := provider.URL(t.Context(), "../app.js"); !errors.Is(
		err,
		static.ErrInvalidResourcePath,
	) {
		t.Fatalf("err = %v, want ErrInvalidResourcePath", err)
	}
}
