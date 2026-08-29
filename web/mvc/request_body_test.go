package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
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

func TestRequestBodyReadsTextByContentType(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.Text(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
		return mvc.RequestBody[string](ctx)
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("plain"))
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "plain" {
		t.Fatalf("body = %q, want plain", recorder.Body.String())
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

func TestBindBodyReadsTextRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.BindBody(http.StatusCreated, func(_ *arkweb.Context, input string) (map[string]string, error) {
		return map[string]string{"body": input}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("arkhos"))
	request.Header.Set("Content-Type", "text/plain")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"body":"arkhos"`) {
		t.Fatalf("body = %s, want bound text JSON response", recorder.Body.String())
	}
}

func TestBindBodyEntityReadsBytesRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.BindBodyEntity(func(_ *arkweb.Context, input []byte) (web.ResponseEntity[map[string]int], error) {
		return web.Status(http.StatusAccepted, map[string]int{"length": len(input)}), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("bytes"))
	request.Header.Set("Content-Type", "application/octet-stream")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"length":5`) {
		t.Fatalf("body = %s, want byte length response", recorder.Body.String())
	}
}
