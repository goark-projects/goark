package goark

import (
	"context"
	"reflect"

	"goark.dev/goark/container"
	appcontext "goark.dev/goark/context"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/expression"
)

// ApplicationContext 是 Goark 应用上下文类型别名。
type ApplicationContext = appcontext.ApplicationContext

// Option 是应用上下文初始化选项。
type Option = appcontext.Option

// Environment 是 Goark 核心配置环境类型别名。
type Environment = coreenv.Environment

// ConfigurableEnvironment 是可配置环境类型别名。
type ConfigurableEnvironment = coreenv.ConfigurableEnvironment

// PropertySource 是配置源类型别名。
type PropertySource = coreenv.PropertySource

// ConfigPropertySource 是文件配置源类型别名。
type ConfigPropertySource = coreenv.ConfigPropertySource

// ConfigFormat 是配置文件格式类型别名。
type ConfigFormat = coreenv.ConfigFormat

// Expression 是已解析的 GaEL 表达式。
type Expression = expression.Expression

// ExpressionParser 定义 GaEL 解析契约。
type ExpressionParser = expression.Parser

// EvaluationContext 定义 GaEL 求值上下文。
type EvaluationContext = expression.EvaluationContext

// ExpressionFunction 是显式注册的 GaEL 安全函数。
type ExpressionFunction = expression.Function

// EvaluationContextOption 配置 GaEL 默认求值上下文。
type EvaluationContextOption = expression.ContextOption

// BeanDefinition 是 Bean 定义类型别名。
type BeanDefinition = container.Definition

// BeanOption 是 Bean 注册选项类型别名。
type BeanOption = container.Option

// BeanScope 是 Bean 作用域类型别名。
type BeanScope = container.Scope

// DependencyDescriptor 是 Bean 依赖描述类型别名。
type DependencyDescriptor = container.DependencyDescriptor

// DependencyKind 是依赖边语义类型别名。
type DependencyKind = container.DependencyKind

// DependencySource 是依赖来源类型别名。
type DependencySource = container.DependencySource

// ResolveOption 是按类型解析选项类型别名。
type ResolveOption = container.ResolveOption

// Resolver 是 Bean 解析器类型别名。
type Resolver = container.Resolver

// Provider 是 Bean 工厂函数类型别名。
type Provider[T any] = container.Provider[T]

// DependencyInjector 是 Bean 依赖注入函数类型别名。
type DependencyInjector = container.DependencyInjector

const (
	// DefaultConfigBaseName 是默认配置文件基础名称。
	DefaultConfigBaseName = coreenv.DefaultConfigBaseName

	// ConfigFormatYAML 表示 YAML 配置格式。
	ConfigFormatYAML = coreenv.ConfigFormatYAML
	// ConfigFormatProperties 表示 properties 配置格式。
	ConfigFormatProperties = coreenv.ConfigFormatProperties
	// ConfigFormatTOML 表示 TOML 配置格式。
	ConfigFormatTOML = coreenv.ConfigFormatTOML

	// ScopeSingleton 表示单例 Bean 作用域。
	ScopeSingleton = container.ScopeSingleton
	// ScopePrototype 表示原型 Bean 作用域。
	ScopePrototype = container.ScopePrototype

	// DependencyKindFactory 表示工厂方法或构造参数依赖。
	DependencyKindFactory = container.DependencyKindFactory
	// DependencyKindInjection 表示字段或 setter 注入依赖。
	DependencyKindInjection = container.DependencyKindInjection
	// DependencyKindDependsOn 表示手工 depends-on 初始化顺序依赖。
	DependencyKindDependsOn = container.DependencyKindDependsOn

	// DependencySourceInferred 表示由生成器自动推导出的依赖。
	DependencySourceInferred = container.DependencySourceInferred
	// DependencySourceManual 表示由用户显式声明的依赖。
	DependencySourceManual = container.DependencySourceManual
)

// Configuration 是应用配置单元类型别名。
type Configuration = appcontext.Configuration

// ConfigurationDescriptor 是应用配置单元只读描述类型别名。
type ConfigurationDescriptor = appcontext.ConfigurationDescriptor

// ConfigurationContext 是配置注册期上下文类型别名。
type ConfigurationContext = appcontext.ConfigurationContext

// ContextAwareConfiguration 是可访问配置上下文的配置单元类型别名。
type ContextAwareConfiguration = appcontext.ContextAwareConfiguration

// AnnotationMetadata 是注解元数据类型别名。
type AnnotationMetadata = appcontext.AnnotationMetadata

// Condition 是条件装配接口类型别名。
type Condition = appcontext.Condition

// ConditionFunc 是条件函数适配器类型别名。
type ConditionFunc = appcontext.ConditionFunc

// ProfileCondition 是 profile 表达式条件类型别名。
type ProfileCondition = appcontext.ProfileCondition

// New 创建应用上下文。
func New(options ...Option) (*ApplicationContext, error) {
	return appcontext.New(options...)
}

// MustNew 创建应用上下文，失败时 panic。
func MustNew(options ...Option) *ApplicationContext {
	app, err := New(options...)
	if err != nil {
		panic(err)
	}
	return app
}

// WithEnvironment 设置应用配置环境。
var WithEnvironment = appcontext.WithEnvironment

// WithPropertySource 添加配置源。
var WithPropertySource = appcontext.WithPropertySource

// WithPropertySourceName 指定加载后的 PropertySource 名称。
var WithPropertySourceName = coreenv.WithPropertySourceName

// WithPropertySourceEncoding 指定配置文本编码。
var WithPropertySourceEncoding = coreenv.WithPropertySourceEncoding

// WithIgnoreResourceNotFound 设置资源不存在时是否忽略。
var WithIgnoreResourceNotFound = coreenv.WithIgnoreResourceNotFound

// LoadConfigPropertySource 从资源位置加载 yml/properties/toml 配置源。
var LoadConfigPropertySource = coreenv.LoadConfigPropertySource

// LoadDefaultConfigPropertySource 按默认名称加载 app.yml/app.properties/app.toml/app.yaml 配置源。
var LoadDefaultConfigPropertySource = coreenv.LoadDefaultConfigPropertySource

// LoadPropertiesPropertySource 从资源位置加载 .properties 配置源。
var LoadPropertiesPropertySource = coreenv.LoadPropertiesPropertySource

// NewExpressionParser 创建 GaEL 默认解析器。
var NewExpressionParser = expression.NewParser

// NewEvaluationContext 创建 GaEL 默认求值上下文。
var NewEvaluationContext = expression.NewEvaluationContext

// WithExpressionVariable 注册 GaEL 只读变量。
var WithExpressionVariable = expression.WithVariable

// WithExpressionFunction 注册 GaEL 白名单函数。
var WithExpressionFunction = expression.WithFunction

// WithConfiguration 注册应用配置单元。
var WithConfiguration = appcontext.WithConfiguration

// WithEventBus 设置事件总线。
var WithEventBus = appcontext.WithEventBus

// WithAllowCircularReferences 设置是否允许单例字段注入循环依赖。
var WithAllowCircularReferences = appcontext.WithAllowCircularReferences

// WithScope 设置 Bean 作用域。
var WithScope = container.WithScope

// WithSingleton 将 Bean 设置为单例作用域。
var WithSingleton = container.WithSingleton

// WithPrototype 将 Bean 设置为原型作用域。
var WithPrototype = container.WithPrototype

// WithLazy 设置单例 Bean 延迟初始化。
var WithLazy = container.WithLazy

// WithPrimary 将 Bean 标记为同类型解析时的首选项。
var WithPrimary = container.WithPrimary

// WithDependsOn 声明当前 Bean 初始化前必须先初始化的 Bean 名称。
var WithDependsOn = container.WithDependsOn

// WithDependencies 声明 Bean 的工厂依赖名称。
var WithDependencies = container.WithDependencies

// WithFactoryDependencies 声明 Bean 的工厂依赖名称。
var WithFactoryDependencies = container.WithFactoryDependencies

// WithInjectionDependencies 声明 Bean 的字段或 setter 注入依赖名称。
var WithInjectionDependencies = container.WithInjectionDependencies

// WithOptionalInjectionDependencies 声明 Bean 的可选字段或 setter 注入依赖名称。
var WithOptionalInjectionDependencies = container.WithOptionalInjectionDependencies

// WithDependencyDescriptors 追加完整依赖描述。
var WithDependencyDescriptors = container.WithDependencyDescriptors

// WithDependencyInjector 设置 Bean 实例创建后的依赖注入函数。
var WithDependencyInjector = container.WithDependencyInjector

// WithOrder 设置 Bean 的稳定排序值。
var WithOrder = container.WithOrder

// WithPriority 设置 Bean 的候选优先级。
var WithPriority = container.WithPriority

// WithQualifier 指定按类型解析时优先使用的 Bean 名称。
var WithQualifier = container.WithQualifier

// WithTypedDependencyInjector 设置类型安全的 Bean 依赖注入函数。
func WithTypedDependencyInjector[T any](injector func(context.Context, Resolver, T) error) BeanOption {
	return container.WithTypedDependencyInjector(injector)
}

// NewConfigurationContext 创建配置注册上下文。
var NewConfigurationContext = appcontext.NewConfigurationContext

// ResolveValue 解析 value 表达式并转换为目标类型。
func ResolveValue(environment Environment, expression string, targetType reflect.Type) (any, error) {
	return coreenv.ResolveValue(environment, expression, targetType)
}

// ResolveValueAs 按泛型目标类型解析 value 表达式。
func ResolveValueAs[T any](environment Environment, expression string) (T, error) {
	return coreenv.ResolveValueAs[T](environment, expression)
}

// Register 注册类型安全 Bean 工厂。
func Register[T any](app *ApplicationContext, name string, provider Provider[T], options ...BeanOption) error {
	if app == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	definition, err := container.NewDefinition(name, provider, options...)
	if err != nil {
		return err
	}
	return app.RegisterDefinition(definition)
}

// RegisterInstance 注册已有实例。
func RegisterInstance[T any](app *ApplicationContext, name string, instance T, options ...BeanOption) error {
	if app == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	definition, err := container.NewInstanceDefinition(name, instance, options...)
	if err != nil {
		return err
	}
	return app.RegisterDefinition(definition)
}

// RegisterConfiguration 注册应用配置单元。
func RegisterConfiguration(app *ApplicationContext, configuration Configuration) error {
	if app == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	return app.RegisterConfiguration(configuration)
}

// Get 按名称解析 Bean。
func Get[T any](ctx context.Context, resolver Resolver, name string) (T, error) {
	return container.Get[T](ctx, resolver, name)
}

// GetByType 按类型解析 Bean。
func GetByType[T any](ctx context.Context, resolver Resolver, options ...ResolveOption) (T, error) {
	return container.GetByType[T](ctx, resolver, options...)
}

// GetAllByType 按类型解析全部 Bean。
func GetAllByType[T any](ctx context.Context, resolver Resolver) ([]T, error) {
	return container.GetAllByType[T](ctx, resolver)
}

// MustGet 是 Get 的 panic 版本。
func MustGet[T any](ctx context.Context, resolver Resolver, name string) T {
	return container.MustGet[T](ctx, resolver, name)
}

// MustGetByType 是 GetByType 的 panic 版本。
func MustGetByType[T any](ctx context.Context, resolver Resolver, options ...ResolveOption) T {
	return container.MustGetByType[T](ctx, resolver, options...)
}
