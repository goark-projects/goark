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

type requestEntityCreateRequest struct {
	Name string `json:"name" arkarta:"required"`
}

func TestRequestEntityBindsJSONAndMetadata(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs/42", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		entity, err := mvc.RequestEntity[requestEntityCreateRequest](ctx)
		if err != nil {
			return nil, err
		}
		body, ok := entity.Body()
		return map[string]any{
			"name":        body.Name,
			"hasBody":     ok && entity.HasBody(),
			"method":      entity.Method(),
			"url":         entity.URL(),
			"requestURI":  entity.RequestURI(),
			"path":        entity.Path(),
			"traceHeader": entity.Headers().Get("X-Trace-Id"),
		}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/jobs/42?trace=1", strings.NewReader(`{"name":"sync"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	request.Header.Set("X-Trace-Id", "trace-1")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	assertStringValue(t, body, "name", "sync")
	assertBoolValue(t, body, "hasBody", true)
	assertStringValue(t, body, "method", http.MethodPost)
	assertStringValue(t, body, "url", "https://api.example.com/jobs/42?trace=1")
	assertStringValue(t, body, "requestURI", "/jobs/42")
	assertStringValue(t, body, "path", "/jobs/42")
	assertStringValue(t, body, "traceHeader", "trace-1")
}

func TestRequestEntityReturnsMetadataWithoutBody(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/jobs", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]any, error) {
		entity, err := mvc.RequestEntity[requestEntityCreateRequest](ctx)
		if err != nil {
			return nil, err
		}
		_, hasBody := entity.Body()
		return map[string]any{
			"hasBody": hasBody,
			"method":  entity.Method(),
			"path":    entity.Path(),
		}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://api.example.com/jobs", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	assertBoolValue(t, body, "hasBody", false)
	assertStringValue(t, body, "method", http.MethodGet)
	assertStringValue(t, body, "path", "/jobs")
}

func TestValidatedRequestEntityUsesValidationGroups(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		entity, err := mvc.ValidatedRequestEntity[createRequest](ctx, "create")
		if err != nil {
			return nil, err
		}
		body, _ := entity.Body()
		return map[string]string{"name": body.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"goark"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRequestEntityWithMediaTypesRejectsUnsupportedContentType(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		entity, err := mvc.RequestEntityWithMediaTypes[requestEntityCreateRequest](ctx, arkjson.ContentType)
		if err != nil {
			return nil, err
		}
		body, _ := entity.Body()
		return map[string]string{"name": body.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader("name=goark"))
	request.Header.Set("Content-Type", "text/plain")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindRequestEntityWritesJSONResponse(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.BindRequestEntity(http.StatusCreated,
		func(_ *arkweb.Context, entity web.RequestEntity[requestEntityCreateRequest]) (map[string]string, error) {
			body, _ := entity.Body()
			return map[string]string{"method": entity.Method(), "name": body.Name}, nil
		})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"sync"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"method":"POST"`) ||
		!strings.Contains(recorder.Body.String(), `"name":"sync"`) {
		t.Fatalf("body = %s, want request entity response", recorder.Body.String())
	}
}

func TestBindRequestEntityEntityWritesResponseEntity(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.BindRequestEntityEntity(
		func(_ *arkweb.Context, entity web.RequestEntity[requestEntityCreateRequest]) (web.ResponseEntity[map[string]string], error) {
			body, _ := entity.Body()
			return web.Created("/jobs/42", map[string]string{"name": body.Name}), nil
		})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"sync"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != "/jobs/42" {
		t.Fatalf("Location = %q, want /jobs/42", got)
	}
}

func TestBindRequestEntityGroupsUsesValidationGroups(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.BindRequestEntityGroups(http.StatusAccepted,
		func(_ *arkweb.Context, entity web.RequestEntity[createRequest]) (map[string]string, error) {
			body, _ := entity.Body()
			return map[string]string{"name": body.Name}, nil
		}, "create")); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"goark"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", recorder.Code, recorder.Body.String())
	}
}

func assertStringValue(t *testing.T, body map[string]any, name, want string) {
	t.Helper()
	got, ok := body[name].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", name, body[name], want)
	}
}

func assertBoolValue(t *testing.T, body map[string]any, name string, want bool) {
	t.Helper()
	got, ok := body[name].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %t", name, body[name], want)
	}
}
