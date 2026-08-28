package container_test

import (
	"context"
	stderrors "errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
)

type testRepository struct {
	ID int
}

type testService struct {
	Repository *testRepository
}

type circularService struct {
	Repository *circularRepository
}

type circularRepository struct {
	Service *circularService
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

func TestContainer_whenSingletonIsBeingPopulated_shouldNotExposeEarlySingletonToExternalGet(t *testing.T) {
	registry := container.NewRegistry()
	injectorEntered := make(chan struct{})
	releaseInjector := make(chan struct{})
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
		return &testService{}, nil
	},
		container.WithLazy(),
		container.WithTypedDependencyInjector(func(ctx context.Context, resolver container.Resolver, service *testService) error {
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
		service, err := container.Get[*testService](context.Background(), runtimeContainer, "service")
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

func TestContainer_whenDependsOnDeclared_shouldInitializeDependencyBeforeBean(t *testing.T) {
	log := make([]string, 0, 2)
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
		log = append(log, "service")
		return &testService{}, nil
	}, container.WithDependsOn("zzRepository")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := container.Register[*testRepository](registry, "zzRepository", func(context.Context, container.Resolver) (*testRepository, error) {
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

func TestContainer_whenSingletonAlreadyCreated_shouldNotResolveDependsOnAgain(t *testing.T) {
	var setupCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "setup", func(context.Context, container.Resolver) (*testRepository, error) {
		setupCreated.Add(1)
		return &testRepository{}, nil
	}, container.WithPrototype()); err != nil {
		t.Fatalf("register setup failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
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
		t.Fatalf("depends-on prototype should be resolved once before singleton creation, got %d", setupCreated.Load())
	}
}

func TestContainer_whenSingletonResolvedConcurrently_shouldResolveDependsOnOnce(t *testing.T) {
	var setupCreated atomic.Int64
	var serviceCreated atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "setup", func(context.Context, container.Resolver) (*testRepository, error) {
		setupCreated.Add(1)
		time.Sleep(10 * time.Millisecond)
		return &testRepository{}, nil
	}, container.WithPrototype()); err != nil {
		t.Fatalf("register setup failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
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

func TestContainer_whenResolvingByTypeWithNilOption_shouldIgnoreOption(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "worker", func(context.Context, container.Resolver) (testWorker, error) {
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

func TestContainer_whenResolvingByTypeWithEmptyQualifier_shouldReturnInvalidArgument(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "worker", func(context.Context, container.Resolver) (testWorker, error) {
		return primaryWorker{}, nil
	}); err != nil {
		t.Fatalf("register worker failed: %v", err)
	}
	runtimeContainer, err := container.New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	_, err = container.GetByType[testWorker](context.Background(), runtimeContainer, container.WithQualifier(" "))
	if err == nil {
		t.Fatal("expected empty qualifier error")
	}
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
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

func TestContainer_whenGetAllByType_shouldUseBeanOrder(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "zLast", func(context.Context, container.Resolver) (testWorker, error) {
		return namedWorker("last"), nil
	}, container.WithOrder(100)); err != nil {
		t.Fatalf("register zLast failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "middle", func(context.Context, container.Resolver) (testWorker, error) {
		return namedWorker("middle"), nil
	}); err != nil {
		t.Fatalf("register middle failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "aFirst", func(context.Context, container.Resolver) (testWorker, error) {
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

type namedWorker string

func (w namedWorker) Work() string {
	return string(w)
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
		t.Fatalf("dependencies should include depends-on metadata for legacy readers: %#v", definition.Dependencies)
	}
	if len(definition.DependencyDescriptors) != 2 {
		t.Fatalf("expected two dependency descriptors, got %#v", definition.DependencyDescriptors)
	}
	for _, descriptor := range definition.DependencyDescriptors {
		if descriptor.Kind != container.DependencyKindDependsOn || descriptor.Source != container.DependencySourceManual {
			t.Fatalf("expected manual depends-on descriptor, got %#v", descriptor)
		}
	}
}

func TestDefinition_whenFactoryAndInjectionDependenciesUsed_shouldKeepDependsOnSeparate(t *testing.T) {
	definition, err := container.NewDefinition[*testRepository]("repo", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	},
		container.WithDependsOn("database, cache"),
		container.WithFactoryDependencies("factoryDependency"),
		container.WithInjectionDependencies("fieldDependency"),
	)
	if err != nil {
		t.Fatalf("new definition failed: %v", err)
	}

	if len(definition.DependsOn) != 2 || definition.DependsOn[0] != "database" || definition.DependsOn[1] != "cache" {
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
	if kinds["database"] != container.DependencyKindDependsOn || kinds["cache"] != container.DependencyKindDependsOn {
		t.Fatalf("manual descriptors should be depends-on, got %#v", kinds)
	}
	if kinds["factoryDependency"] != container.DependencyKindFactory {
		t.Fatalf("factory dependency kind mismatch: %#v", kinds)
	}
	if kinds["fieldDependency"] != container.DependencyKindInjection {
		t.Fatalf("field dependency kind mismatch: %#v", kinds)
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

func TestContainer_whenDependsOnGraphHasCycle_shouldFailFast(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "a", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithDependsOn("b")); err != nil {
		t.Fatalf("register a failed: %v", err)
	}
	if err := container.Register[*testService](registry, "b", func(context.Context, container.Resolver) (*testService, error) {
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

func TestContainer_whenAllowCircularReferencesEnabled_shouldResolveSingletonFieldCycle(t *testing.T) {
	registry := circularRegistry(t, false)
	runtimeContainer, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	if err := runtimeContainer.InitializeSingletons(context.Background()); err != nil {
		t.Fatalf("initialize singletons failed: %v", err)
	}
	service := container.MustGet[*circularService](context.Background(), runtimeContainer, "service")
	repository := container.MustGet[*circularRepository](context.Background(), runtimeContainer, "repository")
	if service.Repository != repository {
		t.Fatalf("service should reference repository, got %#v", service.Repository)
	}
	if repository.Service != service {
		t.Fatalf("repository should reference early service singleton, got %#v", repository.Service)
	}
}

func TestContainer_whenFactoryDependencyGraphHasCycle_shouldFailEvenWhenCircularReferencesAllowed(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repository", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithFactoryDependencies("service")); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithFactoryDependencies("repository")); err != nil {
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

func TestContainer_whenDependsOnGraphHasCycle_shouldFailEvenWhenCircularReferencesAllowed(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repository", func(context.Context, container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithDependsOn("service")); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
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

func TestContainer_whenPrototypeInjectionDependencyGraphHasCycle_shouldFailEvenWhenCircularReferencesAllowed(t *testing.T) {
	registry := circularRegistry(t, true)

	_, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCircularDependency) {
		t.Fatalf("expected circular dependency, got %v", err)
	}
}

func TestContainer_whenOptionalInjectionDependencyIsMissing_shouldCreateContainer(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context, container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithOptionalInjectionDependencies("missingCache")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	if _, err := container.New(registry); err != nil {
		t.Fatalf("optional missing dependency should not fail: %v", err)
	}
}

func circularRegistry(t *testing.T, prototype bool) *container.Registry {
	t.Helper()
	registry := container.NewRegistry()
	serviceOptions := []container.Option{
		container.WithInjectionDependencies("repository"),
		container.WithTypedDependencyInjector(func(ctx context.Context, resolver container.Resolver, service *circularService) error {
			repository, err := container.Get[*circularRepository](ctx, resolver, "repository")
			if err != nil {
				return err
			}
			service.Repository = repository
			return nil
		}),
	}
	repositoryOptions := []container.Option{
		container.WithInjectionDependencies("service"),
		container.WithTypedDependencyInjector(func(ctx context.Context, resolver container.Resolver, repository *circularRepository) error {
			service, err := container.Get[*circularService](ctx, resolver, "service")
			if err != nil {
				return err
			}
			repository.Service = service
			return nil
		}),
	}
	if prototype {
		serviceOptions = append(serviceOptions, container.WithPrototype())
	}
	if err := container.Register[*circularService](registry, "service", func(context.Context, container.Resolver) (*circularService, error) {
		return &circularService{}, nil
	}, serviceOptions...); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	if err := container.Register[*circularRepository](registry, "repository", func(context.Context, container.Resolver) (*circularRepository, error) {
		return &circularRepository{}, nil
	}, repositoryOptions...); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	return registry
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
	definitions[0].DependencyDescriptors[0].Name = "mutated"

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
	if len(definition.DependsOn) != 0 {
		t.Fatalf("factory dependencies should not populate depends-on, got %#v", definition.DependsOn)
	}
	if len(definition.DependencyDescriptors) != 1 || definition.DependencyDescriptors[0].Name != "repo" {
		t.Fatalf("dependency descriptors should be immutable, got %#v", definition.DependencyDescriptors)
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
