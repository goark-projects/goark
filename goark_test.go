package goark_test

import (
	"context"
	"testing"

	"github.com/goark-projects/goark"
)

type facadeRepository struct {
	ID int
}

type facadeService struct {
	Repository *facadeRepository
}

type facadeWorker interface {
	Work() string
}

type facadePrimaryWorker struct{}

func (facadePrimaryWorker) Work() string {
	return "primary"
}

type facadeSecondaryWorker struct{}

func (facadeSecondaryWorker) Work() string {
	return "secondary"
}

func TestGoarkFacade_whenRegisteringGeneratedStyleProviders_shouldResolveBeans(t *testing.T) {
	app := goark.MustNew()
	repo := &facadeRepository{ID: 9}
	if err := goark.RegisterInstance[*facadeRepository](app, "repo", repo); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := goark.Register[*facadeService](app, "service", func(ctx context.Context, resolver goark.Resolver) (*facadeService, error) {
		resolvedRepo, err := goark.Get[*facadeRepository](ctx, resolver, "repo")
		if err != nil {
			return nil, err
		}
		return &facadeService{Repository: resolvedRepo}, nil
	}, goark.WithDependencies("repo")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := app.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	service := goark.MustGet[*facadeService](context.Background(), app, "service")
	if service.Repository != repo {
		t.Fatalf("expected generated-style registration to wire repo, got %#v", service.Repository)
	}
}

func TestGoarkFacade_whenUsingBeanMetadataOptions_shouldResolveByFacadeOptions(t *testing.T) {
	app := goark.MustNew()
	if err := goark.Register[facadeWorker](app, "secondary", func(context.Context, goark.Resolver) (facadeWorker, error) {
		return facadeSecondaryWorker{}, nil
	}, goark.WithPriority(100)); err != nil {
		t.Fatalf("register secondary worker failed: %v", err)
	}
	if err := goark.Register[facadeWorker](app, "primary", func(context.Context, goark.Resolver) (facadeWorker, error) {
		return facadePrimaryWorker{}, nil
	}, goark.WithPrimary(), goark.WithLazy(), goark.WithOrder(10)); err != nil {
		t.Fatalf("register primary worker failed: %v", err)
	}
	if err := app.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	worker := goark.MustGetByType[facadeWorker](context.Background(), app)
	if worker.Work() != "primary" {
		t.Fatalf("expected primary worker, got %q", worker.Work())
	}
	qualified, err := goark.GetByType[facadeWorker](context.Background(), app, goark.WithQualifier("secondary"))
	if err != nil {
		t.Fatalf("resolve qualified worker failed: %v", err)
	}
	if qualified.Work() != "secondary" {
		t.Fatalf("expected qualified secondary worker, got %q", qualified.Work())
	}
}
