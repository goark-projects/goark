package container

import (
	"context"
	"reflect"
	"testing"
)

type indexedRepository interface {
	Find() string
}

type indexedConcreteRepository struct {
	value string
}

func (r *indexedConcreteRepository) Find() string {
	return r.value
}

func TestContainerTypeIndex_whenConcreteBeanImplementsRegisteredInterface_shouldIndexInterface(t *testing.T) {
	registry := NewRegistry()
	if err := Register[indexedRepository](registry, "interfaceRepo", func(context.Context, Resolver) (indexedRepository, error) {
		return &indexedConcreteRepository{value: "interface"}, nil
	}); err != nil {
		t.Fatalf("register interface repo failed: %v", err)
	}
	if err := Register[*indexedConcreteRepository](registry, "concreteRepo", func(context.Context, Resolver) (*indexedConcreteRepository, error) {
		return &indexedConcreteRepository{value: "concrete"}, nil
	}); err != nil {
		t.Fatalf("register concrete repo failed: %v", err)
	}

	runtimeContainer, err := New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	repositoryType := reflect.TypeOf((*indexedRepository)(nil)).Elem()
	names, ok := runtimeContainer.indexedNames(repositoryType)
	if !ok {
		t.Fatalf("expected interface type to be indexed")
	}
	want := []string{"concreteRepo", "interfaceRepo"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected interface index %v, got %v", want, names)
	}
}

func TestContainerTypeIndex_whenResolvingUnregisteredInterface_shouldCacheAssignableImplementations(t *testing.T) {
	registry := NewRegistry()
	if err := Register[*indexedConcreteRepository](registry, "concreteRepo", func(context.Context, Resolver) (*indexedConcreteRepository, error) {
		return &indexedConcreteRepository{value: "concrete"}, nil
	}); err != nil {
		t.Fatalf("register concrete repo failed: %v", err)
	}
	runtimeContainer, err := New(registry)
	if err != nil {
		t.Fatalf("create container failed: %v", err)
	}

	repositoryType := reflect.TypeOf((*indexedRepository)(nil)).Elem()
	if _, ok := runtimeContainer.indexedNames(repositoryType); ok {
		t.Fatalf("interface type should not be pre-indexed before it is requested")
	}
	value, err := runtimeContainer.GetByType(context.Background(), repositoryType)
	if err != nil {
		t.Fatalf("resolve by interface failed: %v", err)
	}
	repository, ok := value.(indexedRepository)
	if !ok {
		t.Fatalf("expected indexedRepository, got %T", value)
	}
	if repository.Find() != "concrete" {
		t.Fatalf("unexpected repository value: %q", repository.Find())
	}

	names, ok := runtimeContainer.indexedNames(repositoryType)
	if !ok {
		t.Fatalf("expected interface type to be cached")
	}
	want := []string{"concreteRepo"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected cached interface index %v, got %v", want, names)
	}
}
