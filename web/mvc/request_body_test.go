package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc"
)

func TestRequestBodyBindsJSONWithoutValidation(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.RequestBody[createRequest](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":""`) {
		t.Fatalf("body = %s, want zero-value bound payload", recorder.Body.String())
	}
}

func TestValidatedRequestBodyUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.ValidatedRequestBody[createRequest](ctx, "create")
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader(`{"name":"arkarta"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}
