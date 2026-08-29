package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type modelAttributeSourcesCriteria struct {
	TenantID       string `form:"tenantId" json:"tenantId"`
	UserID         int64  `form:"userId" json:"userId"`
	RequestID      string `form:"xRequestId" json:"requestId"`
	AcceptLanguage string `json:"acceptLanguage"`
	Mode           string `form:"mode" json:"mode"`
}

func TestModelAttributeBindsPathVariablesAndHeadersWithRequestParameterPriority(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/tenants/{tenantId}/users/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (modelAttributeSourcesCriteria, error) {
		return mvc.ModelAttribute[modelAttributeSourcesCriteria](ctx)
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/tenants/core;scope=internal/users/42;role=admin?tenantId=query&mode=query", nil)
	request.Header.Set("X-Request-Id", "req-1")
	request.Header.Set("Accept-Language", "zh-CN")
	request.Header.Set("Mode", "header")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}

	var got modelAttributeSourcesCriteria
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.TenantID != "query" ||
		got.UserID != 42 ||
		got.RequestID != "req-1" ||
		got.AcceptLanguage != "zh-CN" ||
		got.Mode != "query" {
		t.Fatalf("criteria = %#v, want model attribute sources with request parameter priority", got)
	}
}

func TestModelAttributeFallbackSourcesDoNotSuppressUnrelatedHeaders(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("suppressedSources",
		mvc.GET("/users/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) ([]string, error) {
			_, result, err := mvc.ModelAttributeResult[binderSuppressedInput](ctx)
			if err != nil {
				return nil, err
			}
			return result.SuppressedFields(), nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields("name")
	}))
	if err := controller.Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/42?name=ada&admin=true", nil)
	request.Header.Set("User-Agent", "goark-test")
	request.Header.Set("X-Trace-Id", "trace-1")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got []string
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if len(got) != 1 || got[0] != "admin" {
		t.Fatalf("suppressed fields = %#v, want only bound request fields", got)
	}
}
