package locale

import (
	"reflect"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// Resolver 按请求上下文解析 Locale。
type Resolver interface {
	ResolveLocale(ctx *arkweb.Context) (servlet.Locale, bool)
}

// MutableResolver 支持在当前请求中切换 Locale。
type MutableResolver interface {
	Resolver
	SetLocale(ctx *arkweb.Context, locale servlet.Locale) error
}

// ResolverFunc 将普通函数适配为 Resolver。
type ResolverFunc func(ctx *arkweb.Context) (servlet.Locale, bool)

// ResolveLocale 调用底层函数解析 Locale。
func (f ResolverFunc) ResolveLocale(ctx *arkweb.Context) (servlet.Locale, bool) {
	if f == nil {
		return servlet.Locale{}, false
	}
	return f(ctx)
}

func isNilResolver(resolver Resolver) bool {
	if resolver == nil {
		return true
	}
	value := reflect.ValueOf(resolver)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func isNilMutableResolver(resolver MutableResolver) bool {
	if resolver == nil {
		return true
	}
	value := reflect.ValueOf(resolver)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
