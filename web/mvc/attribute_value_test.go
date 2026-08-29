package mvc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/mvc"
)

type attributeProfile struct {
	ID string `json:"id"`
}

type attributePayload struct {
	TraceID   string `json:"traceId"`
	ProfileID string `json:"profileId"`
	Limit     int    `json:"limit"`
}

func TestAttributeHelpersReadTypedValues(t *testing.T) {
	t.Parallel()

	router := arkweb.NewRouter()
	if err := router.Handle(http.MethodGet, "/attributes", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (attributePayload, error) {
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
	handler := servlet.ChainFilters(router, servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
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
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/attributes", nil))
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
