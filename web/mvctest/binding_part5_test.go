package mvc_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestRequestPartsByNameReturnsNamedMultipartParts(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads/parts", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (
			uploadPartsPayload, error) {
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

func TestAttributeHelpersReadTypedValues(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/attributes", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (attributePayload, error) {
		profile, err := mvc.RequestAttribute[attributeProfile](ctx, "profile")
		if err != nil {
			return attributePayload{}, err
		}
		limit, err := mvc.RequestAttribute[int](ctx, "limit")
		if err != nil {
			return attributePayload{}, err
		}
		traceID, err := mvc.SessionAttribute[string](ctx, "traceID")
		if err != nil {
			return attributePayload{}, err
		}
		return attributePayload{TraceID: traceID, ProfileID: profile.ID, Limit: limit}, nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	handler := servlet.ChainFilters(router, servlet.FilterFunc(func(ctx context.Context,
		req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		req.SetAttribute("profile", attributeProfile{ID: "p-1"})
		req.SetAttribute("limit", "42")
		current, err := session.NewMemoryManager().Create(ctx)
		if err != nil {
			return err
		}
		if err := current.SetAttribute("traceID", "trace-1"); err != nil {
			return err
		}
		req.SetAttribute(session.AttributeCurrentSession, current)
		return chain.Next(ctx, req, res)
	}))

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/attributes", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload attributePayload
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if payload.TraceID != "trace-1" || payload.ProfileID != "p-1" || payload.Limit != 42 {
		t.Fatalf("payload = %#v, want typed attributes", payload)
	}
}

func TestModelAttributeBindsQueryAndFormValues(t *testing.T) {
	t.Parallel()

	type userSearchCriteria struct {
		Username        string `form:"username" json:"username" arkarta:"required,min=2"`
		Page            int    `form:"page" json:"page"`
		IncludeDisabled bool   `form:"includeDisabled" json:"includeDisabled"`
	}
	router := arkweb.NewRouter()
	err := router.Handle(http.MethodPost, "/users/search", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (map[string]any, error) {
		criteria, err := mvc.ModelAttribute[userSearchCriteria](ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"username":        criteria.Username,
			"page":            criteria.Page,
			"includeDisabled": criteria.IncludeDisabled,
		}, nil
	}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users/search?username=ad",
		strings.NewReader("page=2&includeDisabled=true"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	servletnethttp.Handler(router).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Username        string `json:"username"`
		Page            int    `json:"page"`
		IncludeDisabled bool   `json:"includeDisabled"`
	}
	if err := arkjson.Unmarshal(nil, recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Username != "ad" || payload.Page != 2 || !payload.IncludeDisabled {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestValidatedRequestBodyResultUsesExplicitValidationGroup(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodPost, "/users", mvc.JSON(http.StatusOK, func(
		ctx *arkweb.Context) (bindingResultPayload, error) {
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

func TestMatrixVariableSlicesBindRepeatedValues(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	err := router.Handle(http.MethodGet, "/cars/{id}/owners/{ownerId}", mvc.JSON(http.StatusOK,
		func(ctx *arkweb.Context) (map[string]any, error) {
			colors, err := mvc.MatrixVariableStrings(ctx, "color")
			if err != nil {
				return nil, err
			}
			codes, err := mvc.MatrixVariableInts(ctx, "code")
			if err != nil {
				return nil, err
			}
			ownerColors, err := mvc.MatrixVariableStrings(ctx, "color", mvc.WithMatrixPathVariable(
				"ownerId"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"colors": colors, "codes": codes, "ownerColors": ownerColors}, nil
		}))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(router).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/cars/42;color=red;color=blue;code=1,2;code=3/owners/7;color=black;color=white", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"colors":["red","blue","black","white"]`) ||
		!strings.Contains(recorder.Body.String(), `"codes":[1,2,3]`) ||
		!strings.Contains(recorder.Body.String(), `"ownerColors":["black","white"]`) {
		t.Fatalf("body = %s, want repeated matrix values", recorder.Body.String())
	}
}

func TestBindMultipartEntityWritesResponseEntity(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	configurer := mvc.NewConfigurer(mvc.NewController("uploads",
		mvc.POST("/uploads", mvc.BindMultipartEntity(func(_ *arkweb.Context, input uploadRequest) (
			web.ResponseEntity[map[string]string], error) {
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

func binderPreferencePayloadFromInput(input binderPreferenceInput) binderPreferencePayload {
	out := binderPreferencePayload{
		Theme:      input.Theme,
		TagsNil:    input.Tags == nil,
		TagsLength: len(input.Tags),
	}
	if input.Notify != nil {
		out.NotifySet = true
		out.Notify = *input.Notify
	}
	if input.Confirm != nil {
		out.ConfirmSet = true
		out.Confirm = *input.Confirm
	}
	if input.Profile != nil {
		out.ProfileSet = true
		out.Subscribed = input.Profile.Subscribed
	}
	return out
}

type bindingSearchRequest struct {
	Name string `form:"name" json:"name" arkarta:"required"`
	Page int    `form:"page" json:"page"`
}

type attributeProfile struct {
	ID string `json:"id"`
}

type binderSuppressedProfile struct {
	Email string `form:"email" json:"email"`
}
