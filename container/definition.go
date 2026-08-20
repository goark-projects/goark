package container

import (
	"context"
	"reflect"
	"strings"

	"github.com/goark-projects/goark/core/lang"
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
	GetByType(ctx context.Context, typ reflect.Type, options ...ResolveOption) (any, error)
}

// Provider 是类型安全的 Bean 工厂函数。
type Provider[T any] func(ctx context.Context, resolver Resolver) (T, error)

// Factory 是容器内部使用的非泛型工厂函数。
type Factory func(ctx context.Context, resolver Resolver) (any, error)

// Definition 描述一个可被容器管理的 Bean。
type Definition struct {
	Name      string
	Type      reflect.Type
	Scope     Scope
	Lazy      bool
	Primary   bool
	DependsOn []string
	Order     int
	Priority  lang.Optional[int]
	// Dependencies 是 DependsOn 的兼容别名，保留给早期显式注册代码使用。
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

// WithDependsOn 声明当前 Bean 初始化前必须先初始化的 Bean 名称。
func WithDependsOn(names ...string) Option {
	copied := append([]string(nil), names...)
	return func(def *Definition) {
		def.DependsOn = append(def.DependsOn, copied...)
		def.Dependencies = append(def.Dependencies, copied...)
	}
}

// WithDependencies 声明 Bean 的显式依赖名称，用于提前校验与拓扑分析。
func WithDependencies(names ...string) Option {
	return WithDependsOn(names...)
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
	dependsOn := mergeDependsOn(d.DependsOn, d.Dependencies)
	d.DependsOn = append([]string(nil), dependsOn...)
	d.Dependencies = append([]string(nil), dependsOn...)
	return d
}

func (d Definition) clone() Definition {
	d.DependsOn = append([]string(nil), d.DependsOn...)
	d.Dependencies = append([]string(nil), d.Dependencies...)
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
	for _, dependency := range def.DependsOn {
		if strings.TrimSpace(dependency) == "" {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "bean %q has empty dependency name", def.Name)
		}
	}
	return nil
}

func mergeDependsOn(dependsOn []string, dependencies []string) []string {
	merged := make([]string, 0, len(dependsOn)+len(dependencies))
	for _, dependency := range dependsOn {
		dependency = strings.TrimSpace(dependency)
		if !containsDependencyName(merged, dependency) {
			merged = append(merged, dependency)
		}
	}
	for _, dependency := range dependencies {
		dependency = strings.TrimSpace(dependency)
		if !containsDependencyName(merged, dependency) {
			merged = append(merged, dependency)
		}
	}
	return merged
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
