package web

import (
	"context"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
)

// Interceptor 表示 Goark Web 请求处理拦截器。
type Interceptor = arkweb.Interceptor

// InterceptorFunc 将普通函数适配为 Interceptor。
type InterceptorFunc = arkweb.InterceptorFunc

// RegisterInterceptor 注册 Web 拦截器贡献点。
func RegisterInterceptor(registry *container.Registry, name string, interceptor arkweb.Interceptor, options ...container.Option) error {
	if isNilInterceptor(interceptor) {
		return ErrNilInterceptor
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.Use(interceptor)
		return nil
	}), options...)
}

// RegisterMappedInterceptor 注册带路径映射的 Web 拦截器贡献点。
func RegisterMappedInterceptor(registry *container.Registry, name string, interceptor arkweb.Interceptor, mapping InterceptorMapping, options ...container.Option) error {
	if isNilInterceptor(interceptor) {
		return ErrNilInterceptor
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseMapped(interceptor, mapping)
		return nil
	}), options...)
}

func isNilInterceptor(interceptor arkweb.Interceptor) bool {
	return isNilWebValue(interceptor)
}
