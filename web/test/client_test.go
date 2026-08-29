package webtest_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/static"
	webtest "goark.dev/goark/web/test"
)

type createJobRequest struct {
	Name string `json:"name"`
}

type uploadPayload struct {
	Title string                `form:"title"`
	File  servletmultipart.Part `multipart:"file"`
}

func TestRouterClientPerformsJSONRequest(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.POST("/jobs/{id}", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input createJobRequest
		if err := ctx.BindJSON(&input); err != nil {
			return nil, err
		}
		return arkweb.JSON(http.StatusCreated, map[string]string{
			"id":   ctx.PathValue("id"),
			"name": input.Name,
		}), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	client, err := webtest.NewRouter(router)
	client = webtest.Must(t, client, err)
	response := client.Perform(t, http.MethodPost, "/jobs/42",
		webtest.WithAccept(arkjson.ContentType),
		webtest.WithJSONBody(createJobRequest{Name: "sync"}),
	)

	response.
		ExpectStatus(t, http.StatusCreated).
		ExpectHeader(t, "Content-Type", arkjson.ContentType).
		ExpectJSON(t, map[string]string{"id": "42", "name": "sync"}).
		ExpectJSONPath(t, "$.id", "42").
		ExpectJSONPath(t, "$.name", "sync").
		ExpectJSONPathAbsent(t, "$.missing")
	decoded := webtest.DecodeJSON[map[string]string](t, response)
	if decoded["name"] != "sync" {
		t.Fatalf("decoded name = %q, want sync", decoded["name"])
	}
}

func TestResponseJSONPathAssertionsSupportArraysAndQuotedKeys(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/matrix", arkweb.HandlerFunc(func(_ *arkweb.Context) (arkweb.Result, error) {
		return arkweb.JSON(http.StatusOK, map[string]any{
			"items": []map[string]any{
				{"name": "admin", "order": 1},
			},
			"trace.id": "trace-1",
		}), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	client, err := webtest.NewRouter(router)
	client = webtest.Must(t, client, err)
	client.Perform(t, http.MethodGet, "/matrix").
		ExpectStatus(t, http.StatusOK).
		ExpectJSONPath(t, "$.items[0].name", "admin").
		ExpectJSONPath(t, "$.items[0].order", 1).
		ExpectJSONPath(t, "$['trace.id']", "trace-1").
		ExpectJSONPathExists(t, "$.items[0]").
		ExpectJSONPathAbsent(t, "$.items[1]")
}

func TestResponseStatusClassAndHeaderAssertions(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.GET("/metadata", arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		ctx.Response().Header().Add("Vary", "Accept")
		ctx.Response().Header().Add("Vary", "Origin")
		ctx.Response().Header().Set("X-Trace", "trace-1")
		return goweb.NoBody(http.StatusAccepted), nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	client, err := webtest.NewRouter(router)
	client = webtest.Must(t, client, err)
	client.Perform(t, http.MethodGet, "/metadata").
		ExpectStatus2xx(t).
		ExpectHeaderExists(t, "X-Trace").
		ExpectHeaderValues(t, "Vary", "Accept", "Origin").
		ExpectHeaderAbsent(t, "X-Missing")
}

func TestRegistryClientRunsFiltersAndStaticServlet(t *testing.T) {
	t.Parallel()

	configurer, err := static.New("/assets/*", fstest.MapFS{
		"app.txt": &fstest.MapFile{
			Data:    []byte("hello static"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	})
	if err != nil {
		t.Fatalf("static.New failed: %v", err)
	}
	registry := goweb.NewRegistry()
	registry.AddFilter(servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		res.Header().Set("X-WebTest-Filter", "hit")
		return chain.Next(ctx, req, res)
	}))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}

	client, err := webtest.NewRegistry(t.Context(), registry, goweb.DeploymentSpec{ContextPath: "/admin"})
	client = webtest.Must(t, client, err)
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Errorf("client close failed: %v", err)
		}
	})

	client.Perform(t, http.MethodGet, "/admin/assets/app.txt").
		ExpectStatus(t, http.StatusOK).
		ExpectHeader(t, "X-WebTest-Filter", "hit").
		ExpectBody(t, "hello static")
}

func TestResponseCookieAssertions(t *testing.T) {
	t.Parallel()

	registry := goweb.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("cookies",
		mvc.POST("/sessions", mvc.Handler(func(_ *arkweb.Context) (arkweb.Result, error) {
			return goweb.NoBody(http.StatusCreated).WithCookie(&http.Cookie{
				Name:     "sid",
				Value:    "abc",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			}), nil
		})),
	))
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	router, err := registry.Router()
	if err != nil {
		t.Fatalf("Router failed: %v", err)
	}

	client, err := webtest.NewRouter(router)
	client = webtest.Must(t, client, err)
	client.Perform(t, http.MethodPost, "/sessions").
		ExpectStatus(t, http.StatusCreated).
		ExpectCookie(t, "sid", "abc").
		ExpectCookieHTTPOnly(t, "sid", true).
		ExpectCookieSecure(t, "sid", true).
		ExpectNoCookie(t, "missing")
}

func TestMultipartRequestOptionBuildsUpload(t *testing.T) {
	t.Parallel()

	registry := goweb.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipart(http.StatusCreated, func(_ *arkweb.Context, input uploadPayload) (map[string]string, error) {
			file, err := input.File.Open()
			if err != nil {
				return nil, err
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}
			return map[string]string{
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

	client, err := webtest.NewRouter(router)
	client = webtest.Must(t, client, err)
	client.Perform(t, http.MethodPost, "/uploads",
		webtest.WithAccept(arkjson.ContentType),
		webtest.WithMultipartBody(map[string]string{"title": "avatar"}, webtest.MultipartFile{
			FieldName: "file",
			FileName:  "profile.txt",
			Body:      strings.NewReader("hello"),
		}),
	).ExpectStatus(t, http.StatusCreated).
		ExpectJSON(t, map[string]string{
			"title":    "avatar",
			"filename": "profile.txt",
			"body":     "hello",
		})
}
