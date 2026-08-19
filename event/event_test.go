package event_test

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/event"
)

type createdEvent struct {
	ID string
}

type ignoredEvent struct{}

func TestBus_whenPublishingTypedEvent_shouldInvokeHandlersInOrder(t *testing.T) {
	bus := event.NewBus()
	calls := make([]string, 0, 2)
	if err := event.Subscribe[createdEvent](bus, func(_ context.Context, evt createdEvent) error {
		calls = append(calls, "second:"+evt.ID)
		return nil
	}, event.WithOrder(20)); err != nil {
		t.Fatalf("subscribe second failed: %v", err)
	}
	if err := event.Subscribe[createdEvent](bus, func(_ context.Context, evt createdEvent) error {
		calls = append(calls, "first:"+evt.ID)
		return nil
	}, event.WithOrder(10)); err != nil {
		t.Fatalf("subscribe first failed: %v", err)
	}

	if err := bus.Publish(context.Background(), createdEvent{ID: "42"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	expected := []string{"first:42", "second:42"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestBus_whenEventTypeDoesNotMatch_shouldSkipHandler(t *testing.T) {
	bus := event.NewBus()
	called := false
	if err := event.Subscribe[ignoredEvent](bus, func(context.Context, ignoredEvent) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	if err := bus.Publish(context.Background(), createdEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if called {
		t.Fatal("handler should not be called")
	}
}

func TestBus_whenHandlerFails_shouldStopAndWrapError(t *testing.T) {
	bus := event.NewBus()
	calls := 0
	if err := event.Subscribe[createdEvent](bus, func(context.Context, createdEvent) error {
		calls++
		return stderrors.New("failed")
	}, event.WithName("failing"), event.WithOrder(1)); err != nil {
		t.Fatalf("subscribe failing handler failed: %v", err)
	}
	if err := event.Subscribe[createdEvent](bus, func(context.Context, createdEvent) error {
		calls++
		return nil
	}, event.WithName("after"), event.WithOrder(2)); err != nil {
		t.Fatalf("subscribe after handler failed: %v", err)
	}

	err := bus.Publish(context.Background(), createdEvent{})
	if err == nil {
		t.Fatal("expected publish error")
	}
	if !arkerrors.Is(err, arkerrors.CodeLifecycle) {
		t.Fatalf("expected lifecycle error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected publish to stop on first error, got %d calls", calls)
	}
}
