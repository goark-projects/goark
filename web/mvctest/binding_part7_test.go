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
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/web"
	"goark.dev/goark/web/filter"
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindMatrixVariableMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Matrix      map[string]string   `json:"matrix"`
		Values      map[string][]string `json:"values"`
		OwnerMatrix map[string]string   `json:"ownerMatrix"`
		OwnerValues map[string][]string `json:"ownerValues"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/cars/{id}/owners/{ownerId}", mvc.JSON(http.StatusOK,
		func(ctx *arkweb.Context) (payload, error) {
			matrix, err := mvc.MatrixVariableMap(ctx)
			if err != nil {
				return payload{}, err
			}
			values, err := mvc.MatrixVariableValuesMap(ctx)
			if err != nil {
				return payload{}, err
			}
			ownerMatrix, err := mvc.MatrixVariableMap(ctx, mvc.WithMatrixPathVariable("ownerId"))
			if err != nil {
				return payload{}, err
			}
			ownerValues, err := mvc.MatrixVariableValuesMap(ctx, mvc.WithMatrixPathVariable("ownerId"))
			if err != nil {
				return payload{}, err
			}
			return payload{Matrix: matrix, Values: values, OwnerMatrix: ownerMatrix,
				OwnerValues: ownerValues}, nil
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/cars/42;color=red;color=blue;year=2026/owners/7;color=black;color=white;q=owner", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Matrix, map[string]string{"color": "red", "year": "2026",
		"q": "owner"}) {
		t.Fatalf("matrix = %#v", got.Matrix)
	}
	if !reflect.DeepEqual(got.Values["color"], []string{"red", "blue", "black", "white"}) ||
		!reflect.DeepEqual(got.Values["year"], []string{"2026"}) ||
		!reflect.DeepEqual(got.Values["q"], []string{"owner"}) {
		t.Fatalf("values = %#v", got.Values)
	}
	if !reflect.DeepEqual(got.OwnerMatrix, map[string]string{"color": "black", "q": "owner"}) {
		t.Fatalf("owner matrix = %#v", got.OwnerMatrix)
	}
	if !reflect.DeepEqual(got.OwnerValues["color"], []string{"black", "white"}) ||
		!reflect.DeepEqual(got.OwnerValues["q"], []string{"owner"}) {
		t.Fatalf("owner values = %#v", got.OwnerValues)
	}
}

func TestModelAttributeBindsIndexedProperties(t *testing.T) {
	t.Parallel()

	type owner struct {
		Name    string   `form:"name" json:"name"`
		Age     int      `form:"age" json:"age"`
		Aliases []string `form:"aliases" json:"aliases"`
	}
	type userSearchCriteria struct {
		Owners []owner `form:"owners" json:"owners"`
		Page   int     `form:"page" json:"page"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"firstName":   criteria.Owners[0].Name,
			"firstAge":    criteria.Owners[0].Age,
			"firstAlias":  criteria.Owners[0].Aliases[0],
			"secondName":  criteria.Owners[1].Name,
			"secondAge":   criteria.Owners[1].Age,
			"secondAlias": criteria.Owners[1].Aliases[0],
			"page":        criteria.Page,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?"+
		"owners[0].name=ada&owners[0].age=37&owners[0].aliases[0]=lead&"+
		"owners[1].name=linus&owners[1].age=55&owners[1].aliases[0]=kernel&page=2", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"firstName":"ada"`) ||
		!strings.Contains(recorder.Body.String(), `"firstAge":37`) ||
		!strings.Contains(recorder.Body.String(), `"firstAlias":"lead"`) ||
		!strings.Contains(recorder.Body.String(), `"secondName":"linus"`) ||
		!strings.Contains(recorder.Body.String(), `"secondAge":55`) ||
		!strings.Contains(recorder.Body.String(), `"secondAlias":"kernel"`) ||
		!strings.Contains(recorder.Body.String(), `"page":2`) {
		t.Fatalf("body = %s, want indexed model attribute values", recorder.Body.String())
	}
}

func TestRequestPartReturnsNamedMultipartPart(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/part", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			map[string]string, error) {
			part, err := mvc.RequestPart(ctx, "file")
			if err != nil {
				return nil, err
			}
			file, err := part.Open()
			if err != nil {
				return nil, err
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}
			return map[string]string{
				"filename": part.SubmittedFileName(),
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
	request.URL.Path = "/uploads/part"
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"filename":"profile.txt"`) ||
		!strings.Contains(recorder.Body.String(), `"body":"hello"`) {
		t.Fatalf("body = %s, want part payload", recorder.Body.String())
	}
}

func TestMultipartResultPassesValidationErrorsToHandler(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/helper", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			uploadBindingPayload, error) {
			input, result, err := mvc.MultipartResult[uploadValidatedRequest](ctx)
			if err != nil {
				return uploadBindingPayload{}, err
			}
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
	servletnethttp.Handler(router).ServeHTTP(recorder, multipartFileOnlyRequest(t, "/uploads/helper"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload uploadBindingPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "title" || payload.Filename != "profile.txt" {
		t.Fatalf("payload = %#v, want helper multipart binding result", payload)
	}
}

func TestModelAttributeReadsFormContentFilterValues(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username string `form:"username" json:"username" arkarta:"required,min=2"`
		Page     int    `form:"page" json:"page"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodDelete, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"username": criteria.Username,
			"page":     criteria.Page,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/users/search?username=ad",
		strings.NewReader("page=2"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(servlet.ChainFilters(router, filter.FormContent())).ServeHTTP(recorder,
		request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"username":"ad"`) ||
		!strings.Contains(recorder.Body.String(), `"page":2`) {
		t.Fatalf("body = %s, want model attribute values", recorder.Body.String())
	}
}

func TestModelAttributeRejectsOversizedIndexedProperties(t *testing.T) {
	t.Parallel()

	type owner struct {
		Name string `form:"name" json:"name"`
	}
	type userSearchCriteria struct {
		Owners []owner `form:"owners" json:"owners"`
	}

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]int, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]int{"owners": len(criteria.Owners)}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/search?owners[256].name=ada", nil)
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func multipartFileOnlyRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
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
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

type uploadPartsPayload struct {
	Names  []string `json:"names"`
	Bodies []string `json:"bodies"`
	Count  int      `json:"count"`
}

type uploadRequest struct {
	Title string                `form:"title"`
	File  servletmultipart.Part `multipart:"file"`
}
