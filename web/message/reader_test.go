package message_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

func TestReaderReadsStructuredJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}
	var got payload
	recorder := serveReadMessage(t, "application/vnd.goark+json; charset=utf-8", `{"name":"arkarta"}`, func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", recorder.Code, recorder.Body.String())
	}
	if got.Name != "arkarta" {
		t.Fatalf("name = %q, want arkarta", got.Name)
	}
}

func TestReaderReadsTextBody(t *testing.T) {
	t.Parallel()

	var got string
	recorder := serveReadMessage(t, "text/html; charset=utf-8", "<b>ok</b>", func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.Text(http.StatusOK, got), nil
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got != "<b>ok</b>" || recorder.Body.String() != "<b>ok</b>" {
		t.Fatalf("body = %q/%q, want text payload", got, recorder.Body.String())
	}
}

func TestReaderReadsRawBytes(t *testing.T) {
	t.Parallel()

	var got []byte
	recorder := serveReadMessage(t, "application/octet-stream", "raw", func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.Text(http.StatusOK, string(got)), nil
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Equal(got, []byte("raw")) {
		t.Fatalf("bytes = %q, want raw", string(got))
	}
}

func TestReaderRejectsUnsupportedRequestContentType(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}
	recorder := serveReadMessage(t, "application/xml", `<name>ark</name>`, func(ctx *arkweb.Context) (arkweb.Result, error) {
		var got payload
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestReaderReturnsNilTargetError(t *testing.T) {
	t.Parallel()

	var got *string
	recorder := serveReadMessage(t, "text/plain", "bad", func(ctx *arkweb.Context) (arkweb.Result, error) {
		err := message.NewReader().Read(ctx, got)
		if !errors.Is(err, arkjson.ErrNilTarget) {
			t.Fatalf("error = %v, want ErrNilTarget", err)
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", recorder.Code, recorder.Body.String())
	}
}

func serveReadMessage(t *testing.T, contentType string, body string, fn func(ctx *arkweb.Context) (arkweb.Result, error)) *httptest.ResponseRecorder {
	t.Helper()
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/messages", arkweb.HandlerFunc(fn)); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	nethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}
