package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
)

func TestAttachmentStreamsDownload(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/reports/today", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Attachment("reports/today.csv", strings.NewReader("id,name\n1,goark\n"),
			web.WithDownloadContentType("text/csv"),
			web.WithDownloadContentLength(16),
		), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/reports/today", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "id,name\n1,goark\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("Content-Type = %q, want text/csv", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "16" {
		t.Fatalf("Content-Length = %q, want 16", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename=today.csv` {
		t.Fatalf("Content-Disposition = %q", got)
	}
}

func TestAttachmentDoesNotReadBodyForHead(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/reports/today", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Attachment("today.csv", failReadDownload{}), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/reports/today", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", recorder.Body.Len())
	}
}

type failReadDownload struct{}

func (failReadDownload) Read([]byte) (int, error) {
	return 0, errors.New("download body must not be read")
}
