package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
)

func TestMessageResultWritesNegotiatedString(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/messages", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return web.Message(http.StatusOK, "hello"), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/messages", nil)
	request.Header.Set("Accept", "text/plain")
	nethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != message.MediaTypeTextPlain {
		t.Fatalf("content type = %q, want text/plain", recorder.Header().Get("Content-Type"))
	}
	if recorder.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", recorder.Body.String())
	}
}
