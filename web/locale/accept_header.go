package locale

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// AcceptHeaderResolver 使用 Accept-Language 头解析 Locale。
type AcceptHeaderResolver struct {
	defaultLocale servlet.Locale
	hasDefault    bool
}

// AcceptHeaderOption 定制 AcceptHeaderResolver。
type AcceptHeaderOption func(*AcceptHeaderResolver)

// NewAcceptHeaderResolver 创建 Accept-Language 解析器。
func NewAcceptHeaderResolver(options ...AcceptHeaderOption) *AcceptHeaderResolver {
	resolver := &AcceptHeaderResolver{}
	for _, option := range options {
		if option != nil {
			option(resolver)
		}
	}
	return resolver
}

// WithDefaultLocale 设置请求没有声明语言时使用的默认 Locale。
func WithDefaultLocale(locale servlet.Locale) AcceptHeaderOption {
	return func(resolver *AcceptHeaderResolver) {
		if resolver != nil && validLocale(locale) {
			resolver.defaultLocale = locale
			resolver.hasDefault = true
		}
	}
}

// ResolveLocale 从请求头解析 Locale。
func (r *AcceptHeaderResolver) ResolveLocale(ctx *arkweb.Context) (servlet.Locale, bool) {
	if ctx != nil && ctx.Request() != nil {
		if locale, ok := ctx.Request().Locale(); ok {
			return locale, true
		}
	}
	if r != nil && r.hasDefault {
		return r.defaultLocale, true
	}
	return servlet.Locale{}, false
}
