package mvc_test

import (
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

func TestControllerAdviceRequestBodyAdviceAdvisesBindJSON(t *testing.T) {
	t.Parallel()

	var typedNil *panicRequestBodyAdvice
	registry := web.NewRegistry()
	advice := mvc.NewRestControllerAdvice("api-bodies").WithRequestBodyAdvice(nil, typedNil,
		web.RequestBodyAdviceFunc{
			After: func(_ *arkweb.Context, input web.RequestBodyAdviceContext) error {
				target := input.Target.(*advisedCreateRequest)
				target.Name += "-controller-advised"
				return nil
			},
		})
	configurer := mvc.NewConfigurer(mvc.NewRestController("users",
		mvc.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context,
			input advisedCreateRequest) (map[string]string, error) {
			return map[string]string{"name": input.Name}, nil
		})),
	)).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptestResponse(router, http.MethodPost, "/users", `{"name":"goark"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"goark-controller-advised"`) {
		t.Fatalf("body = %s, want controller advice request body", recorder.Body.String())
	}
}

func TestOptionalRequestBodyBindsPresentBody(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name"`
	}
	type payload struct {
		Present bool   `json:"present"`
		Name    string `json:"name"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body/optional", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		input, present, err := mvc.OptionalRequestBodyWithMediaTypes[createRequest](ctx,
			arkjson.ContentType)
		if err != nil {
			return payload{}, err
		}
		return payload{Present: present, Name: input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/body/optional", strings.NewReader(
		`{"name":"goark"}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != `{"present":true,"name":"goark"}` {
		t.Fatalf("body = %s, want present optional body", recorder.Body.String())
	}
}

func TestBindJSONUsesRequestBodyAdvice(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.UseRequestBodyAdvice(message.ReadAdviceFunc{
		After: func(_ *arkweb.Context, input message.ReadAdviceContext) error {
			target := input.Target.(*advisedCreateRequest)
			target.Name += "-advised"
			return nil
		},
	})
	if err := mvc.NewRestController("users",
		mvc.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context,
			input advisedCreateRequest) (map[string]string, error) {
			return map[string]string{"name": input.Name}, nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptestResponse(router, http.MethodPost, "/users", `{"name":"goark"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"goark-advised"`) {
		t.Fatalf("body = %s, want advised request body", recorder.Body.String())
	}
}

func TestRequestBodyBindsJSONWithoutValidation(t *testing.T) {
	t.Parallel()

	type createRequest struct {
		Name string `json:"name" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
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

func TestRequestPartJSONBindsTypedPart(t *testing.T) {
	t.Parallel()

	type metadata struct {
		Name string `json:"name" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.RequestPartJSON[metadata](ctx, "metadata")
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, jsonPartRequest(t, `{"name":"avatar"}`,
		arkjson.ContentType))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"avatar"`) {
		t.Fatalf("body = %s, want JSON part payload", recorder.Body.String())
	}
}

func TestRequestEntityWithMediaTypesRejectsUnsupportedContentType(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/jobs", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
		entity, err := mvc.RequestEntityWithMediaTypes[requestEntityCreateRequest](ctx,
			arkjson.ContentType)
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

func TestValidatedRequestPartJSONUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	type metadata struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.ValidatedRequestPartJSON[metadata](ctx, "metadata", []string{"create"})
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, jsonPartRequest(t, `{"name":"avatar"}`,
		"application/vnd.goark.metadata+json"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
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

func jsonPartRequest(t testing.TB, body string, contentType string) *http.Request {
	t.Helper()
	var requestBody strings.Builder
	writer := multipart.NewWriter(&requestBody)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "metadata",
		"filename": "metadata.json",
	}))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("CreatePart failed: %v", err)
	}
	if _, err := part.Write([]byte(body)); err != nil {
		t.Fatalf("write JSON part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/parts", strings.NewReader(requestBody.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
func TestBindBodyReadsTextRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/body", mvc.BindBody(http.StatusCreated, func(
		_ *arkweb.Context, input string) (map[string]string, error) {
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

func TestRequestPartJSONRejectsNonJSONPart(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]string, error) {
		return mvc.RequestPartJSON[map[string]string](ctx, "metadata")
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, jsonPartRequest(t, "plain", "text/plain"))
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415, body=%s", recorder.Code, recorder.Body.String())
	}
}

func assertBoolValue(t *testing.T, body map[string]any, name string, want bool) {
	t.Helper()
	got, ok := body[name].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %t", name, body[name], want)
	}
}

func (*panicRequestBodyAdvice) BeforeRead(*arkweb.Context, message.ReadAdviceContext) error {
	panic("nil request body advice must not run")
}

type requestEntityCreateRequest struct {
	Name string `json:"name" arkarta:"required"`
}
