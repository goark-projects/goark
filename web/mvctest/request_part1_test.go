package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

func TestRequestEntityBindsJSONAndMetadata(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs/42", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
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
	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/jobs/42?trace=1",
		strings.NewReader(`{"name":"sync"}`))
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
	if err := router.Handle(http.MethodGet, "/jobs", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
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
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"https://api.example.com/jobs", nil))

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

func TestOptionalValidatedRequestBodySkipsMissingBody(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required"`
	}
	type payload struct {
		Present bool   `json:"present"`
		Name    string `json:"name"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body/optional", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		input, present, err := mvc.OptionalValidatedRequestBody[createRequest](ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Present: present, Name: input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body/optional", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != `{"present":false,"name":""}` {
		t.Fatalf("body = %s, want absent optional body", recorder.Body.String())
	}
}

func TestValidatedRequestEntityUsesValidationGroups(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
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

func TestValidatedRequestBodyUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
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

func TestBindRequestEntityWritesJSONResponse(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.BindRequestEntity(http.StatusCreated,
		func(_ *arkweb.Context, entity web.RequestEntity[requestEntityCreateRequest]) (
			map[string]string, error) {
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
		func(_ *arkweb.Context, entity web.RequestEntity[requestEntityCreateRequest]) (
			web.ResponseEntity[map[string]string], error) {
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

func TestBindBodyReadsURLEncodedFormRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.BindBody(http.StatusCreated, func(
		_ *arkweb.Context, input url.Values) (map[string][]string, error) {
		return map[string][]string(input), nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader(
		"name=goark&tag=web&tag=mvc"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":["goark"]`) ||
		!strings.Contains(recorder.Body.String(), `"tag":["web","mvc"]`) {
		t.Fatalf("body = %s, want bound form JSON response", recorder.Body.String())
	}
}

func TestRequestBodyReadsTextByContentType(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.Text(http.StatusOK, func(
		ctx *arkweb.Context) (string, error) {
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

func TestBindBodyEntityReadsBytesRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.BindBodyEntity(func(_ *arkweb.Context,
		input []byte) (web.ResponseEntity[map[string]int], error) {
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

func httptestResponse(router *arkweb.Router, method string, target string,
	body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", arkjson.ContentType)
	request.Header.Set("Accept", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	return recorder
}

func assertStringValue(t *testing.T, body map[string]any, name, want string) {
	t.Helper()
	got, ok := body[name].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", name, body[name], want)
	}
}

type advisedCreateRequest struct {
	Name string `json:"name"`
}

func (*panicRequestBodyAdvice) AfterRead(*arkweb.Context, message.ReadAdviceContext) error {
	panic("nil request body advice must not run")
}

type panicRequestBodyAdvice struct{}
