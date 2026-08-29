package mvc

import (
	"context"

	"goark.dev/goark/container"
	appcontext "goark.dev/goark/context"
	goweb "goark.dev/goark/web"
)

// Configuration 将 MVC 控制器注册为 Goark 配置单元。
type Configuration struct {
	name              string
	order             int
	controllers       []Controller
	advices           []ControllerAdvice
	exceptionHandlers []goweb.ErrorMapper
}

// NewConfiguration 创建 MVC 配置单元。
func NewConfiguration(name string, controllers ...Controller) Configuration {
	return Configuration{
		name:        name,
		controllers: append([]Controller(nil), controllers...),
	}
}

// WithOrder 设置配置单元顺序。
func (c Configuration) WithOrder(order int) Configuration {
	c.order = order
	return c
}

// WithExceptionHandlers 添加 MVC 全局异常处理器。
func (c Configuration) WithExceptionHandlers(handlers ...goweb.ErrorMapper) Configuration {
	c.exceptionHandlers = append(c.exceptionHandlers, handlers...)
	return c
}

// WithControllerAdvices 添加 MVC 全局 advice。
func (c Configuration) WithControllerAdvices(advices ...ControllerAdvice) Configuration {
	c.advices = append(c.advices, advices...)
	return c
}

// Name 返回配置名称。
func (c Configuration) Name() string {
	if c.name == "" {
		return "goark.web.mvc"
	}
	return c.name
}

// Order 返回配置顺序。
func (c Configuration) Order() int {
	return c.order
}

// Register 注册 MVC Web 配置器 Bean。
func (c Configuration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, appcontext.NewConfigurationContext(nil, registry))
}

// RegisterWithContext 注册 MVC Web 配置器 Bean。
func (c Configuration) RegisterWithContext(_ context.Context, config appcontext.ConfigurationContext) error {
	return goweb.RegisterConfigurer(
		config.Registry(),
		c.Name()+".configurer",
		NewConfigurer(c.controllers...).WithExceptionHandlers(c.exceptionHandlers...).WithControllerAdvices(c.advices...),
		container.WithOrder(c.order),
	)
}
