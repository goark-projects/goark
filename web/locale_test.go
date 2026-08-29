package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
)

func TestRequestLocaleReadsAcceptLanguageAndWritesContentLanguage(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := registry.GET("/locale", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		locale, ok := web.RequestLocale(ctx)
		locales := web.RequestLocales(ctx)
		payload := map[string]any{
			"ok":         ok,
			"locale":     locale.Tag(),
			"language":   locale.Language(),
			"region":     locale.Region(),
			"localeSize": len(locales),
		}
		return web.OK(payload).WithContentLanguage(locale), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/locale", nil)
	request.Header.Set("Accept", arkjson.ContentType)
	request.Header.Set("Accept-Language", "en-US;q=0.8, zh-CN;q=0.9")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Language"); got != "zh-CN" {
		t.Fatalf("Content-Language = %q, want zh-CN", got)
	}
	var payload map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["ok"] != true || payload["locale"] != "zh-CN" || payload["language"] != "zh" || payload["region"] != "CN" || payload["localeSize"] != float64(2) {
		t.Fatalf("payload = %#v, want request locale details", payload)
	}
}
