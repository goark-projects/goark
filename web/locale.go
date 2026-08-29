package web

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// RequestLocale 返回请求 Accept-Language 中优先级最高的 Locale。
func RequestLocale(ctx *arkweb.Context) (servlet.Locale, bool) {
	if ctx == nil || ctx.Request() == nil {
		return servlet.Locale{}, false
	}
	return ctx.Request().Locale()
}

// RequestLocales 按客户端声明优先级返回请求 Locale 列表。
func RequestLocales(ctx *arkweb.Context) []servlet.Locale {
	if ctx == nil || ctx.Request() == nil {
		return nil
	}
	return ctx.Request().Locales()
}

// SetResponseLocale 设置响应 Content-Language。
func SetResponseLocale(ctx *arkweb.Context, locale servlet.Locale) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	return servlet.SetLocale(ctx.Response(), locale)
}
