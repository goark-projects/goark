package container

import (
	"context"
	"reflect"

	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/internal/reflectx"
)

// Scope 表示 Bean 实例生命周期范围。
type Scope string

const (
	ScopeSingleton Scope = "singleton"
	ScopePrototype Scope = "prototype"
)

// Resolver 提供 Bean 解析能力，Provider 通过它显式获取依赖。
type Resolver interface {
	Get(ctx context.Context, name string) (any, error)
	GetByType(ctx context.Context, typ reflect.Type) (any, error)
}

// Provider 是类型安全的 Bean 工厂函数。
type Provider[T any] func(ctx context.Context, resolver Resolver) (T, error)

// Factory 是容器内部使用的非泛型工厂函数。
type Factory func(ctx context.Context, resolver Resolver) (any, error)

// Definition 描述一个可被容器管理的 Bean。
type Definition struct {
	Name         string
	Type         reflect.Type
	Scope        Scope
	Lazy         bool
	Primary      bool
	Dependencies []string
	Factory      Factory
}

// Option 调整 Bean 定义元数据。
type Option func(*Definition)

// WithScope 设置 Bean 作用域。
func WithScope(scope Scope) Option {
	return func(def *Definition) {
		def.Scope = scope
	}
}

// WithSingleton 将 Bean 设置为单例作用域。
func WithSingleton() Option {
	return WithScope(ScopeSingleton)
}

// WithPrototype 将 Bean 设置为原型作用域。
func WithPrototype() Option {
	return WithScope(ScopePrototype)
}

// WithLazy 设置单例 Bean 延迟初始化。
func WithLazy() Option {
	return func(def *Definition) {
		def.Lazy = true
	}
}

// WithPrimary 将 Bean 标记为同类型解析时的首选项。
func WithPrimary() Option {
	return func(def *Definition) {
		def.Primary = true
	}
}

// WithDependencies 声明 Bean 的显式依赖名称，用于提前校验与拓扑分析。
func WithDependencies(names ...string) Option {
	copied := append([]string(nil), names...)
	return func(def *Definition) {
		def.Dependencies = append(def.Dependencies, copied...)
	}
}

// NewDefinition 创建 Bean 定义，供编译期生成代码直接调用。
func NewDefinition[T any](name string, provider Provider[T], options ...Option) (Definition, error) {
	if provider == nil {
		return Definition{}, arkerrors.New(arkerrors.CodeInvalidArgument, "bean provider is nil")
	}

	beanType := reflectx.TypeOf[T]()
	definition := Definition{
		Name:  name,
		Type:  beanType,
		Scope: ScopeSingleton,
	}
	definition.Factory = func(ctx context.Context, resolver Resolver) (any, error) {
		value, err := provider(ctx, resolver)
		if err != nil {
			return nil, err
		}
		return normalizeInstance(definition.Name, beanType, value)
	}

	for _, option := range options {
		if option != nil {
			option(&definition)
		}
	}
	if err := validateDefinition(definition); err != nil {
		return Definition{}, err
	}
	return definition.clone(), nil
}

// NewInstanceDefinition 创建已有实例的单例 Bean 定义。
func NewInstanceDefinition[T any](name string, instance T, options ...Option) (Definition, error) {
	if reflectx.IsNil(instance) {
		return Definition{}, arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q instance is nil", name)
	}

	beanType := reflectx.TypeOf[T]()
	definition := Definition{
		Name:  name,
		Type:  beanType,
		Scope: ScopeSingleton,
	}
	definition.Factory = func(context.Context, Resolver) (any, error) {
		return normalizeInstance(definition.Name, beanType, instance)
	}

	for _, option := range options {
		if option != nil {
			option(&definition)
		}
	}
	if definition.Scope != ScopeSingleton {
		return Definition{}, arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q instance must use singleton scope", name)
	}
	if err := validateDefinition(definition); err != nil {
		return Definition{}, err
	}
	return definition.clone(), nil
}

func (d Definition) clone() Definition {
	d.Dependencies = append([]string(nil), d.Dependencies...)
	return d
}

func validateDefinition(def Definition) error {
	if def.Name == "" {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bean name is empty")
	}
	if def.Type == nil {
		return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q type is nil", def.Name)
	}
	if def.Factory == nil {
		return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q factory is nil", def.Name)
	}
	switch def.Scope {
	case ScopeSingleton, ScopePrototype:
	default:
		return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q has invalid scope %q", def.Name, def.Scope)
	}
	for _, dependency := range def.Dependencies {
		if dependency == "" {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q has empty dependency name", def.Name)
		}
	}
	return nil
}

func normalizeInstance(name string, expected reflect.Type, value any) (any, error) {
	if reflectx.IsNil(value) {
		return nil, arkerrors.Newf(arkerrors.CodeCreation, "bean %q provider returned nil", name)
	}
	actual := reflect.TypeOf(value)
	if !typeAssignable(actual, expected) {
		return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean %q provider returned %s, expected %s", name, actual, expected)
	}
	return value, nil
}

func typeAssignable(actual reflect.Type, expected reflect.Type) bool {
	if actual == nil || expected == nil {
		return false
	}
	if actual.AssignableTo(expected) {
		return true
	}
	return expected.Kind() == reflect.Interface && actual.Implements(expected)
}
