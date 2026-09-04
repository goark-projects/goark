package container

import (
	"context"
	"reflect"
	"strings"

	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/internal/reflectx"
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
	GetByType(ctx context.Context, typ reflect.Type, options ...ResolveOption) (any, error)
	GetAllByType(ctx context.Context, typ reflect.Type) ([]any, error)
}

// Provider 是类型安全的 Bean 工厂函数。
type Provider[T any] func(ctx context.Context, resolver Resolver) (T, error)

// Factory 是容器内部使用的非泛型工厂函数。
type Factory func(ctx context.Context, resolver Resolver) (any, error)

// DependencyInjector 在 Bean 实例创建后执行字段或 setter 注入。
type DependencyInjector func(ctx context.Context, resolver Resolver, bean any) error

// Definition 描述一个可被容器管理的 Bean。
type Definition struct {
	Name                  string
	Type                  reflect.Type
	Scope                 Scope
	Lazy                  bool
	Primary               bool
	DependsOn             []string
	Order                 int
	Priority              lang.Optional[int]
	Dependencies          []string
	DependencyDescriptors []DependencyDescriptor
	Factory               Factory
	DependencyInjector    DependencyInjector
	LifecycleManaged      bool
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

// WithLifecycleManaged 控制单例 Bean 是否自动参与容器生命周期。
func WithLifecycleManaged(managed bool) Option {
	return func(def *Definition) {
		def.LifecycleManaged = managed
	}
}

// WithDependsOn 声明当前 Bean 初始化前必须先初始化的 Bean 名称。
func WithDependsOn(names ...string) Option {
	copied := splitDependencyNames(names)
	return func(def *Definition) {
		def.DependsOn = append(def.DependsOn, copied...)
		for _, name := range copied {
			def.DependencyDescriptors = appendDependencyDescriptor(
				def.DependencyDescriptors,
				dependencyDescriptor(name, DependencyKindDependsOn, DependencySourceManual, false),
			)
		}
	}
}

// WithDependencies 声明生成器推导出的工厂依赖名称，用于校验与拓扑分析。
func WithDependencies(names ...string) Option {
	return WithFactoryDependencies(names...)
}

// WithFactoryDependencies 声明工厂方法或构造参数依赖。
func WithFactoryDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindFactory, DependencySourceInferred, false, names...)
}

// WithInjectionDependencies 声明字段或 setter 注入依赖。
func WithInjectionDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindInjection, DependencySourceInferred, false, names...)
}

// WithOptionalInjectionDependencies 声明可选字段或 setter 注入依赖。
func WithOptionalInjectionDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindInjection, DependencySourceInferred, true, names...)
}

// WithDependencyDescriptors 追加完整依赖描述，供生成器写入精确依赖图。
func WithDependencyDescriptors(descriptors ...DependencyDescriptor) Option {
	copied := append([]DependencyDescriptor(nil), descriptors...)
	return func(def *Definition) {
		for _, descriptor := range copied {
			def.DependencyDescriptors = appendDependencyDescriptor(def.DependencyDescriptors, descriptor)
		}
	}
}

// WithDependencyInjector 设置 Bean 实例创建后的依赖注入函数。
func WithDependencyInjector(injector DependencyInjector) Option {
	return func(def *Definition) {
		def.DependencyInjector = injector
	}
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
				return arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean %q injector received %T, expected %s", def.Name, bean, typeName[T]())
			}
			return injector(ctx, resolver, typed)
		}
	}
}

func withDependencyNames(kind DependencyKind, source DependencySource, optional bool, names ...string) Option {
	copied := splitDependencyNames(names)
	return func(def *Definition) {
		for _, name := range copied {
			descriptor := dependencyDescriptor(name, kind, source, optional)
			def.DependencyDescriptors = appendDependencyDescriptor(def.DependencyDescriptors, descriptor)
			def.Dependencies = append(def.Dependencies, descriptor.Name)
		}
	}
}

// WithOrder 设置 Bean 的稳定排序值，数值越小优先级越高。
func WithOrder(value int) Option {
	return func(def *Definition) {
		def.Order = value
	}
}

// WithPriority 设置 Bean 的候选优先级，数值越小优先级越高。
func WithPriority(value int) Option {
	return func(def *Definition) {
		def.Priority = lang.Some(value)
	}
}

// NewDefinition 创建 Bean 定义，供编译期生成代码直接调用。
func NewDefinition[T any](name string, provider Provider[T], options ...Option) (Definition, error) {
	if provider == nil {
		return Definition{}, arkerrors.New(arkerrors.CodeInvalidArgument, "bean provider is nil")
	}

	beanType := reflectx.TypeOf[T]()
	definition := Definition{
		Name:             name,
		Type:             beanType,
		Scope:            ScopeSingleton,
		LifecycleManaged: true,
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
	definition = definition.normalized()
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
		return Definition{}, arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q instance must use singleton scope", name)
	}
	if err := validateDefinition(definition); err != nil {
		return Definition{}, err
	}
	return definition.clone(), nil
}

func (d Definition) normalized() Definition {
	d.Name = strings.TrimSpace(d.Name)
	descriptors, dependsOn, dependencies := normalizeDefinitionDependencies(d.DependencyDescriptors, d.DependsOn, d.Dependencies)
	d.DependencyDescriptors = append([]DependencyDescriptor(nil), descriptors...)
	d.DependsOn = append([]string(nil), dependsOn...)
	d.Dependencies = append([]string(nil), dependencies...)
	return d
}

func (d Definition) clone() Definition {
	d.DependsOn = append([]string(nil), d.DependsOn...)
	d.Dependencies = append([]string(nil), d.Dependencies...)
	d.DependencyDescriptors = append([]DependencyDescriptor(nil), d.DependencyDescriptors...)
	return d
}

func validateDefinition(def Definition) error {
	if strings.TrimSpace(def.Name) == "" {
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
	for _, dependency := range def.DependencyDescriptors {
		if strings.TrimSpace(dependency.Name) == "" {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q has empty dependency name", def.Name)
		}
	}
	return nil
}

func containsDependencyName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
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
