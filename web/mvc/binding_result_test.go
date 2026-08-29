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

type bindingCreateRequest struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

type bindingResultPayload struct {
	Valid   bool   `json:"valid"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

func TestBindJSONResultPassesValidationErrorsToHandler(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.BindJSONResult(http.StatusOK,
		func(_ *arkweb.Context, input bindingCreateRequest, result mvc.BindingResult) (bindingResultPayload, error) {
			field, ok := result.FieldError("code")
			if !result.HasErrors() || !ok {
				return bindingResultPayload{Valid: true}, nil
			}
			return bindingResultPayload{
				Valid:   result.Valid(),
				Field:   field.Path(),
				Message: field.Message(),
				Name:    input.Name,
			}, nil
		},
	)); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "code" || payload.Message == "" || payload.Name != "" {
		t.Fatalf("payload = %#v, want binding result violation", payload)
	}
}

func TestBindJSONResultGroupsUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.BindJSONResultGroups(http.StatusOK,
		func(_ *arkweb.Context, _ bindingCreateRequest, result mvc.BindingResult) (bindingResultPayload, error) {
			field, ok := result.FieldError("name")
			if !result.HasErrors() || !ok {
				return bindingResultPayload{Valid: true}, nil
			}
			return bindingResultPayload{Valid: result.Valid(), Field: field.Path(), Message: field.Message()}, nil
		},
		"create",
	)); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "name" || payload.Message == "" {
		t.Fatalf("payload = %#v, want grouped binding result violation", payload)
	}
}

func TestValidatedRequestBodyResultUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (bindingResultPayload, error) {
		input, result, err := mvc.ValidatedRequestBodyResult[bindingCreateRequest](ctx, "create")
		if err != nil {
			return bindingResultPayload{}, err
		}
		if !result.HasErrors() {
			return bindingResultPayload{Valid: true, Name: input.Name}, nil
		}
		field, ok := result.FieldError("name")
		if !ok {
			return bindingResultPayload{Valid: result.Valid(), Name: input.Name}, nil
		}
		return bindingResultPayload{
			Valid:   result.Valid(),
			Field:   field.Path(),
			Message: field.Message(),
			Name:    input.Name,
		}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", arkjson.ContentType)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "name" || payload.Message == "" || payload.Name != "" {
		t.Fatalf("payload = %#v, want grouped binding result violation", payload)
	}
}
