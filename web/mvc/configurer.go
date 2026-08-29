package mvc

import (
	"context"

	goweb "goark.dev/goark/web"
)

// Configurer 将 MVC 控制器注册到 Web 注册表。
type Configurer struct {
	controllers       []Controller
	advices           []ControllerAdvice
	exceptionHandlers []goweb.ErrorMapper
}

// NewConfigurer 创建 MVC Web 配置器。
func NewConfigurer(controllers ...Controller) Configurer {
	return Configurer{controllers: append([]Controller(nil), controllers...)}
}

// WithExceptionHandlers 添加 MVC 全局异常处理器。
func (c Configurer) WithExceptionHandlers(handlers ...goweb.ErrorMapper) Configurer {
	c.exceptionHandlers = append(c.exceptionHandlers, handlers...)
	return c
}

// WithControllerAdvices 添加 MVC 全局 advice。
func (c Configurer) WithControllerAdvices(advices ...ControllerAdvice) Configurer {
	c.advices = append(c.advices, advices...)
	return c
}

// ConfigureWeb 注册控制器路由。
func (c Configurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	for _, handler := range c.exceptionHandlers {
		registry.UseErrorMapper(handler)
	}
	for _, advice := range c.advices {
		if err := advice.ConfigureWeb(ctx, registry); err != nil {
			return err
		}
	}
	return registerControllers(registry, c.controllers)
}
