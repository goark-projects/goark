package goark

import (
	"context"
	"reflect"

	"github.com/goark-projects/goark/container"
	appcontext "github.com/goark-projects/goark/context"
	coreenv "github.com/goark-projects/goark/core/env"
	arkerrors "github.com/goark-projects/goark/errors"
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

const (
	// DefaultConfigBaseName 是默认配置文件基础名称。
	DefaultConfigBaseName = coreenv.DefaultConfigBaseName

	// ConfigFormatYAML 表示 YAML 配置格式。
	ConfigFormatYAML = coreenv.ConfigFormatYAML
	// ConfigFormatProperties 表示 properties 配置格式。
	ConfigFormatProperties = coreenv.ConfigFormatProperties
	// ConfigFormatTOML 表示 TOML 配置格式。
	ConfigFormatTOML = coreenv.ConfigFormatTOML
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

// LoadDefaultConfigPropertySource 按默认名称加载 app.yml/app.yaml/app.properties/app.toml 配置源。
var LoadDefaultConfigPropertySource = coreenv.LoadDefaultConfigPropertySource

// LoadPropertiesPropertySource 从资源位置加载 .properties 配置源。
var LoadPropertiesPropertySource = coreenv.LoadPropertiesPropertySource

// WithConfiguration 注册应用配置单元。
var WithConfiguration = appcontext.WithConfiguration

// WithEventBus 设置事件总线。
var WithEventBus = appcontext.WithEventBus

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
func Register[T any](app *ApplicationContext, name string, provider container.Provider[T], options ...container.Option) error {
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
func RegisterInstance[T any](app *ApplicationContext, name string, instance T, options ...container.Option) error {
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
func Get[T any](ctx context.Context, resolver container.Resolver, name string) (T, error) {
	return container.Get[T](ctx, resolver, name)
}

// GetByType 按类型解析 Bean。
func GetByType[T any](ctx context.Context, resolver container.Resolver, options ...container.ResolveOption) (T, error) {
	return container.GetByType[T](ctx, resolver, options...)
}

// MustGet 是 Get 的 panic 版本。
func MustGet[T any](ctx context.Context, resolver container.Resolver, name string) T {
	return container.MustGet[T](ctx, resolver, name)
}

// MustGetByType 是 GetByType 的 panic 版本。
func MustGetByType[T any](ctx context.Context, resolver container.Resolver, options ...container.ResolveOption) T {
	return container.MustGetByType[T](ctx, resolver, options...)
}
