package mvc

import (
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
)

// RegisterHandlerInterceptor 注册 MVC HandlerInterceptor 配置贡献点。
func RegisterHandlerInterceptor(registry *container.Registry, name string, interceptor HandlerInterceptor, options ...container.Option) error {
	return goweb.RegisterInterceptor(registry, name, HandlerInterceptorAdapter(interceptor), options...)
}

// RegisterMappedHandlerInterceptor 注册带路径映射的 MVC HandlerInterceptor 配置贡献点。
func RegisterMappedHandlerInterceptor(
	registry *container.Registry,
	name string,
	interceptor HandlerInterceptor,
	mapping goweb.InterceptorMapping,
	options ...container.Option,
) error {
	return goweb.RegisterMappedInterceptor(registry, name, HandlerInterceptorAdapter(interceptor), mapping, options...)
}
