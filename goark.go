package goark

import (
	"context"

	"github.com/goark-projects/goark/container"
	appcontext "github.com/goark-projects/goark/context"
	arkerrors "github.com/goark-projects/goark/errors"
)

// ApplicationContext 是 Goark 应用上下文类型别名。
type ApplicationContext = appcontext.ApplicationContext

// Option 是应用上下文初始化选项。
type Option = appcontext.Option

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

// WithEventBus 设置事件总线。
var WithEventBus = appcontext.WithEventBus

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

// Get 按名称解析 Bean。
func Get[T any](ctx context.Context, resolver container.Resolver, name string) (T, error) {
	return container.Get[T](ctx, resolver, name)
}

// GetByType 按类型解析 Bean。
func GetByType[T any](ctx context.Context, resolver container.Resolver) (T, error) {
	return container.GetByType[T](ctx, resolver)
}

// MustGet 是 Get 的 panic 版本。
func MustGet[T any](ctx context.Context, resolver container.Resolver, name string) T {
	return container.MustGet[T](ctx, resolver, name)
}

// MustGetByType 是 GetByType 的 panic 版本。
func MustGetByType[T any](ctx context.Context, resolver container.Resolver) T {
	return container.MustGetByType[T](ctx, resolver)
}
