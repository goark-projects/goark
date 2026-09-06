package container

import (
	"context"
	"reflect"
	"strings"

	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/internal/reflectx"
)

// NewDefinition 创建 Bean 定义，供编译期生成代码直接调用。
func NewDefinition[T any](
	name string,
	provider Provider[T],
	options ...Option,
) (Definition, error) {
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
		return arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"bean %q has invalid scope %q",
			def.Name,
			def.Scope,
		)
	}
	for _, dependency := range def.DependencyDescriptors {
		if strings.TrimSpace(dependency.Name) == "" {
			return arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"bean %q has empty dependency name",
				def.Name,
			)
		}
	}
	return nil
}

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

func (d Definition) normalized() Definition {
	d.Name = strings.TrimSpace(d.Name)
	descriptors, dependsOn, dependencies := normalizeDefinitionDependencies(
		d.DependencyDescriptors,
		d.DependsOn,
		d.Dependencies,
	)
	d.DependencyDescriptors = append([]DependencyDescriptor(nil), descriptors...)
	d.DependsOn = append([]string(nil), dependsOn...)
	d.Dependencies = append([]string(nil), dependencies...)
	return d
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

// WithScope 设置 Bean 作用域。
func WithScope(scope Scope) Option {
	return func(def *Definition) {
		def.Scope = scope
	}
}

// WithPrimary 将 Bean 标记为同类型解析时的首选项。
func WithPrimary() Option {
	return func(def *Definition) {
		def.Primary = true
	}
}

// WithDependencyInjector 设置 Bean 实例创建后的依赖注入函数。
func WithDependencyInjector(injector DependencyInjector) Option {
	return func(def *Definition) {
		def.DependencyInjector = injector
	}
}

// WithPriority 设置 Bean 的候选优先级，数值越小优先级越高。
func WithPriority(value int) Option {
	return func(def *Definition) {
		def.Priority = lang.Some(value)
	}
}

const (
	ScopeSingleton Scope = "singleton"
	ScopePrototype Scope = "prototype"
)

// WithSingleton 将 Bean 设置为单例作用域。
func WithSingleton() Option {
	return WithScope(ScopeSingleton)
}

// WithDependencies 声明生成器推导出的工厂依赖名称，用于校验与拓扑分析。
func WithDependencies(names ...string) Option {
	return WithFactoryDependencies(names...)
}

// WithInjectionDependencies 声明字段或 setter 注入依赖。
func WithInjectionDependencies(names ...string) Option {
	return withDependencyNames(DependencyKindInjection, DependencySourceInferred, false, names...)
}

// Scope 表示 Bean 实例生命周期范围。
type Scope string

// Factory 是容器内部使用的非泛型工厂函数。
type Factory func(ctx context.Context, resolver Resolver) (any, error)

// Option 调整 Bean 定义元数据。
type Option func(*Definition)
