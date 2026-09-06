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

func TestContainer_whenSingletonResolvedConcurrently_shouldCreateOnce(t *testing.T) {
	var created atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
		created.Add(1)
		time.Sleep(20 * time.Millisecond)
		return &testRepository{ID: 7}, nil
	}, container.WithLazy()); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	const goroutines = 64
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	results := make(chan *testRepository, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo, err := container.Get[*testRepository](
				context.Background(),
				runtimeContainer,
				"repo",
			)
			if err != nil {
				errs <- err
				return
			}
			results <- repo
		}()
	}
	wg.Wait()
	close(errs)
	close(results)

	for err := range errs {
		t.Fatalf("resolve failed: %v", err)
	}
	var first *testRepository
	for result := range results {
		if first == nil {
			first = result
			continue
		}
		if first != result {
			t.Fatal("expected all goroutines to receive same singleton instance")
		}
	}
	if created.Load() != 1 {
		t.Fatalf("expected one creation, got %d", created.Load())
	}
}

func TestDefinition_whenFactoryAndInjectionDependenciesUsed_shouldKeepDependsOnSeparate(
	t *testing.T,
) {
	definition, err := container.NewDefinition[*testRepository](
		"repo",
		func(context.Context, container.Resolver) (*testRepository, error) {
			return &testRepository{}, nil
		},
		container.WithDependsOn("database, cache"),
		container.WithFactoryDependencies("factoryDependency"),
		container.WithInjectionDependencies("fieldDependency"),
	)
	if err != nil {
		t.Fatalf("new definition failed: %v", err)
	}

	if len(definition.DependsOn) != 2 || definition.DependsOn[0] != "database" ||
		definition.DependsOn[1] != "cache" {
		t.Fatalf("expected only manual depends-on names, got %#v", definition.DependsOn)
	}
	expectedDependencies := []string{"database", "cache", "factoryDependency", "fieldDependency"}
	if !reflect.DeepEqual(definition.Dependencies, expectedDependencies) {
		t.Fatalf("expected all dependency names, got %#v", definition.Dependencies)
	}
	kinds := make(map[string]container.DependencyKind)
	for _, descriptor := range definition.DependencyDescriptors {
		kinds[descriptor.Name] = descriptor.Kind
	}
	if kinds["database"] != container.DependencyKindDependsOn ||
		kinds["cache"] != container.DependencyKindDependsOn {
		t.Fatalf("manual descriptors should be depends-on, got %#v", kinds)
	}
	if kinds["factoryDependency"] != container.DependencyKindFactory {
		t.Fatalf("factory dependency kind mismatch: %#v", kinds)
	}
	if kinds["fieldDependency"] != container.DependencyKindInjection {
		t.Fatalf("field dependency kind mismatch: %#v", kinds)
	}
}

func TestContainer_whenSingletonAlreadyCreated_shouldNotResolveDependsOnAgain(t *testing.T) {
	var setupCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "setup", func(context.Context,
		container.Resolver) (*testRepository, error) {
		setupCreated.Add(1)
		return &testRepository{}, nil
	}, container.WithPrototype()); err != nil {
		t.Fatalf("register setup failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithDependsOn("setup")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	first := container.MustGet[*testService](context.Background(), runtimeContainer, "service")
	second := container.MustGet[*testService](context.Background(), runtimeContainer, "service")
	if first != second {
		t.Fatal("expected singleton service")
	}
	if setupCreated.Load() != 1 {
		t.Fatalf(
			"depends-on prototype should be resolved once before singleton creation, got %d",
			setupCreated.Load(),
		)
	}
}

func TestContainer_whenResolvingByTypeWithQualifier_shouldSelectNamedCandidate(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "primary", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPrimary()); err != nil {
		t.Fatalf("register primary failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "secondary", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register secondary failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	worker, err := container.GetByType[testWorker](
		context.Background(),
		runtimeContainer,
		container.WithQualifier("secondary"),
	)
	if err != nil {
		t.Fatalf("resolve by qualifier failed: %v", err)
	}
	if worker.Work() != "secondary" {
		t.Fatalf("expected secondary worker, got %q", worker.Work())
	}
}

func TestContainer_whenResolvingByTypeWithPrimary_shouldSelectPrimary(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "secondary", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register secondary failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "primary", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPrimary()); err != nil {
		t.Fatalf("register primary failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	worker, err := container.GetByType[testWorker](context.Background(), runtimeContainer)
	if err != nil {
		t.Fatalf("resolve by type failed: %v", err)
	}
	if worker.Work() != "primary" {
		t.Fatalf("expected primary worker, got %q", worker.Work())
	}
}

func TestContainer_whenTypeHasMultipleCandidatesWithoutPrimary_shouldReturnConflict(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "a", func(context.Context,
		container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register worker a failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "b", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}); err != nil {
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

func TestContainer_whenResolvingByTypeWithNilOption_shouldIgnoreOption(t *testing.T) {
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

	worker, err := container.GetByType[testWorker](context.Background(), runtimeContainer, nil)
	if err != nil {
		t.Fatalf("resolve by type with nil option failed: %v", err)
	}
	if worker.Work() != "primary" {
		t.Fatalf("expected primary worker, got %q", worker.Work())
	}
}

func TestContainer_whenDependencyIsMissing_shouldFailFast(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithDependsOn("missing")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	_, err := container.New(registry)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

type circularService struct {
	Repository *circularRepository
}

func (primaryWorker) Work() string {
	return "primary"
}

type secondaryWorker struct{}
