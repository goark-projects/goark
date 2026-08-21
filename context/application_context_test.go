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

type namedRuntimeComponent struct {
	name string
	log  *[]string
}

func (c *namedRuntimeComponent) Start(stdcontext.Context) error {
	*c.log = append(*c.log, "start:"+c.name)
	return nil
}

func (c *namedRuntimeComponent) Stop(stdcontext.Context) error {
	*c.log = append(*c.log, "stop:"+c.name)
	return nil
}

func (c *namedRuntimeComponent) Close() error {
	*c.log = append(*c.log, "close:"+c.name)
	return nil
}

type circularRuntimeService struct {
	Repository *circularRuntimeRepository
	log        *[]string
}

func (s *circularRuntimeService) Start(stdcontext.Context) error {
	*s.log = append(*s.log, "start:service")
	return nil
}

func (s *circularRuntimeService) Stop(stdcontext.Context) error {
	*s.log = append(*s.log, "stop:service")
	return nil
}

func (s *circularRuntimeService) Close() error {
	*s.log = append(*s.log, "close:service")
	return nil
}

type circularRuntimeRepository struct {
	Service *circularRuntimeService
	log     *[]string
}

func (r *circularRuntimeRepository) Start(stdcontext.Context) error {
	*r.log = append(*r.log, "start:repository")
	return nil
}

func (r *circularRuntimeRepository) Stop(stdcontext.Context) error {
	*r.log = append(*r.log, "stop:repository")
	return nil
}

func (r *circularRuntimeRepository) Close() error {
	*r.log = append(*r.log, "close:repository")
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

func TestApplicationContext_whenAllowedCircularLifecycleBeans_shouldStartAndClose(t *testing.T) {
	log := make([]string, 0, 6)
	app, err := appcontext.New(appcontext.WithAllowCircularReferences(true))
	if err != nil {
		t.Fatalf("create app context failed: %v", err)
	}
	service, err := container.NewDefinition[*circularRuntimeService]("service", func(stdcontext.Context, container.Resolver) (*circularRuntimeService, error) {
		return &circularRuntimeService{log: &log}, nil
	},
		container.WithInjectionDependencies("repository"),
		container.WithTypedDependencyInjector(func(ctx stdcontext.Context, resolver container.Resolver, service *circularRuntimeService) error {
			repository, err := container.Get[*circularRuntimeRepository](ctx, resolver, "repository")
			if err != nil {
				return err
			}
			service.Repository = repository
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("create service definition failed: %v", err)
	}
	repository, err := container.NewDefinition[*circularRuntimeRepository]("repository", func(stdcontext.Context, container.Resolver) (*circularRuntimeRepository, error) {
		return &circularRuntimeRepository{log: &log}, nil
	},
		container.WithInjectionDependencies("service"),
		container.WithTypedDependencyInjector(func(ctx stdcontext.Context, resolver container.Resolver, repository *circularRuntimeRepository) error {
			service, err := container.Get[*circularRuntimeService](ctx, resolver, "service")
			if err != nil {
				return err
			}
			repository.Service = service
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("create repository definition failed: %v", err)
	}
	if err := app.RegisterDefinition(service); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := app.RegisterDefinition(repository); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}

	if err := app.Start(stdcontext.Background()); err != nil {
		t.Fatalf("start app failed: %v", err)
	}
	resolved := container.MustGet[*circularRuntimeService](stdcontext.Background(), app, "service")
	if resolved.Repository == nil || resolved.Repository.Service != resolved {
		t.Fatalf("expected resolved circular lifecycle beans, got %#v", resolved.Repository)
	}
	if err := app.Close(stdcontext.Background()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
	if len(log) != 6 {
		t.Fatalf("expected lifecycle callbacks for both beans, got %#v", log)
	}
}

func TestApplicationContext_whenBeanDependsOnLifecycleBean_shouldDestroyDependentFirst(t *testing.T) {
	log := make([]string, 0, 6)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app context failed: %v", err)
	}
	repository, err := container.NewDefinition[*namedRuntimeComponent]("zzRepository", func(stdcontext.Context, container.Resolver) (*namedRuntimeComponent, error) {
		return &namedRuntimeComponent{name: "repository", log: &log}, nil
	})
	if err != nil {
		t.Fatalf("create repository definition failed: %v", err)
	}
	service, err := container.NewDefinition[*namedRuntimeComponent]("aaService", func(stdcontext.Context, container.Resolver) (*namedRuntimeComponent, error) {
		return &namedRuntimeComponent{name: "service", log: &log}, nil
	}, container.WithInjectionDependencies("zzRepository"))
	if err != nil {
		t.Fatalf("create service definition failed: %v", err)
	}
	if err := app.RegisterDefinition(repository); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	if err := app.RegisterDefinition(service); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	if err := app.Start(stdcontext.Background()); err != nil {
		t.Fatalf("start app failed: %v", err)
	}
	if err := app.Close(stdcontext.Background()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}

	expected := []string{
		"start:repository",
		"start:service",
		"stop:service",
		"stop:repository",
		"close:service",
		"close:repository",
	}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected dependency lifecycle log: %#v", log)
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
