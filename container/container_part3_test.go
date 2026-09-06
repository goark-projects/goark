package container_test

import (
	"context"
	"sync/atomic"
	"testing"

	"goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
)

func TestDefinition_whenOptionsApplied_shouldExposeBeanMetadata(t *testing.T) {
	definition, err := container.NewDefinition[*testRepository](
		"repo",
		func(context.Context, container.Resolver) (*testRepository, error) {
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
	if len(definition.DependsOn) != 2 || definition.DependsOn[0] != "database" ||
		definition.DependsOn[1] != "cache" {
		t.Fatalf("unexpected depends-on metadata: %#v", definition.DependsOn)
	}
	if len(definition.Dependencies) != 2 || definition.Dependencies[0] != "database" ||
		definition.Dependencies[1] != "cache" {
		t.Fatalf(
			"dependencies should include depends-on metadata for legacy readers: %#v",
			definition.Dependencies,
		)
	}
	if len(definition.DependencyDescriptors) != 2 {
		t.Fatalf("expected two dependency descriptors, got %#v", definition.DependencyDescriptors)
	}
	for _, descriptor := range definition.DependencyDescriptors {
		if descriptor.Kind != container.DependencyKindDependsOn ||
			descriptor.Source != container.DependencySourceManual {
			t.Fatalf("expected manual depends-on descriptor, got %#v", descriptor)
		}
	}
}

func TestRegistry_whenDefinitionsReturned_shouldReturnDefensiveCopies(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
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
		t.Fatalf(
			"factory dependencies should not populate depends-on, got %#v",
			definition.DependsOn,
		)
	}
	if len(definition.DependencyDescriptors) != 1 ||
		definition.DependencyDescriptors[0].Name != "repo" {
		t.Fatalf(
			"dependency descriptors should be immutable, got %#v",
			definition.DependencyDescriptors,
		)
	}
}

func TestContainer_whenDependencyGraphHasCycle_shouldReturnCircularDependency(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "a", func(ctx context.Context,
		resolver container.Resolver) (*testRepository, error) {
		_, err := container.Get[*testService](ctx, resolver, "b")
		if err != nil {
			return nil, err
		}
		return &testRepository{}, nil
	}); err != nil {
		t.Fatalf("register a failed: %v", err)
	}
	if err := container.Register[*testService](registry, "b", func(ctx context.Context,
		resolver container.Resolver) (*testService, error) {
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

func TestContainer_whenAllowCircularReferencesEnabled_shouldResolveSingletonFieldCycle(
	t *testing.T,
) {
	registry := circularRegistry(t, false)
	runtimeContainer, err := container.New(registry, container.WithAllowCircularReferences(true))
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	if err := runtimeContainer.InitializeSingletons(context.Background()); err != nil {
		t.Fatalf("initialize singletons failed: %v", err)
	}
	service := container.MustGet[*circularService](
		context.Background(),
		runtimeContainer,
		"service",
	)
	repository := container.MustGet[*circularRepository](
		context.Background(),
		runtimeContainer,
		"repository",
	)
	if service.Repository != repository {
		t.Fatalf("service should reference repository, got %#v", service.Repository)
	}
	if repository.Service != service {
		t.Fatalf("repository should reference early service singleton, got %#v", repository.Service)
	}
}

func TestContainer_whenPrimaryAndPriorityBothExist_shouldSelectPrimary(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[testWorker](registry, "priority", func(context.Context,
		container.Resolver) (testWorker, error) {
		return secondaryWorker{}, nil
	}, container.WithPriority(0)); err != nil {
		t.Fatalf("register priority failed: %v", err)
	}
	if err := container.Register[testWorker](registry, "primary", func(context.Context,
		container.Resolver) (testWorker, error) {
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

func TestContainer_whenFactoryDependencyGraphHasCycle_shouldFailEvenWhenCircularReferencesAllowed(
	t *testing.T,
) {
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repository", func(context.Context,
		container.Resolver) (*testRepository, error) {
		return &testRepository{}, nil
	}, container.WithFactoryDependencies("service")); err != nil {
		t.Fatalf("register repository failed: %v", err)
	}
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
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

func TestContainer_whenBeanIsPrototype_shouldCreateEachTime(t *testing.T) {
	var created atomic.Int64
	registry := container.NewRegistry()
	if err := container.Register[*testRepository](registry, "repo", func(context.Context,
		container.Resolver) (*testRepository, error) {
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

func TestContainer_whenOptionalInjectionDependencyIsMissing_shouldCreateContainer(t *testing.T) {
	registry := container.NewRegistry()
	if err := container.Register[*testService](registry, "service", func(context.Context,
		container.Resolver) (*testService, error) {
		return &testService{}, nil
	}, container.WithOptionalInjectionDependencies("missingCache")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	if _, err := container.New(registry); err != nil {
		t.Fatalf("optional missing dependency should not fail: %v", err)
	}
}

type testRepository struct {
	ID int
}

type circularRepository struct {
	Service *circularService
}

func (w namedWorker) Work() string {
	return string(w)
}
