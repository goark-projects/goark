package container_test

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
)

func TestContainer_whenSingletonResolvedConcurrently_shouldResolveDependsOnOnce(t *testing.T) {
	var setupCreated atomic.Int64
	var serviceCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "setup", func(context.Context,
		container.Resolver) (*testRepository, error) {
		setupCreated.Add(1)
		time.Sleep(10 * time.Millisecond)
		return &testRepository{}, nil
	}, container.WithPrototype()); err != nil {
		t.Fatalf("register setup failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		serviceCreated.Add(1)
		time.Sleep(10 * time.Millisecond)
		return &testService{}, nil
	}, container.WithLazy(), container.WithDependsOn("setup")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := container.Get[*testService](context.Background(), runtimeContainer, "service")
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("resolve service failed: %v", err)
	}
	if setupCreated.Load() != 1 {
		t.Fatalf("depends-on setup should be resolved once, got %d", setupCreated.Load())
	}
	if serviceCreated.Load() != 1 {
		t.Fatalf("singleton service should be created once, got %d", serviceCreated.Load())
	}
}

func circularRegistry(t *testing.T, prototype bool) *container.Registry {
	t.Helper()
	registry := container.NewRegistry()
	serviceOptions := []container.Option{
		container.WithInjectionDependencies("repository"),
		container.WithTypedDependencyInjector(
			func(ctx context.Context, resolver container.Resolver, service *circularService) error {
				repository, err := container.Get[*circularRepository](ctx, resolver, "repository")
				if err != nil {
					return err
				}
				service.Repository = repository
				return nil
			},
		),
	}
	repositoryOptions := []container.Option{
		container.WithInjectionDependencies("service"),
		container.WithTypedDependencyInjector(
			func(ctx context.Context, resolver container.Resolver, repository *circularRepository) error {
				service, err := container.Get[*circularService](ctx, resolver, "service")
				if err != nil {
					return err
				}
				repository.Service = service
				return nil
			},
		),
	}
	if prototype {
		serviceOptions = append(serviceOptions, container.WithPrototype())
	}
	if err := container.Register[*circularService](registry, "service", func(context.Context,
		container.Resolver) (*circularService, error) {
		return &circularService{}, nil
	}, serviceOptions...); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := container.Register[*circularRepository](registry, "repository", func(
		context.Context, container.Resolver) (*circularRepository, error) {
		return &circularRepository{}, nil
	}, repositoryOptions...); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	return registry
}

func TestContainer_whenGetAllByType_shouldUseBeanOrder(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "zLast", func(context.Context,
		container.Resolver) (testWorker, error) {
		return namedWorker("last"), nil
	}, container.WithOrder(100)); err != nil {
		t.Fatalf("register zLast failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "middle", func(context.Context,
		container.Resolver) (testWorker, error) {
		return namedWorker("middle"), nil
	}); err != nil {
		t.Fatalf("register middle failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "aFirst", func(context.Context,
		container.Resolver) (testWorker, error) {
		return namedWorker("first"), nil
	}, container.WithOrder(-100)); err != nil {
		t.Fatalf("register aFirst failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	workers, err := container.GetAllByType[testWorker](context.Background(), runtimeContainer)
	if err != nil {
		t.Fatalf("get all workers failed: %v", err)
	}
	got := make([]string, 0, len(workers))
	for _, worker := range workers {
		got = append(got, worker.Work())
	}
	expected := []string{"first", "middle", "last"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("workers = %#v, want %#v", got, expected)
	}
}

func TestContainer_whenDependsOnDeclared_shouldInitializeDependencyBeforeBean(t *testing.T) {
	log := make([]string, 0, 2)
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		log = append(log, "service")
		return &testService{}, nil
	}, container.WithDependsOn("zzRepository")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := container.Register[*testRepository](registry, "zzRepository", func(context.Context,
		container.Resolver) (*testRepository, error) {
		log = append(log, "repository")
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	if err := runtimeContainer.InitializeSingletons(context.Background()); err != nil {
		t.Fatalf("initialize singletons failed: %v", err)
	}
	expected := []string{"repository", "service"}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("depends-on order should be enforced, got %#v", log)
	}
}

func TestContainer_whenResolvingByTypeWithPriority_shouldSelectHighestPriority(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "slow", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}, container.WithPriority(100)); err != nil {
		t.Fatalf("register slow failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "fast", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPriority(10)); err != nil {
		t.Fatalf("register fast failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	worker, err := container.GetByType[testWorker](context.Background(), runtimeContainer)
	if err != nil {
		t.Fatalf("resolve by priority failed: %v", err)
	}
	if worker.Work() != "primary" {
		t.Fatalf("expected highest priority worker, got %q", worker.Work())
	}
}

func TestContainer_whenResolvingByTypeWithEmptyQualifier_shouldReturnInvalidArgument(t *testing.T) {
	registry := container.NewRegistry()
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
		container.WithQualifier(" "),
	)
	if err == nil {
		t.Fatal("expected empty qualifier error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestContainer_whenDependsOnGraphHasCycle_shouldFailFast(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "a", func(context.Context,
		container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithDependsOn("b")); err != nil {
		t.Fatalf("register a failed: %v", err)
	}
	if err := container.Register[*testService](registry, "b", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithDependsOn("a")); err != nil {
		t.Fatalf("register b failed: %v", err)
	}

	_, err := container.New(registry)
	if err == nil {
		t.Fatal("expected depends-on cycle error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

func TestContainer_whenPrototypeGraphCycles_shouldFailEvenWhenCircularReferencesAllowed(
	t *testing.T,
) {
	registry := circularRegistry(t, true)

	_, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

type testService struct {
	Repository *testRepository
}

type testWorker interface {
	Work() string
}

type primaryWorker struct{}

type namedWorker string
