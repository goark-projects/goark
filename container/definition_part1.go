package container

import (
	"context"
	"reflect"

	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/internal/reflectx"
)

// NewInstanceDefinition 创建已有实例的单例 Bean 定义。
func NewInstanceDefinition[T any](name string, instance T, options ...Option) (Definition, error) {
	if reflectx.IsNil(instance) {
		return Definition{}, arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"bean %q instance is nil",
			name,
		)
	}

	beanType := reflectx.TypeOf[T]()
	definition := Definition{
		Name:             name,
		Type:             beanType,
		Scope:            ScopeSingleton,
		LifecycleManaged: true,
	}
	definition.Factory = func(context.Context, Resolver) (any, error) {
		return normalizeInstance(definition.Name, beanType, instance)
	}

	for _, option := range options {
		if option != nil {
			option(&definition)
		}
	}
	definition = definition.normalized()
	if definition.Scope != ScopeSingleton {
		return Definition{}, arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"bean %q instance must use singleton scope",
			name,
		)
	}
	if err := validateDefinition(definition); err != nil {
		return Definition{}, err
	}
	return definition.clone(), nil
}

// WithTypedDependencyInjector 设置类型安全的依赖注入函数。
func WithTypedDependencyInjector[T any](injector func(context.Context, Resolver, T) error) Option {
	return func(def *Definition) {
		if injector == nil {
			def.DependencyInjector = nil
			return
		}
		def.DependencyInjector = func(ctx context.Context, resolver Resolver, bean any) error {
			typed, ok := bean.(T)
			if !ok {
				return arkerrors.Newf(
					arkerrors.CodeTypeMismatch,
					"bean %q injector received %T, expected %s",
					def.Name,
					bean,
					typeName[T](),
				)
			}
			return injector(ctx, resolver, typed)
		}
	}
}

func withDependencyNames(
	kind DependencyKind,
	source DependencySource,
	optional bool,
	names ...string,
) Option {
	copied := splitDependencyNames(names)
	return func(def *Definition) {
		for _, name := range copied {
			descriptor := dependencyDescriptor(name, kind, source, optional)
			def.DependencyDescriptors = appendDependencyDescriptor(
				def.DependencyDescriptors,
				descriptor,
			)
			def.Dependencies = append(def.Dependencies, descriptor.Name)
		}
	}
}

func normalizeInstance(name string, expected reflect.Type, value any) (any, error) {
	if reflectx.IsNil(value) {
		return nil, arkerrors.Newf(arkerrors.CodeCreation, "bean %q provider returned nil", name)
	}
	actual := reflect.TypeOf(value)
	if !typeAssignable(actual, expected) {
		return nil, arkerrors.Newf(
			arkerrors.CodeTypeMismatch,
			"bean %q provider returned %s, expected %s",
			name,
			actual,
			expected,
		)
	}
	return value, nil
}

// WithDependencyDescriptors 追加完整依赖描述，供生成器写入精确依赖图。
func WithDependencyDescriptors(descriptors ...DependencyDescriptor) Option {
	copied := append([]DependencyDescriptor(nil), descriptors...)
	return func(def *Definition) {
		for _, descriptor := range copied {
			def.DependencyDescriptors = appendDependencyDescriptor(
				def.DependencyDescriptors,
				descriptor,
			)
		}
	}
}

func containsDependencyName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

// Resolver 提供 Bean 解析能力，Provider 通过它显式获取依赖。
type Resolver interface {
	Get(ctx context.Context, name string) (any, error)
	GetByType(ctx context.Context, typ reflect.Type, options ...ResolveOption) (any, error)
	GetAllByType(ctx context.Context, typ reflect.Type) ([]any, error)
}

// WithLazy 设置单例 Bean 延迟初始化。
func WithLazy() Option {
	return func(def *Definition) {
		def.Lazy = true
	}
}

// WithLifecycleManaged 控制单例 Bean 是否自动参与容器生命周期。
func WithLifecycleManaged(managed bool) Option {
	return func(def *Definition) {
		def.LifecycleManaged = managed
	}
}

// WithOrder 设置 Bean 的稳定排序值，数值越小优先级越高。
func WithOrder(value int) Option {
	return func(def *Definition) {
		def.Order = value
	}
}

func (d Definition) clone() Definition {
	d.DependsOn = append([]string(nil), d.DependsOn...)
	d.Dependencies = append([]string(nil), d.Dependencies...)
	d.DependencyDescriptors = append([]DependencyDescriptor(nil), d.DependencyDescriptors...)
	return d
}

// WithPrototype 将 Bean 设置为原型作用域。
func WithPrototype() Option {
	return WithScope(ScopePrototype)
}

// WithFactoryDependencies 声明工厂方法或构造参数依赖。
func WithFactoryDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindFactory, DependencySourceInferred, false, names...)
}

// WithOptionalInjectionDependencies 声明可选字段或 setter 注入依赖。
func WithOptionalInjectionDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindInjection, DependencySourceInferred, true, names...)
}

// Provider 是类型安全的 Bean 工厂函数。
type Provider[T any] func(ctx context.Context, resolver Resolver) (T, error)

// DependencyInjector 在 Bean 实例创建后执行字段或 setter 注入。
type DependencyInjector func(ctx context.Context, resolver Resolver, bean any) error
