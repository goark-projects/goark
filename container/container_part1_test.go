package container_test

import (
	"context"
	stderrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
)

func TestContainer_whenSingletonIsBeingPopulated_shouldNotExposeEarlySingletonToExternalGet(
	t *testing.T,
) {
	registry := container.NewRegistry()
	injectorEntered := make(chan struct{})
	releaseInjector := make(chan struct{})
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	},
		container.WithLazy(),
		container.WithTypedDependencyInjector(func(ctx context.Context, resolver container.Resolver,
			service *testService) error {
			close(injectorEntered)
			select {
			case <-releaseInjector:
			case <-ctx.Done():
				return ctx.Err()
			}
			service.Repository = &testRepository{ID: 42}
			return nil
		}),
	); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	runtimeContainer, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	firstDone := make(chan error, 1)
	go func() {
		_, err := container.Get[*testService](context.Background(), runtimeContainer, "service")
		firstDone <- err
	}()
	<-injectorEntered

	type resolveResult struct {
		service *testService
		err     error
	}
	secondDone := make(chan resolveResult, 1)
	go func() {
		service, err := container.Get[*testService](
			context.Background(),
			runtimeContainer,
			"service",
		)
		secondDone <- resolveResult{service: service, err: err}
	}()

	select {
	case result := <-secondDone:
		close(releaseInjector)
		if result.err != nil {
			t.Fatalf("external get should wait instead of returning early error: %v", result.err)
		}
		t.Fatalf("external get received singleton before injection completed: %#v", result.service)
	case <-time.After(20 * time.Millisecond):
	}

	close(releaseInjector)
	if err := <-firstDone; err != nil {
		t.Fatalf("first resolve failed: %v", err)
	}
	result := <-secondDone
	if result.err != nil {
		t.Fatalf("second resolve failed: %v", result.err)
	}
	if result.service.Repository == nil || result.service.Repository.ID != 42 {
		t.Fatalf("second resolve should receive populated singleton, got %#v", result.service)
	}
}

func TestContainer_whenResolvingDependencyGraph_shouldReturnSingletons(t *testing.T) {
	var repoCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
		repoCreated.Add(1)
		return &testRepository{ID: 1}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(ctx context.Context,
		resolver container.Resolver) (*testService, error) {
		repo, err := container.Get[*testRepository](ctx, resolver, "repo")
		if err != nil {
			return nil, err
		}
		return &testService{Repository: repo}, nil
	}, container.WithDependencies("repo")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}
	if err := runtimeContainer.InitializeSingletons(context.Background()); err != nil {
		t.Fatalf("initialize singletons failed: %v", err)
	}

	first := container.MustGet[*testService](context.Background(), runtimeContainer, "service")
	second := container.MustGet[*testService](context.Background(), runtimeContainer, "service")
	if first != second {
		t.Fatal("expected singleton service instance")
	}
	if first.Repository == nil || first.Repository.ID != 1 {
		t.Fatalf("unexpected dependency: %#v", first.Repository)
	}
	if repoCreated.Load() != 1 {
		t.Fatalf("expected repo to be created once, got %d", repoCreated.Load())
	}
}

func TestContainer_whenResolvingByTypeWithQualifierTypeMismatch_shouldReturnTypeMismatch(
	t *testing.T,
) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "worker", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register worker failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.GetByType[testWorker](
		context.Background(),
		runtimeContainer,
		container.WithQualifier("repo"),
	)
	if err == nil {
		t.Fatal("expected type mismatch")
	}
	if !arkerrors.Is(err, arkerrors.CodeTypeMismatch) {
		t.Fatalf("expected type mismatch, got %v", err)
	}
}

func TestContainer_whenRegistryChangesAfterCreation_shouldKeepDefinitionSnapshot(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}); err != nil {
		t.Fatalf("register service after container creation failed: %v", err)
	}

	names := runtimeContainer.Names()
	if len(names) != 1 || names[0] != "repo" {
		t.Fatalf("container should keep original definition snapshot, got %#v", names)
	}
	_, err = runtimeContainer.Get(context.Background(), "service")
	if err == nil {
		t.Fatal("container should not see registry changes after creation")
	}
	if !arkerrors.Is(err, arkerrors.CodeNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
func TestContainer_whenTypeHasMultipleSamePriorityCandidates_shouldReturnConflict(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "a", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPriority(10)); err != nil {
		t.Fatalf("register worker a failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "b", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}, container.WithPriority(10)); err != nil {
		t.Fatalf("register worker b failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.GetByType[testWorker](context.Background(), runtimeContainer)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !arkerrors.Is(err, arkerrors.CodeConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestContainer_whenDependsOnGraphHasCycle_shouldFailEvenWhenCircularReferencesAllowed(
	t *testing.T,
) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repository", func(context.Context,
		container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithDependsOn("service")); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithDependsOn("repository")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	_, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

func TestContainer_whenProviderPanics_shouldReturnCreationError(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
		panic(stderrors.New("boom"))
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.Get[*testRepository](context.Background(), runtimeContainer, "repo")
	if err == nil {
		t.Fatal("expected creation error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCreation) {
		t.Fatalf("expected creation error, got %v", err)
	}
}

func TestContainer_whenInjectionDependencyGraphHasCycle_shouldFailFastByDefault(t *testing.T) {
	registry := circularRegistry(t, false)

	_, err := container.New(registry)
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

func (secondaryWorker) Work() string {
	return "secondary"
}
