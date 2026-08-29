package locale

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// AttributeResolver 将当前 Locale 存放在请求属性中。
type AttributeResolver struct {
	fallback Resolver
}

// AttributeOption 定制 AttributeResolver。
type AttributeOption func(*AttributeResolver)

// NewAttributeResolver 创建请求属性 Locale 解析器。
func NewAttributeResolver(options ...AttributeOption) *AttributeResolver {
	resolver := &AttributeResolver{fallback: NewAcceptHeaderResolver()}
	for _, option := range options {
		if option != nil {
			option(resolver)
		}
	}
	return resolver
}

// WithFallbackResolver 设置请求属性缺失时使用的回退解析器。
func WithFallbackResolver(fallback Resolver) AttributeOption {
	return func(resolver *AttributeResolver) {
		if resolver != nil && !isNilResolver(fallback) {
			resolver.fallback = fallback
		}
	}
}

// ResolveLocale 从请求属性读取 Locale，缺失时回退到委托解析器。
func (r *AttributeResolver) ResolveLocale(ctx *arkweb.Context) (servlet.Locale, bool) {
	if locale, ok := currentAttribute(ctx); ok {
		return locale, true
	}
	if r != nil && !isNilResolver(r.fallback) {
		return r.fallback.ResolveLocale(ctx)
	}
	return servlet.Locale{}, false
}

// SetLocale 设置当前请求 Locale。
func (r *AttributeResolver) SetLocale(ctx *arkweb.Context, locale servlet.Locale) error {
	return SetCurrent(ctx, locale)
}
