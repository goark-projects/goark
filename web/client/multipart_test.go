package client_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webclient "goark.dev/goark/web/client"
)

func TestClientPostMultipartBody(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			failServer(serverErrors, writer, "method = %q, want POST", request.Method)
			return
		}
		if contentType := request.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
			failServer(serverErrors, writer, "content type = %q", contentType)
			return
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			failServer(serverErrors, writer, "parse multipart failed: %v", err)
			return
		}
		if got := request.FormValue("title"); got != "avatar" {
			failServer(serverErrors, writer, "title = %q, want avatar", got)
			return
		}
		file, header, err := request.FormFile("file")
		if err != nil {
			failServer(serverErrors, writer, "form file failed: %v", err)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			failServer(serverErrors, writer, "read file failed: %v", err)
			return
		}
		if header.Filename != "profile.txt" || string(data) != "hello" || header.Header.Get("Content-Type") != "text/plain" {
			failServer(serverErrors, writer, "file = %q %q %q", header.Filename, string(data), header.Header.Get("Content-Type"))
			return
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, "uploaded")
	}))
	defer server.Close()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	response, err := client.Post(t.Context(), server.URL,
		webclient.WithMultipartFields(map[string]string{"title": "avatar"}, webclient.MultipartFile{
			FieldName:   "file",
			FileName:    "profile.txt",
			ContentType: "text/plain",
			Body:        strings.NewReader("hello"),
		}),
	)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	assertNoServerError(t, serverErrors)
	if response.StatusCode() != http.StatusCreated || response.BodyString() != "uploaded" {
		t.Fatalf("response = %d %q, want 201 uploaded", response.StatusCode(), response.BodyString())
	}
}

func TestClientRejectsInvalidMultipartBody(t *testing.T) {
	t.Parallel()

	client, err := webclient.New()
	if err != nil {
		t.Fatalf("client new failed: %v", err)
	}
	_, err = client.NewRequest(t.Context(), http.MethodPost, "http://example.com", webclient.WithMultipartFields(nil, webclient.MultipartFile{}))
	if !errors.Is(err, webclient.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}
