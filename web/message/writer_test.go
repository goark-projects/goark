package message_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

func TestWriterWritesStringByAcceptNegotiation(t *testing.T) {
	t.Parallel()

	recorder := serveMessage(t, "text/plain", func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusAccepted, "ok")
	})

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", contentType)
	}
	if recorder.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", recorder.Body.String())
	}
}

func TestWriterWritesJSONWithSonicDefault(t *testing.T) {
	t.Parallel()

	recorder := serveMessage(t, arkjson.ContentType, func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusCreated, map[string]string{"name": "goark"}, arkjson.ContentType)
	})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != arkjson.ContentType {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json invalid: %v", err)
	}
	if payload["name"] != "goark" {
		t.Fatalf("payload = %#v, want name", payload)
	}
}

func TestWriterReturnsNotAcceptable(t *testing.T) {
	t.Parallel()

	recorder := serveMessage(t, "application/xml", func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusOK, map[string]string{"name": "goark"}, arkjson.ContentType)
	})

	if recorder.Code != http.StatusNotAcceptable {
		t.Fatalf("status = %d, want 406, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestWriterStreamsReader(t *testing.T) {
	t.Parallel()

	recorder := serveMessage(t, message.MediaTypeOctetStream, func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusOK, strings.NewReader("stream"))
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "stream" {
		t.Fatalf("body = %q, want stream", recorder.Body.String())
	}
}

func TestWriterWritesBytes(t *testing.T) {
	t.Parallel()

	recorder := serveMessage(t, message.MediaTypeOctetStream, func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusOK, []byte("bytes"))
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "bytes" {
		t.Fatalf("body = %q, want bytes", recorder.Body.String())
	}
}

func serveMessage(t *testing.T, accept string, fn func(ctx *arkweb.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/messages", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := fn(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/messages", nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	nethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
