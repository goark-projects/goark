package container_test

import (
	"context"
	stderrors "errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goark-projects/goark/container"
	arkerrors "github.com/goark-projects/goark/errors"
)

type testRepository struct {
	ID int
}

type testService struct {
	Repository *testRepository
}

type testWorker interface {
	Work() string
}

type primaryWorker struct{}

func (primaryWorker) Work() string {
	return "primary"
}

type secondaryWorker struct{}

func (secondaryWorker) Work() string {
	return "secondary"
}

func TestContainer_whenResolvingDependencyGraph_shouldReturnSingletons(t *testing.T) {
	var repoCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
		repoCreated.Add(1)
		return &testRepository{ID: 1}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(ctx context.Context, resolver container.Resolver) (*testService, error) {
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

func TestContainer_whenSingletonResolvedConcurrently_shouldCreateOnce(t *testing.T) {
	var created atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
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
			repo, err := container.Get[*testRepository](context.Background(), runtimeContainer, "repo")
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

func TestContainer_whenBeanIsPrototype_shouldCreateEachTime(t *testing.T) {
	var created atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{ID: int(created.Add(1))}, nil
	}, container.WithPrototype()); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	first := container.MustGet[*testRepository](context.Background(), runtimeContainer, "repo")
	second := container.MustGet[*testRepository](context.Background(), runtimeContainer, "repo")
	if first == second {
		t.Fatal("expected prototype to create different instances")
	}
	if first.ID != 1 || second.ID != 2 {
		t.Fatalf("unexpected prototype IDs: %d, %d", first.ID, second.ID)
	}
}

func TestContainer_whenResolvingByTypeWithPrimary_shouldSelectPrimary(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "secondary", func(context.Context, container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register secondary failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "primary", func(context.Context, container.Resolver) (testWorker, error) {
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

func TestContainer_whenResolvingByTypeWithQualifier_shouldSelectNamedCandidate(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "primary", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPrimary()); err != nil {
		t.Fatalf("register primary failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "secondary", func(context.Context, container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register secondary failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	worker, err := container.GetByType[testWorker](context.Background(), runtimeContainer, container.WithQualifier("secondary"))
	if err != nil {
		t.Fatalf("resolve by qualifier failed: %v", err)
	}
	if worker.Work() != "secondary" {
		t.Fatalf("expected secondary worker, got %q", worker.Work())
	}
}

func TestContainer_whenResolvingByTypeWithQualifierTypeMismatch_shouldReturnTypeMismatch(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "worker", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register worker failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.GetByType[testWorker](context.Background(), runtimeContainer, container.WithQualifier("repo"))
	if err == nil {
		t.Fatal("expected type mismatch")
	}
	if !arkerrors.Is(err, arkerrors.CodeTypeMismatch) {
		t.Fatalf("expected type mismatch, got %v", err)
	}
}

func TestContainer_whenResolvingByTypeWithPriority_shouldSelectHighestPriority(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "slow", func(context.Context, container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}, container.WithPriority(100)); err != nil {
		t.Fatalf("register slow failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "fast", func(context.Context, container.Resolver) (testWorker, error) {
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

func TestContainer_whenPrimaryAndPriorityBothExist_shouldSelectPrimary(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "priority", func(context.Context, container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}, container.WithPriority(0)); err != nil {
		t.Fatalf("register priority failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "primary", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPrimary(), container.WithPriority(100)); err != nil {
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

func TestContainer_whenTypeHasMultipleSamePriorityCandidates_shouldReturnConflict(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "a", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}, container.WithPriority(10)); err != nil {
		t.Fatalf("register worker a failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "b", func(context.Context, container.Resolver) (testWorker, error) {
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

func TestContainer_whenTypeHasMultipleCandidatesWithoutPrimary_shouldReturnConflict(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "a", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register worker a failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "b", func(context.Context, container.Resolver) (testWorker, error) {
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

func TestContainer_whenDependencyIsMissing_shouldFailFast(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
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

func TestDefinition_whenOptionsApplied_shouldExposeBeanMetadata(t *testing.T) {
	definition, err := container.NewDefinition[*testRepository]("repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	},
		container.WithPrimary(),
		container.WithLazy(),
		container.WithPrototype(),
		container.WithDependsOn("database", "cache"),
		container.WithOrder(-100),
		container.WithPriority(10),
	)
	if err != nil {
		t.Fatalf("new definition failed: %v", err)
	}

	if !definition.Primary {
		t.Fatal("expected primary metadata")
	}
	if !definition.Lazy {
		t.Fatal("expected lazy metadata")
	}
	if definition.Scope != container.ScopePrototype {
		t.Fatalf("expected prototype scope, got %q", definition.Scope)
	}
	if definition.Order != -100 {
		t.Fatalf("expected order -100, got %d", definition.Order)
	}
	priority, ok := definition.Priority.Value()
	if !ok || priority != 10 {
		t.Fatalf("expected priority 10, got value=%d present=%v", priority, ok)
	}
	if len(definition.DependsOn) != 2 || definition.DependsOn[0] != "database" || definition.DependsOn[1] != "cache" {
		t.Fatalf("unexpected depends-on metadata: %#v", definition.DependsOn)
	}
	if len(definition.Dependencies) != 2 || definition.Dependencies[0] != "database" || definition.Dependencies[1] != "cache" {
		t.Fatalf("legacy dependencies should mirror depends-on metadata: %#v", definition.Dependencies)
	}
}

func TestDefinition_whenLegacyDependenciesUsed_shouldMirrorDependsOnMetadata(t *testing.T) {
	definition, err := container.NewDefinition[*testRepository]("repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithDependencies("database"))
	if err != nil {
		t.Fatalf("new definition failed: %v", err)
	}

	if len(definition.DependsOn) != 1 || definition.DependsOn[0] != "database" {
		t.Fatalf("expected legacy dependencies to populate depends-on, got %#v", definition.DependsOn)
	}
	if len(definition.Dependencies) != 1 || definition.Dependencies[0] != "database" {
		t.Fatalf("expected legacy dependencies to remain populated, got %#v", definition.Dependencies)
	}
}

func TestContainer_whenDependencyGraphHasCycle_shouldReturnCircularDependency(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "a", func(ctx context.Context, resolver container.Resolver) (*testRepository, error) {
		_, err := container.Get[*testService](ctx, resolver, "b")
		if err != nil {
			return nil, err
		}
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register a failed: %v", err)
	}
	if err := container.Register[*testService](registry, "b", func(ctx context.Context, resolver container.Resolver) (*testService, error) {
		_, err := container.Get[*testRepository](ctx, resolver, "a")
		if err != nil {
			return nil, err
		}
		return &testService{}, nil
	}); err != nil {
		t.Fatalf("register b failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.Get[*testRepository](context.Background(), runtimeContainer, "a")
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

func TestContainer_whenProviderPanics_shouldReturnCreationError(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
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

func TestRegistry_whenDefinitionsReturned_shouldReturnDefensiveCopies(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithDependencies("repo")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 1 {
		t.Fatalf("expected one definition, got %d", len(definitions))
	}
	definitions[0].Name = "mutated"
	definitions[0].Dependencies[0] = "mutated"
	definitions[0].DependsOn[0] = "mutated"

	definition, ok := registry.Definition("service")
	if !ok {
		t.Fatal("expected original service definition")
	}
	if definition.Name != "service" {
		t.Fatalf("definition name should be immutable, got %q", definition.Name)
	}
	if len(definition.Dependencies) != 1 || definition.Dependencies[0] != "repo" {
		t.Fatalf("definition dependencies should be immutable, got %#v", definition.Dependencies)
	}
	if len(definition.DependsOn) != 1 || definition.DependsOn[0] != "repo" {
		t.Fatalf("definition depends-on should be immutable, got %#v", definition.DependsOn)
	}
}

func TestContainer_whenRegistryChangesAfterCreation_shouldKeepDefinitionSnapshot(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register repo failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
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
