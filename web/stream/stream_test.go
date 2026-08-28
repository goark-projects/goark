package stream_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/stream"
)

type ssePayload struct {
	State string `json:"state"`
}

func TestResultStreamsTextAndFlushes(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/stream", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return stream.Text(func(_ context.Context, writer *stream.Writer) error {
			if _, err := writer.WriteString("hello "); err != nil {
				return err
			}
			if err := writer.Flush(); err != nil {
				return err
			}
			_, err := writer.WriteString("stream")
			return err
		}, stream.WithHeader("X-Stream", "hit")), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	recorder := serveStreamRouter(router, http.MethodGet, "/stream")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "hello stream" {
		t.Fatalf("body = %q, want streamed body", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("X-Stream") != "hit" {
		t.Fatalf("X-Stream = %q, want hit", recorder.Header().Get("X-Stream"))
	}
}

func TestEventsWritesServerSentEventFrames(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/events", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return stream.Events(func(_ context.Context, writer *stream.SSEWriter) error {
			return writer.Send(stream.Event{
				ID:      "42",
				Name:    "job",
				Retry:   2 * time.Second,
				Comment: "started",
				Data:    ssePayload{State: "ready"},
			})
		}), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	recorder := serveStreamRouter(router, http.MethodGet, "/events")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Header().Get("Content-Type") != stream.EventStreamContentType {
		t.Fatalf("Content-Type = %q, want event-stream", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", recorder.Header().Get("Cache-Control"))
	}
	body := recorder.Body.String()
	for _, fragment := range []string{
		": started\n",
		"id: 42\n",
		"event: job\n",
		"retry: 2000\n",
		"data: {\"state\":\"ready\"}\n\n",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("SSE body missing %q:\n%s", fragment, body)
		}
	}
}

func serveStreamRouter(router *arkweb.Router, method string, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}
