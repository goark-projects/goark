package locale_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/locale"
)

type localePayload struct {
	OK         bool   `json:"ok"`
	Locale     string `json:"locale"`
	Language   string `json:"language"`
	Region     string `json:"region"`
	LocaleSize int    `json:"localeSize"`
}

func TestResolverInterceptorOverridesRequestLocale(t *testing.T) {
	t.Parallel()

	zhCN, ok := servlet.NewLocale("zh-CN")
	if !ok {
		t.Fatal("NewLocale failed")
	}
	resolver, err := locale.NewFixedResolver(zhCN)
	if err != nil {
		t.Fatalf("NewFixedResolver failed: %v", err)
	}
	interceptor, err := locale.ResolverInterceptor(resolver)
	if err != nil {
		t.Fatalf("ResolverInterceptor failed: %v", err)
	}
	registry := newLocaleRegistry(t, interceptor)

	recorder := serveLocale(t, registry, "/locale", "en-US")
	if recorder.Header().Get("Content-Language") != "zh-CN" {
		t.Fatalf("Content-Language = %q, want zh-CN", recorder.Header().Get("Content-Language"))
	}
	payload := readLocalePayload(t, recorder)
	if !payload.OK || payload.Locale != "zh-CN" || payload.Language != "zh" || payload.Region != "CN" || payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want fixed locale before accepted locale", payload)
	}
}

func TestChangeInterceptorUsesLocaleParameter(t *testing.T) {
	t.Parallel()

	interceptor, err := locale.ChangeInterceptor()
	if err != nil {
		t.Fatalf("ChangeInterceptor failed: %v", err)
	}
	registry := newLocaleRegistry(t, interceptor)

	recorder := serveLocale(t, registry, "/locale?locale=ja-JP", "en-US")
	if recorder.Header().Get("Content-Language") != "ja-JP" {
		t.Fatalf("Content-Language = %q, want ja-JP", recorder.Header().Get("Content-Language"))
	}
	payload := readLocalePayload(t, recorder)
	if !payload.OK || payload.Locale != "ja-JP" || payload.Language != "ja" || payload.Region != "JP" || payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want changed locale before accepted locale", payload)
	}
}

func newLocaleRegistry(t *testing.T, interceptor arkweb.Interceptor) *web.Registry {
	t.Helper()
	registry := web.NewRegistry()
	registry.Use(interceptor)
	if err := registry.GET("/locale", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		current, ok := web.RequestLocale(ctx)
		locales := web.RequestLocales(ctx)
		return web.OK(localePayload{
			OK:         ok,
			Locale:     current.Tag(),
			Language:   current.Language(),
			Region:     current.Region(),
			LocaleSize: len(locales),
		}).WithContentLanguage(current), nil
	})); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	return registry
}

func serveLocale(t *testing.T, registry *web.Registry, target string, acceptLanguage string) *httptest.ResponseRecorder {
	t.Helper()
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Header.Set("Accept", arkjson.ContentType)
	request.Header.Set("Accept-Language", acceptLanguage)
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

func readLocalePayload(t *testing.T, recorder *httptest.ResponseRecorder) localePayload {
	t.Helper()
	var payload localePayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	return payload
}
