package locale

import (
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const (
	// AttributeCurrent 保存 resolver 为当前请求选定的 Locale。
	AttributeCurrent = "goark.web.locale.current"
)

// Current 返回当前请求 Locale，优先使用 resolver 写入的请求属性。
func Current(ctx *arkweb.Context) (servlet.Locale, bool) {
	if locale, ok := currentAttribute(ctx); ok {
		return locale, true
	}
	if ctx == nil || ctx.Request() == nil {
		return servlet.Locale{}, false
	}
	return ctx.Request().Locale()
}

// Locales 返回当前 Locale 和客户端可接受 Locale 的有序集合。
func Locales(ctx *arkweb.Context) []servlet.Locale {
	if ctx == nil || ctx.Request() == nil {
		return nil
	}
	accepted := ctx.Request().Locales()
	current, ok := currentAttribute(ctx)
	if !ok {
		return accepted
	}
	out := make([]servlet.Locale, 0, len(accepted)+1)
	out = append(out, current)
	for _, item := range accepted {
		if item.Tag() != current.Tag() {
			out = append(out, item)
		}
	}
	return out
}

// SetCurrent 设置当前请求 Locale。
func SetCurrent(ctx *arkweb.Context, locale servlet.Locale) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	if !validLocale(locale) {
		return ErrInvalidLocale
	}
	ctx.Request().SetAttribute(AttributeCurrent, locale)
	return nil
}

// ClearCurrent 清除当前请求 Locale。
func ClearCurrent(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	ctx.Request().SetAttribute(AttributeCurrent, nil)
	return nil
}

func currentAttribute(ctx *arkweb.Context) (servlet.Locale, bool) {
	if ctx == nil || ctx.Request() == nil {
		return servlet.Locale{}, false
	}
	value, ok := ctx.Request().Attribute(AttributeCurrent)
	if !ok {
		return servlet.Locale{}, false
	}
	locale, ok := value.(servlet.Locale)
	if !ok || !validLocale(locale) {
		return servlet.Locale{}, false
	}
	return locale, true
}

func validLocale(locale servlet.Locale) bool {
	return locale.Tag() != ""
}
