package context

import (
	"github.com/goark-projects/goark/config"
	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/event"
)

// Option 调整应用上下文初始化参数。
type Option func(*ApplicationContext) error

// WithEnvironment 设置应用配置环境。
func WithEnvironment(env *config.Environment) Option {
	return func(app *ApplicationContext) error {
		if env == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
		}
		app.env = env
		return nil
	}
}

// WithPropertySource 添加配置源到最低优先级。
func WithPropertySource(source config.PropertySource) Option {
	return func(app *ApplicationContext) error {
		return app.env.AddLast(source)
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
