package mvc_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindExtendedConversions(t *testing.T) {
	t.Parallel()

	type payload struct {
		Score     float64   `json:"score"`
		PathIDs   []int64   `json:"pathIds"`
		Tags      []string  `json:"tags"`
		IDs       []int64   `json:"ids"`
		Enabled   []bool    `json:"enabled"`
		Ratios    []float64 `json:"ratios"`
		HeaderAt  string    `json:"headerAt"`
		Roles     []string  `json:"roles"`
		Threshold float64   `json:"threshold"`
		Day       string    `json:"day"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/reports/{date}/{pathIds}", mvc.JSON(http.StatusOK,
		func(ctx *arkweb.Context) (payload, error) {
			score, err := mvc.RequestParamFloat64(ctx, "score")
			if err != nil {
				return payload{}, err
			}
			pathIDs, err := mvc.PathInt64s(ctx, "pathIds")
			if err != nil {
				return payload{}, err
			}
			tags, err := mvc.RequestParamStrings(ctx, "tag")
			if err != nil {
				return payload{}, err
			}
			ids, err := mvc.RequestParamInt64s(ctx, "ids")
			if err != nil {
				return payload{}, err
			}
			enabled, err := mvc.RequestParamBools(ctx, "enabled")
			if err != nil {
				return payload{}, err
			}
			ratios, err := mvc.RequestParamFloat64s(ctx, "ratio")
			if err != nil {
				return payload{}, err
			}
			headerAt, err := mvc.RequestHeaderTime(ctx, "X-At")
			if err != nil {
				return payload{}, err
			}
			roles, err := mvc.RequestHeaderStrings(ctx, "X-Role")
			if err != nil {
				return payload{}, err
			}
			threshold, err := mvc.CookieValueFloat64(ctx, "threshold")
			if err != nil {
				return payload{}, err
			}
			day, err := mvc.PathTime(ctx, "date", mvc.WithTimeLayout("20060102"))
			if err != nil {
				return payload{}, err
			}
			return payload{
				Score:     score,
				PathIDs:   pathIDs,
				Tags:      tags,
				IDs:       ids,
				Enabled:   enabled,
				Ratios:    ratios,
				HeaderAt:  headerAt.UTC().Format(time.RFC3339),
				Roles:     roles,
				Threshold: threshold,
				Day:       day.Format("2006-01-02"),
			}, nil
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet,
		"/reports/20260829/7,8?score=98.5&tag=ops&tag=web,api&ids=1,2&enabled=true,false&ratio=1.5&"+
			"ratio=2.5", nil)
	request.Header.Set("X-At", "2026-08-29T02:30:00Z")
	request.Header.Add("X-Role", "admin,ops")
	request.AddCookie(&http.Cookie{Name: "threshold", Value: "0.75"})
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Score != 98.5 ||
		!reflect.DeepEqual(got.PathIDs, []int64{7, 8}) ||
		!reflect.DeepEqual(got.Tags, []string{"ops", "web", "api"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) ||
		!reflect.DeepEqual(got.Enabled, []bool{true, false}) ||
		!reflect.DeepEqual(got.Ratios, []float64{1.5, 2.5}) ||
		got.HeaderAt != "2026-08-29T02:30:00Z" ||
		!reflect.DeepEqual(got.Roles, []string{"admin", "ops"}) ||
		got.Threshold != 0.75 ||
		got.Day != "2026-08-29" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestBindMultipartBindsValuesAndParts(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipart(http.StatusCreated, func(_ *arkweb.Context,
			input uploadRequest) (map[string]any, error) {
			file, err := input.File.Open()
			if err != nil {
				return nil, err
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"title":    input.Title,
				"filename": input.File.SubmittedFileName(),
				"body":     string(data),
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

	request := multipartRequest(t)
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusCreated,
			recorder.Body.String())
	}
	var payload map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["title"] != "avatar" || payload["filename"] != "profile.txt" || payload["body"] !=
		"hello" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestParameterHelpersBindCookieValueMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Cookies map[string]string   `json:"cookies"`
		Values  map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/cookies", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (payload, error) {
		cookies, err := mvc.CookieValueMap(ctx)
		if err != nil {
			return payload{}, err
		}
		values, err := mvc.CookieValueValuesMap(ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Cookies: cookies, Values: values}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/cookies", nil)
	request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	request.AddCookie(&http.Cookie{Name: "role", Value: "admin"})
	request.AddCookie(&http.Cookie{Name: "role", Value: "ops"})
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Cookies, map[string]string{"theme": "dark", "role": "admin"}) {
		t.Fatalf("cookies = %#v", got.Cookies)
	}
	if !reflect.DeepEqual(got.Values["role"], []string{"admin", "ops"}) ||
		!reflect.DeepEqual(got.Values["theme"], []string{"dark"}) {
		t.Fatalf("values = %#v", got.Values)
	}
}

func TestModelAttributeResultPassesValidationErrorsToHandler(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (bindingResultPayload, error) {
		input, result, err := mvc.ModelAttributeResult[bindingSearchRequest](ctx)
		if err != nil {
			return bindingResultPayload{}, err
		}
		field, ok := result.FieldError("name")
		if !result.HasErrors() || !ok {
			return bindingResultPayload{Valid: result.Valid(), Name: input.Name, Page: input.Page}, nil
		}
		return bindingResultPayload{
			Valid:   result.Valid(),
			Field:   field.Path(),
			Message: field.Message(),
			Name:    input.Name,
			Page:    input.Page,
		}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/users/search?page=2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "name" || payload.Message == "" || payload.Page != 2 {
		t.Fatalf("payload = %#v, want model attribute validation result", payload)
	}
}

func TestModelAttributeResultCapturesBindingError(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (bindingResultPayload, error) {
		input, result, err := mvc.ModelAttributeResult[bindingSearchRequest](ctx)
		if err != nil {
			return bindingResultPayload{}, err
		}
		return bindingResultPayload{
			Valid:        result.Valid(),
			BindingError: result.BindingError() != nil,
			Name:         input.Name,
			Page:         input.Page,
		}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/users/search?name=goark&page=bad", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload bindingResultPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || !payload.BindingError || payload.Name != "goark" || payload.Page != 0 {
		t.Fatalf("payload = %#v, want captured model attribute binding error", payload)
	}
}

func multipartRequest(t *testing.T) *http.Request {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "avatar"); err != nil {
		t.Fatalf("WriteField failed: %v", err)
	}
	part, err := writer.CreateFormFile("file", "profile.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := io.WriteString(part, "hello"); err != nil {
		t.Fatalf("write part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

type binderSuppressedInput struct {
	Name    string                   `form:"name" json:"name"`
	Admin   bool                     `form:"admin" json:"admin"`
	Profile *binderSuppressedProfile `form:"profile" json:"profile"`
}

type binderTagInput struct {
	Tags    []string          `form:"tags" json:"tags"`
	Profile *binderTagProfile `form:"profile" json:"profile"`
}

type binderPreferenceProfile struct {
	Subscribed bool `form:"subscribed" json:"subscribed"`
}
