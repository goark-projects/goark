package mvc_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type uploadRequest struct {
	Title string                `form:"title"`
	File  servletmultipart.Part `multipart:"file"`
}

type uploadPartsPayload struct {
	Names  []string `json:"names"`
	Bodies []string `json:"bodies"`
	Count  int      `json:"count"`
}

func TestBindMultipartBindsValuesAndParts(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipart(http.StatusCreated, func(_ *arkweb.Context, input uploadRequest) (map[string]any, error) {
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
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var payload map[string]any
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["title"] != "avatar" || payload["filename"] != "profile.txt" || payload["body"] != "hello" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestBindMultipartEntityWritesResponseEntity(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipartEntity(func(_ *arkweb.Context, input uploadRequest) (web.ResponseEntity[map[string]string], error) {
			return web.Status(http.StatusAccepted, map[string]string{"title": input.Title}).
				WithHeader("X-Upload", input.File.SubmittedFileName()), nil
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
	request.Header.Set("Accept", arkjson.ContentType)
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := recorder.Header().Get("X-Upload"); got != "profile.txt" {
		t.Fatalf("X-Upload = %q, want profile.txt", got)
	}
}

func TestRequestPartReturnsNamedMultipartPart(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/part", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
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

func TestRequestPartsByNameReturnsNamedMultipartParts(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/parts", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (uploadPartsPayload, error) {
			parts, err := mvc.RequestPartsByName(ctx, "file")
			if err != nil {
				return uploadPartsPayload{}, err
			}
			optional, err := mvc.RequestPartsByName(ctx, "missing", mvc.WithRequired(false))
			if err != nil {
				return uploadPartsPayload{}, err
			}
			payload := uploadPartsPayload{
				Names:  make([]string, 0, len(parts)),
				Bodies: make([]string, 0, len(parts)),
				Count:  len(optional),
			}
			for _, part := range parts {
				file, err := part.Open()
				if err != nil {
					return uploadPartsPayload{}, err
				}
				data, readErr := io.ReadAll(file)
				closeErr := file.Close()
				if readErr != nil {
					return uploadPartsPayload{}, readErr
				}
				if closeErr != nil {
					return uploadPartsPayload{}, closeErr
				}
				payload.Names = append(payload.Names, part.SubmittedFileName())
				payload.Bodies = append(payload.Bodies, string(data))
			}
			return payload, nil
		})),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	request := multipartPartsRequest(t)
	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload uploadPartsPayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if len(payload.Names) != 2 ||
		payload.Names[0] != "first.txt" ||
		payload.Names[1] != "second.txt" ||
		len(payload.Bodies) != 2 ||
		payload.Bodies[0] != "first" ||
		payload.Bodies[1] != "second" ||
		payload.Count != 0 {
		t.Fatalf("payload = %#v, want named multipart parts", payload)
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
