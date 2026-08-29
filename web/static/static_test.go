package static_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	servletresource "goark.dev/arkarta/servlet/resource"
	"goark.dev/goark/web"
	"goark.dev/goark/web/static"
)

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
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", contentType)
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

func TestConfigurerAppliesCacheControlToStaticResource(t *testing.T) {
	t.Parallel()

	configurer, err := static.New("/assets/*", fstest.MapFS{
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('goark')"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}, static.WithCacheMaxAge(time.Hour))
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := web.NewRegistry()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveStaticRegistry(t, registry, http.MethodGet, "/assets/app.js")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("Cache-Control = %q, want public max-age", got)
	}
}

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
	configurer, err := static.New("/assets/*", root, static.WithContentVersioning(), static.WithCacheMaxAge(time.Hour))
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

func TestConfigurerServesFixedVersionedResource(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('goark')"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}
	versioned, err := static.FixedVersionPath("v1", "app.js")
	if err != nil {
		t.Fatalf("FixedVersionPath failed: %v", err)
	}
	configurer, err := static.New("/assets/*", root, static.WithFixedVersion("v1"))
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
}

func TestConfigurerServesCombinedVersionedResource(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('goark')"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	}
	contentPath, err := static.ContentVersionPath(t.Context(), root, "app.js")
	if err != nil {
		t.Fatalf("ContentVersionPath failed: %v", err)
	}
	versioned, err := static.FixedVersionPath("v1", contentPath)
	if err != nil {
		t.Fatalf("FixedVersionPath failed: %v", err)
	}
	configurer, err := static.New("/assets/*", root, static.WithContentVersioning(), static.WithFixedVersion("v1"))
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
}

func TestContentVersionPathRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	root := fstest.MapFS{"app.js": &fstest.MapFile{Data: []byte("ok")}}
	if _, err := static.ContentVersionPath(t.Context(), root, "../app.js"); !errors.Is(err, static.ErrInvalidResourcePath) {
		t.Fatalf("err = %v, want ErrInvalidResourcePath", err)
	}
	if _, err := static.ContentVersionPath(t.Context(), nil, "app.js"); !errors.Is(err, servletresource.ErrNilFileSystem) {
		t.Fatalf("err = %v, want ErrNilFileSystem", err)
	}
	if _, err := static.FixedVersionPath("../v1", "app.js"); !errors.Is(err, static.ErrInvalidResourceVersion) {
		t.Fatalf("err = %v, want ErrInvalidResourceVersion", err)
	}
}

func TestConfigurerAppliesGlobalFiltersToStaticResource(t *testing.T) {
	t.Parallel()

	configurer, err := static.New("/assets/*", fstest.MapFS{
		"app.txt": &fstest.MapFile{
			Data:    []byte("filtered"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	})
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := web.NewRegistry()
	registry.AddFilter(servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		res.Header().Set("X-Static-Filter", "hit")
		return chain.Next(ctx, req, res)
	}))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	recorder := serveStaticRegistry(t, registry, http.MethodGet, "/assets/app.txt")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("X-Static-Filter"); got != "hit" {
		t.Fatalf("X-Static-Filter = %q, want hit", got)
	}
}

func TestNewRejectsNilFileSystem(t *testing.T) {
	t.Parallel()

	_, err := static.New("/assets/*", nil)
	if !errors.Is(err, servletresource.ErrNilFileSystem) {
		t.Fatalf("err = %v, want ErrNilFileSystem", err)
	}
}

func serveStaticRegistry(t *testing.T, registry *web.Registry, method string, target string) *httptest.ResponseRecorder {
	t.Helper()
	deployment, err := web.BuildDeployment(registry, web.DeploymentSpec{})
	if err != nil {
		t.Fatalf("BuildDeployment failed: %v", err)
	}
	if !servletcontainer.SupportsProfile(deployment.Profiles(), servletcontainer.ProfileCore) {
		t.Fatal("deployment should keep core profile")
	}
	handler, err := deployment.Handler()
	if err != nil {
		t.Fatalf("Deployment Handler failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
