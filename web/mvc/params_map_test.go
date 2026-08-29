package mvc_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc"
)

func TestParameterHelpersBindRequestParamMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Params map[string]string   `json:"params"`
		Values map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodPost, "/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
		params, err := mvc.RequestParamMap(ctx)
		if err != nil {
			return payload{}, err
		}
		values, err := mvc.RequestParamValuesMap(ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Params: params, Values: values}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/search?tag=query&empty=", strings.NewReader("tag=form&q=goark"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Params, map[string]string{"tag": "query", "empty": "", "q": "goark"}) {
		t.Fatalf("params = %#v", got.Params)
	}
	if !reflect.DeepEqual(got.Values, map[string][]string{"tag": {"query", "form"}, "empty": {""}, "q": {"goark"}}) {
		t.Fatalf("values = %#v", got.Values)
	}
}

func TestParameterHelpersBindRequestHeaderMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Headers map[string]string   `json:"headers"`
		Values  map[string][]string `json:"values"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/headers", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
		headers, err := mvc.RequestHeaderMap(ctx)
		if err != nil {
			return payload{}, err
		}
		values, err := mvc.RequestHeaderValuesMap(ctx)
		if err != nil {
			return payload{}, err
		}
		return payload{Headers: headers, Values: values}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/headers", nil)
	request.Header.Add("X-Role", "admin")
	request.Header.Add("X-Role", "ops")
	request.Header.Set("X-Request-ID", "req-1")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if got.Headers["X-Role"] != "admin" || got.Headers["X-Request-Id"] != "req-1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if !reflect.DeepEqual(got.Values["X-Role"], []string{"admin", "ops"}) {
		t.Fatalf("header values = %#v", got.Values)
	}
}

func TestParameterHelpersBindMatrixVariableMaps(t *testing.T) {
	t.Parallel()

	type payload struct {
		Matrix      map[string]string   `json:"matrix"`
		Values      map[string][]string `json:"values"`
		OwnerMatrix map[string]string   `json:"ownerMatrix"`
		OwnerValues map[string][]string `json:"ownerValues"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/cars/{id}/owners/{ownerId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (payload, error) {
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
		return payload{Matrix: matrix, Values: values, OwnerMatrix: ownerMatrix, OwnerValues: ownerValues}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/cars/42;color=red;color=blue;year=2026/owners/7;color=black;color=white;q=owner", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var got payload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Matrix, map[string]string{"color": "red", "year": "2026", "q": "owner"}) {
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
