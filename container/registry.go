package container

import (
	"sort"
	"sync"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Registry 保存 Bean 定义，通常由编译期生成代码写入。
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

// NewRegistry 创建空 Bean 注册表。
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]Definition),
	}
}

// Register 注册 Bean 定义。
func (r *Registry) Register(definition Definition) error {
	if r == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bean registry is nil")
	}
	definition = definition.normalized()
	if err := validateDefinition(definition); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[definition.Name]; exists {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "bean %q already exists", definition.Name)
	}
	r.definitions[definition.Name] = definition.clone()
	return nil
}

// Definition 返回指定名称的 Bean 定义副本。
func (r *Registry) Definition(name string) (Definition, bool) {
	if r == nil || name == "" {
		return Definition{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, ok := r.definitions[name]
	return definition.clone(), ok
}

// Definitions 返回所有 Bean 定义副本，按名称排序。
func (r *Registry) Definitions() []Definition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	definitions := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		definitions = append(definitions, definition.clone())
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Name < definitions[j].Name
	})
	return definitions
}

// Register 注册类型安全的 Bean 工厂。
func Register[T any](r *Registry, name string, provider Provider[T], options ...Option) error {
	definition, err := NewDefinition(name, provider, options...)
	if err != nil {
		return err
	}
	return r.Register(definition)
}

// RegisterInstance 注册已有实例。
func RegisterInstance[T any](r *Registry, name string, instance T, options ...Option) error {
	definition, err := NewInstanceDefinition(name, instance, options...)
	if err != nil {
		return err
	}
	return r.Register(definition)
}
