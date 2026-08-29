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
	"goark.dev/goark/web/mvc"
)

func TestRequestPartJSONBindsTypedPart(t *testing.T) {
	t.Parallel()

	type metadata struct {
		Name string `json:"name" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.RequestPartJSON[metadata](ctx, "metadata")
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, jsonPartRequest(t, `{"name":"avatar"}`, arkjson.ContentType))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"name":"avatar"`) {
		t.Fatalf("body = %s, want JSON part payload", recorder.Body.String())
	}
}

func TestValidatedRequestPartJSONUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	type metadata struct {
		Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
		Code string `json:"code" arkarta:"required"`
	}
	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
		input, err := mvc.ValidatedRequestPartJSON[metadata](ctx, "metadata", []string{"create"})
		if err != nil {
			return nil, err
		}
		return map[string]string{"name": input.Name}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, jsonPartRequest(t, `{"name":"avatar"}`, "application/vnd.goark.metadata+json"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRequestPartJSONRejectsNonJSONPart(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/parts", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
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
