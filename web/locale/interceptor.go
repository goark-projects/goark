package locale

import (
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const (
	// DefaultChangeParameterName 是 LocaleChangeInterceptor 的默认请求参数名。
	DefaultChangeParameterName = "locale"
)

// ChangeOption 定制 Locale 切换拦截器。
type ChangeOption func(*changeConfig) error

type changeConfig struct {
	parameterName string
	resolver      MutableResolver
	methods       map[string]struct{}
}

// ResolverInterceptor 将 Resolver 结果写入当前请求属性。
func ResolverInterceptor(resolver Resolver) (arkweb.Interceptor, error) {
	if isNilResolver(resolver) {
		return nil, ErrNilResolver
	}
	return resolverInterceptor{resolver: resolver}, nil
}

// ChangeInterceptor 创建基于请求参数切换 Locale 的拦截器。
func ChangeInterceptor(options ...ChangeOption) (arkweb.Interceptor, error) {
	config := changeConfig{
		parameterName: DefaultChangeParameterName,
		resolver:      NewAttributeResolver(),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(config.parameterName) == "" {
		return nil, ErrInvalidParameterName
	}
	if isNilMutableResolver(config.resolver) {
		return nil, ErrNilResolver
	}
	return changeInterceptor{config: config}, nil
}

// WithParameterName 设置用于切换 Locale 的请求参数名。
func WithParameterName(name string) ChangeOption {
	return func(config *changeConfig) error {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, " \t\r\n\x00") {
			return ErrInvalidParameterName
		}
		config.parameterName = name
		return nil
	}
}

// WithResolver 设置用于保存 Locale 的可变解析器。
func WithResolver(resolver MutableResolver) ChangeOption {
	return func(config *changeConfig) error {
		if isNilMutableResolver(resolver) {
			return ErrNilResolver
		}
		config.resolver = resolver
		return nil
	}
}

// WithHTTPMethods 限定允许切换 Locale 的 HTTP 方法；不设置时允许全部方法。
func WithHTTPMethods(methods ...string) ChangeOption {
	return func(config *changeConfig) error {
		if len(methods) == 0 {
			return ErrInvalidHTTPMethod
		}
		config.methods = make(map[string]struct{}, len(methods))
		for _, method := range methods {
			method = cleanHTTPMethod(method)
			if method == "" {
				return ErrInvalidHTTPMethod
			}
			config.methods[method] = struct{}{}
		}
		return nil
	}
}

type resolverInterceptor struct {
	resolver Resolver
}

func (i resolverInterceptor) Intercept(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
	locale, ok := i.resolver.ResolveLocale(ctx)
	if !ok {
		return next.Handle(ctx)
	}
	if err := SetCurrent(ctx, locale); err != nil {
		return nil, err
	}
	return next.Handle(ctx)
}

type changeInterceptor struct {
	config changeConfig
}

func (i changeInterceptor) Intercept(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
	if ctx == nil || ctx.Request() == nil {
		return next.Handle(ctx)
	}
	if !i.allowsMethod(ctx.Request().Method()) {
		return next.Handle(ctx)
	}
	value := strings.TrimSpace(ctx.QueryValue(i.config.parameterName))
	if value == "" {
		return next.Handle(ctx)
	}
	locale, ok := servlet.NewLocale(value)
	if !ok {
		return nil, servlet.NewHTTPError(http.StatusBadRequest, "非法 Locale 参数", nil)
	}
	if err := i.config.resolver.SetLocale(ctx, locale); err != nil {
		return nil, err
	}
	if resolved, ok := i.config.resolver.ResolveLocale(ctx); ok {
		locale = resolved
	}
	if err := SetCurrent(ctx, locale); err != nil {
		return nil, err
	}
	return next.Handle(ctx)
}

func (i changeInterceptor) allowsMethod(method string) bool {
	if len(i.config.methods) == 0 {
		return true
	}
	_, ok := i.config.methods[cleanHTTPMethod(method)]
	return ok
}

func cleanHTTPMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" || strings.ContainsAny(method, " \t\r\n\x00") {
		return ""
	}
	return method
}
