package message_test

import (
	"net/http"
	"net/url"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

func TestReaderReadsURLEncodedFormBody(t *testing.T) {
	t.Parallel()

	var got url.Values
	recorder := serveReadMessage(t, message.MediaTypeFormURLEncoded, "name=goark&tag=web&tag=mvc", func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", recorder.Code, recorder.Body.String())
	}
	if got.Get("name") != "goark" || len(got["tag"]) != 2 || got["tag"][1] != "mvc" {
		t.Fatalf("form = %#v, want decoded url values", got)
	}
}

func TestReaderRejectsMalformedURLEncodedFormBody(t *testing.T) {
	t.Parallel()

	var got url.Values
	recorder := serveReadMessage(t, message.MediaTypeFormURLEncoded, "%zz", func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader().Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestWriterWritesURLEncodedFormBody(t *testing.T) {
	t.Parallel()

	values := url.Values{}
	values.Set("name", "goark")
	values.Add("tag", "web")
	values.Add("tag", "mvc")
	recorder := serveMessage(t, message.MediaTypeFormURLEncoded, func(ctx *arkweb.Context) error {
		return message.NewWriter().Write(ctx, http.StatusCreated, values)
	})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != message.MediaTypeFormURLEncoded {
		t.Fatalf("Content-Type = %q, want form", got)
	}
	if recorder.Body.String() != "name=goark&tag=web&tag=mvc" {
		t.Fatalf("body = %q, want encoded form", recorder.Body.String())
	}
}
