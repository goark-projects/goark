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

type orderedHandler struct {
	name  string
	order int
	calls *[]string
}

func (h *orderedHandler) Order() int {
	return h.order
}

func (h *orderedHandler) HandleEvent(context.Context, any) error {
	*h.calls = append(*h.calls, h.name)
	return nil
}

type priorityOrderedHandler struct {
	*orderedHandler
}

func (h *priorityOrderedHandler) PriorityOrdered() {
}

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

func TestBus_whenHandlersImplementOrdered_shouldUseGlobalOrderContracts(t *testing.T) {
	bus := event.NewBus()
	calls := make([]string, 0, 3)
	handlers := []event.Handler{
		&orderedHandler{name: "normal-early", order: 10, calls: &calls},
		&priorityOrderedHandler{orderedHandler: &orderedHandler{name: "priority", order: 100, calls: &calls}},
		&orderedHandler{name: "normal-late", order: 20, calls: &calls},
	}
	for _, handler := range handlers {
		if err := bus.Subscribe(handler); err != nil {
			t.Fatalf("subscribe handler failed: %v", err)
		}
	}

	if err := bus.Publish(context.Background(), createdEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	expected := []string{"priority", "normal-early", "normal-late"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected ordered handler calls: %#v", calls)
	}
}

func TestBus_whenExplicitOrderProvided_shouldOverrideHandlerOrder(t *testing.T) {
	bus := event.NewBus()
	calls := make([]string, 0, 2)
	if err := bus.Subscribe(&orderedHandler{name: "second", order: 1, calls: &calls}, event.WithOrder(20)); err != nil {
		t.Fatalf("subscribe second failed: %v", err)
	}
	if err := bus.Subscribe(&orderedHandler{name: "first", order: 100, calls: &calls}, event.WithOrder(10)); err != nil {
		t.Fatalf("subscribe first failed: %v", err)
	}

	if err := bus.Publish(context.Background(), createdEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	expected := []string{"first", "second"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected explicit order calls: %#v", calls)
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
