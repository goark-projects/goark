package view

import (
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// Interceptor 将视图解析器绑定到当前请求，供 Render 使用。
func Interceptor(resolver Resolver) arkweb.Interceptor {
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		if resolver != nil && ctx != nil && ctx.Request() != nil {
			ctx.Request().SetAttribute(AttributeResolver, resolver)
		}
		return next.Handle(ctx)
	})
}

// ResolverFromContext 从当前请求读取视图解析器。
func ResolverFromContext(ctx *arkweb.Context) (Resolver, bool) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false
	}
	value, ok := ctx.Request().Attribute(AttributeResolver)
	if !ok {
		return nil, false
	}
	resolver, ok := value.(Resolver)
	return resolver, ok && resolver != nil
}

func cleanHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}
