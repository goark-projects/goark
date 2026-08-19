package context_test

import (
	stdcontext "context"
	"reflect"
	"testing"

	"github.com/goark-projects/goark/container"
	appcontext "github.com/goark-projects/goark/context"
	arkerrors "github.com/goark-projects/goark/errors"
)

type runtimeComponent struct {
	log    *[]string
	events *[]string
}

func (c *runtimeComponent) Start(stdcontext.Context) error {
	*c.log = append(*c.log, "start")
	return nil
}

func (c *runtimeComponent) Stop(stdcontext.Context) error {
	*c.log = append(*c.log, "stop")
	return nil
}

func (c *runtimeComponent) Close() error {
	*c.log = append(*c.log, "close")
	return nil
}

func (c *runtimeComponent) HandleEvent(_ stdcontext.Context, evt any) error {
	switch evt.(type) {
	case appcontext.RefreshedEvent:
		*c.events = append(*c.events, "refreshed")
	case appcontext.StartedEvent:
		*c.events = append(*c.events, "started")
	case appcontext.StoppedEvent:
		*c.events = append(*c.events, "stopped")
	case appcontext.ClosedEvent:
		*c.events = append(*c.events, "closed")
	}
	return nil
}

func TestApplicationContext_whenStartedAndClosed_shouldManageLifecycleAndEvents(t *testing.T) {
	log := make([]string, 0, 3)
	events := make([]string, 0, 4)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app context failed: %v", err)
	}
	definition, err := container.NewDefinition[*runtimeComponent]("component", func(stdcontext.Context, container.Resolver) (*runtimeComponent, error) {
		return &runtimeComponent{log: &log, events: &events}, nil
	})
	if err != nil {
		t.Fatalf("create definition failed: %v", err)
	}
	if err := app.RegisterDefinition(definition); err != nil {
		t.Fatalf("register definition failed: %v", err)
	}

	if err := app.Start(stdcontext.Background()); err != nil {
		t.Fatalf("start app failed: %v", err)
	}
	component := container.MustGet[*runtimeComponent](stdcontext.Background(), app, "component")
	if component == nil {
		t.Fatal("expected component")
	}
	if err := app.Close(stdcontext.Background()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
	if err := app.Close(stdcontext.Background()); err != nil {
		t.Fatalf("close app should be idempotent: %v", err)
	}

	expectedLog := []string{"start", "stop", "close"}
	if !reflect.DeepEqual(log, expectedLog) {
		t.Fatalf("unexpected lifecycle log: %#v", log)
	}
	expectedEvents := []string{"refreshed", "started", "stopped", "closed"}
	if !reflect.DeepEqual(events, expectedEvents) {
		t.Fatalf("unexpected event log: %#v", events)
	}
}

func TestApplicationContext_whenGetBeforeRefresh_shouldReturnConflict(t *testing.T) {
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app context failed: %v", err)
	}

	_, err = app.Get(stdcontext.Background(), "missing")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !arkerrors.Is(err, arkerrors.CodeConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestApplicationContext_whenRegisterAfterRefresh_shouldReturnConflict(t *testing.T) {
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app context failed: %v", err)
	}
	if err := app.Refresh(stdcontext.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	definition, err := container.NewDefinition[*runtimeComponent]("component", func(stdcontext.Context, container.Resolver) (*runtimeComponent, error) {
		return &runtimeComponent{}, nil
	})
	if err != nil {
		t.Fatalf("create definition failed: %v", err)
	}

	err = app.RegisterDefinition(definition)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !arkerrors.Is(err, arkerrors.CodeConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
