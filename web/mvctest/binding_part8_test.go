package mvc_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindRequestSources(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/{id}", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
		id, err := mvc.PathInt64(ctx, "id")
		if err != nil {
			return nil, err
		}
		page, err := mvc.RequestParamInt(ctx, "page", mvc.WithDefaultValue("1"))
		if err != nil {
			return nil, err
		}
		requestID, err := mvc.RequestHeaderString(ctx, "X-Request-ID")
		if err != nil {
			return nil, err
		}
		theme, err := mvc.CookieValueString(ctx, "theme", mvc.WithRequired(false))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"id":        id,
			"page":      page,
			"requestID": requestID,
			"theme":     theme,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	request.Header.Set("X-Request-ID", "req-1")
	request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ID        int64  `json:"id"`
		Page      int    `json:"page"`
		RequestID string `json:"requestID"`
		Theme     string `json:"theme"`
	}
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.ID != 42 || payload.Page != 1 || payload.RequestID != "req-1" || payload.Theme !=
		"dark" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestControllerAdviceInitBinderSkipsDisallowedModelAttributeFields(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("users",
		mvc.GET("/users/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			binderUserPayload, error) {
			input, err := mvc.ModelAttribute[binderUserInput](ctx)
			if err != nil {
				return binderUserPayload{}, err
			}
			return binderPayload(input), nil
		})),
	)
	advice := mvc.NewRestControllerAdvice("global-binder").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.SetDisallowedFields("ADMIN", "PROFILE.ADMIN", "ROLES[*].ADMIN", "METADATA[SECRET]")
		}),
	)
	configurer := mvc.NewConfigurer(controller).WithControllerAdvices(advice)
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?"+
		"name=ada&admin=true&profile.email=ada@example.test&profile.admin=true&"+
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]="+
		"root", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderUserPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Name != "ada" || got.Admin || got.ProfileEmail != "ada@example.test" || got.ProfileAdmin {
		t.Fatalf("payload = %#v, want disallowed scalar and nested values skipped", got)
	}
	if len(got.Roles) != 1 || got.Roles[0].Name != "reader" || got.Roles[0].Admin {
		t.Fatalf("roles = %#v, want disallowed indexed role admin skipped", got.Roles)
	}
	if got.Metadata["department"] != "engineering" {
		t.Fatalf("metadata = %#v, want allowed department", got.Metadata)
	}
	if _, ok := got.Metadata["secret"]; ok {
		t.Fatalf("metadata = %#v, want disallowed secret skipped", got.Metadata)
	}
}

func TestModelAttributeResultReportsSuppressedFields(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	controller := mvc.NewRestController("suppressed",
		mvc.GET("/suppressed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			binderSuppressedPayload, error) {
			input, result, err := mvc.ModelAttributeResult[binderSuppressedInput](ctx)
			if err != nil {
				return binderSuppressedPayload{}, err
			}
			return binderSuppressedPayload{
				Name:             input.Name,
				Admin:            input.Admin,
				SuppressedFields: result.SuppressedFields(),
			}, nil
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
	request := httptest.NewRequest(http.MethodGet,
		"/suppressed?name=ada&admin=true&profile.email=ada@example.test", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got binderSuppressedPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Name != "ada" || got.Admin || len(got.SuppressedFields) != 2 ||
		got.SuppressedFields[0] != "admin" ||
		got.SuppressedFields[1] != "profile.email" {
		t.Fatalf("payload = %#v, want suppressed fields", got)
	}
}

func TestRequestParamReadsEmptyArrayIndexValues(t *testing.T) {
	t.Parallel()

	type payload struct {
		Query string   `json:"query"`
		Tags  []string `json:"tags"`
		IDs   []int64  `json:"ids"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		query, err := mvc.RequestParamString(ctx, "q")
		if err != nil {
			return payload{}, err
		}
		tags, err := mvc.RequestParamStrings(ctx, "tag")
		if err != nil {
			return payload{}, err
		}
		ids, err := mvc.RequestParamInt64s(ctx, "id")
		if err != nil {
			return payload{}, err
		}
		return payload{Query: query, Tags: tags, IDs: ids}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/search?q[]=goark&tag[]=web&tag[]=mvc&id[]=1&id[]=2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Query != "goark" ||
		!reflect.DeepEqual(got.Tags, []string{"web", "mvc"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) {
		t.Fatalf("payload = %#v, want empty array index request params", got)
	}
}

func TestBindMultipartResultPassesValidationErrorsToHandler(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/result", mvc.BindMultipartResult(http.StatusOK, func(_ *arkweb.Context,
			input uploadValidatedRequest, result mvc.BindingResult) (uploadBindingPayload, error) {
			field, ok := result.FieldError("title")
			if !ok {
				return uploadBindingPayload{Valid: result.Valid()}, nil
			}
			return uploadBindingPayload{
				Valid:    result.Valid(),
				Field:    field.Path(),
				Filename: input.File.SubmittedFileName(),
			}, nil
		})),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, multipartFileOnlyRequest(t, "/uploads/result"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload uploadBindingPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "title" || payload.Filename != "profile.txt" {
		t.Fatalf("payload = %#v, want multipart binding result", payload)
	}
}

func multipartPartsRequest(t *testing.T) *http.Request {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	for _, item := range []struct {
		filename string
		body     string
	}{
		{filename: "first.txt", body: "first"},
		{filename: "second.txt", body: "second"},
	} {
		part, err := writer.CreateFormFile("file", item.filename)
		if err != nil {
			t.Fatalf("CreateFormFile failed: %v", err)
		}
		if _, err := io.WriteString(part, item.body); err != nil {
			t.Fatalf("write part failed: %v", err)
		}
	}
	other, err := writer.CreateFormFile("avatar", "ignored.txt")
	if err != nil {
		t.Fatalf("CreateFormFile avatar failed: %v", err)
	}
	if _, err := io.WriteString(other, "ignored"); err != nil {
		t.Fatalf("write avatar failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/uploads/parts", strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func binderPayload(input binderUserInput) binderUserPayload {
	out := binderUserPayload{
		Name:     input.Name,
		Admin:    input.Admin,
		Roles:    input.Roles,
		Metadata: input.Metadata,
	}
	if input.Profile != nil {
		out.ProfileEmail = input.Profile.Email
		out.ProfileAdmin = input.Profile.Admin
	}
	return out
}

type attributePayload struct {
	TraceID   string `json:"traceId"`
	ProfileID string `json:"profileId"`
	Limit     int    `json:"limit"`
}

type binderProfile struct {
	Email string `form:"email" json:"email"`
	Admin bool   `form:"admin" json:"admin"`
}

type scopedTenantID struct {
	value string
}
