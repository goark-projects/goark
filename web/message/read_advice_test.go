package message_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

type advicePayload struct {
	Name string `json:"name"`
}

func TestReaderReadAdviceRunsBeforeAndAfterConverter(t *testing.T) {
	t.Parallel()

	var got advicePayload
	var events []string
	advice := message.ReadAdviceFunc{
		Before: func(ctx *arkweb.Context, input message.ReadAdviceContext) error {
			if input.Target != &got {
				return errors.New("target mismatch")
			}
			if input.Converter == nil {
				return errors.New("converter missing")
			}
			ctx.Request().SetAttribute("advice.before", true)
			events = append(events, "before:"+input.MediaType)
			return nil
		},
		After: func(ctx *arkweb.Context, input message.ReadAdviceContext) error {
			if value, ok := ctx.Request().Attribute("advice.before"); !ok || value != true {
				return errors.New("before marker missing")
			}
			target := input.Target.(*advicePayload)
			target.Name += "-after"
			events = append(events, "after:"+input.MediaType)
			return nil
		},
	}
	recorder := serveReadMessage(t, arkjson.ContentType, `{"name":"goark"}`, func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := message.NewReader(message.WithReadAdvice(advice)).Read(ctx, &got); err != nil {
			return nil, err
		}
		return arkweb.Text(http.StatusOK, got.Name), nil
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "goark-after" {
		t.Fatalf("body = %q, want advised payload", recorder.Body.String())
	}
	wantEvents := []string{"before:" + arkjson.ContentType, "after:" + arkjson.ContentType}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", events, wantEvents)
	}
}

func TestReaderReadAdviceBeforeErrorSkipsConverter(t *testing.T) {
	t.Parallel()

	errBlocked := errors.New("blocked")
	var got string
	recorder := serveReadMessage(t, "text/plain", "blocked body", func(ctx *arkweb.Context) (arkweb.Result, error) {
		reader := message.NewReader(message.WithReadAdvice(message.ReadAdviceFunc{
			Before: func(*arkweb.Context, message.ReadAdviceContext) error {
				return errBlocked
			},
		}))
		err := reader.Read(ctx, &got)
		if !errors.Is(err, errBlocked) {
			t.Fatalf("err = %v, want blocked", err)
		}
		return arkweb.NoContent(), nil
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", recorder.Code, recorder.Body.String())
	}
	if got != "" {
		t.Fatalf("got = %q, want converter skipped", got)
	}
}
