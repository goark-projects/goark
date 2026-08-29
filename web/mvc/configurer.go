package mvc

import (
	"context"

	"goark.dev/goark/core/util"
	goweb "goark.dev/goark/web"
)

type mappedHandlerInterceptor struct {
	interceptor HandlerInterceptor
	mapping     goweb.InterceptorMapping
}

// Configurer 将 MVC 控制器注册到 Web 注册表。
type Configurer struct {
	controllers               []Controller
	advices                   []ControllerAdvice
	exceptionHandlers         []goweb.ErrorMapper
	handlerInterceptors       []HandlerInterceptor
	mappedHandlerInterceptors []mappedHandlerInterceptor
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

// WithHandlerInterceptors 添加 MVC 全局处理器拦截器。
func (c Configurer) WithHandlerInterceptors(interceptors ...HandlerInterceptor) Configurer {
	c.handlerInterceptors = append(c.handlerInterceptors, interceptors...)
	return c
}

// WithMappedHandlerInterceptor 添加带路径映射的 MVC 处理器拦截器。
func (c Configurer) WithMappedHandlerInterceptor(interceptor HandlerInterceptor, mapping goweb.InterceptorMapping) Configurer {
	c.mappedHandlerInterceptors = append(c.mappedHandlerInterceptors, mappedHandlerInterceptor{
		interceptor: interceptor,
		mapping:     mapping,
	})
	return c
}

func (c Configurer) withMappedHandlerInterceptors(interceptors ...mappedHandlerInterceptor) Configurer {
	c.mappedHandlerInterceptors = append(c.mappedHandlerInterceptors, interceptors...)
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
	for _, interceptor := range c.handlerInterceptors {
		if util.IsNil(interceptor) {
			continue
		}
		registry.Use(HandlerInterceptorAdapter(interceptor))
	}
	for _, registration := range c.mappedHandlerInterceptors {
		if util.IsNil(registration.interceptor) {
			continue
		}
		registry.UseMapped(HandlerInterceptorAdapter(registration.interceptor), registration.mapping)
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
