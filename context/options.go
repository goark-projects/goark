package context

import (
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/event"
)

// Option 调整应用上下文初始化参数。
type Option func(*ApplicationContext) error

// WithEnvironment 设置应用配置环境。
func WithEnvironment(env coreenv.ConfigurableEnvironment) Option {
	return func(app *ApplicationContext) error {
		if env == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
		}
		app.env = env
		return nil
	}
}

// WithPropertySource 添加配置源到最低优先级。
func WithPropertySource(source coreenv.PropertySource) Option {
	return func(app *ApplicationContext) error {
		return app.env.PropertySources().AddLast(source)
	}
}

// WithConfiguration 注册应用配置单元。
func WithConfiguration(configuration Configuration) Option {
	return func(app *ApplicationContext) error {
		return app.registerConfigurationLocked(configuration)
	}
}

// WithEventBus 设置事件总线。
func WithEventBus(bus *event.Bus) Option {
	return func(app *ApplicationContext) error {
		if bus == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "event bus is nil")
		}
		app.events = bus
		return nil
	}
}

// WithAllowCircularReferences 设置是否允许单例字段注入循环依赖。
func WithAllowCircularReferences(allow bool) Option {
	return func(app *ApplicationContext) error {
		app.allowCircularReferences = allow
		return nil
	}
}
