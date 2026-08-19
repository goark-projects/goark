package goark_test

import (
	"context"
	"testing"

	"github.com/goark-projects/goark"
	"github.com/goark-projects/goark/container"
)

type facadeRepository struct {
	ID int
}

type facadeService struct {
	Repository *facadeRepository
}

func TestGoarkFacade_whenRegisteringGeneratedStyleProviders_shouldResolveBeans(t *testing.T) {
	app := goark.MustNew()
	repo := &facadeRepository{ID: 9}
	if err := goark.RegisterInstance[*facadeRepository](app, "repo", repo); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := goark.Register[*facadeService](app, "service", func(ctx context.Context, resolver container.Resolver) (*facadeService, error) {
		resolvedRepo, err := goark.Get[*facadeRepository](ctx, resolver, "repo")
		if err != nil {
			return nil, err
		}
		return &facadeService{Repository: resolvedRepo}, nil
	}, container.WithDependencies("repo")); err != nil {
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
