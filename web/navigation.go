package web

import (
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	weblocale "goark.dev/goark/web/locale"
)

// RequestLocale 返回请求 Accept-Language 中优先级最高的 Locale。
func RequestLocale(ctx *arkweb.Context) (servlet.Locale, bool) {
	return weblocale.Current(ctx)
}

// RequestLocales 按客户端声明优先级返回请求 Locale 列表。
func RequestLocales(ctx *arkweb.Context) []servlet.Locale {
	return weblocale.Locales(ctx)
}

// SetResponseLocale 设置响应 Content-Language。
func SetResponseLocale(ctx *arkweb.Context, locale servlet.Locale) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	return servlet.SetLocale(ctx.Response(), locale)
}

// RedirectOption 定制重定向响应。
type RedirectOption func(*redirectOptions)

type redirectOptions struct {
	statusCode int
	headers    http.Header
}

type redirectResult struct {
	location string
	options  redirectOptions
}

// Redirect 创建 302 Found 重定向响应。
func Redirect(location string, options ...RedirectOption) arkweb.Result {
	return redirectResult{
		location: location,
		options:  newRedirectOptions(options),
	}
}

// SeeOther 创建 303 See Other 重定向响应。
func SeeOther(location string, options ...RedirectOption) arkweb.Result {
	all := make([]RedirectOption, 0, len(options)+1)
	all = append(all, WithRedirectStatus(http.StatusSeeOther))
	all = append(all, options...)
	return Redirect(location, all...)
}

// PermanentRedirect 创建 308 Permanent Redirect 重定向响应。
func PermanentRedirect(location string, options ...RedirectOption) arkweb.Result {
	all := make([]RedirectOption, 0, len(options)+1)
	all = append(all, WithRedirectStatus(http.StatusPermanentRedirect))
	all = append(all, options...)
	return Redirect(location, all...)
}

// WithRedirectStatus 设置重定向状态码。
func WithRedirectStatus(statusCode int) RedirectOption {
	return func(options *redirectOptions) {
		if isRedirectStatus(statusCode) {
			options.statusCode = statusCode
		}
	}
}

// WithRedirectHeader 设置重定向响应头。
func WithRedirectHeader(name string, value string) RedirectOption {
	return func(options *redirectOptions) {
		name = http.CanonicalHeaderKey(strings.TrimSpace(name))
		if name == "" || strings.ContainsAny(value, "\r\n") {
			return
		}
		if options.headers == nil {
			options.headers = make(http.Header, 1)
		}
		options.headers.Set(name, value)
	}
}

// Write 写出重定向响应。
func (r redirectResult) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	location := cleanRedirectLocation(r.location)
	if location == "" {
		return servlet.NewHTTPError(
			http.StatusInternalServerError,
			http.StatusText(http.StatusInternalServerError),
			ErrInvalidRedirectLocation,
		)
	}
	response := ctx.Response()
	applyEntityHeaders(response.Header(), r.options.headers)
	response.Header().Set("Location", location)
	response.SetStatus(normalizeEntityStatus(r.options.statusCode, http.StatusFound))
	return nil
}

func newRedirectOptions(options []RedirectOption) redirectOptions {
	out := redirectOptions{statusCode: http.StatusFound}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

func cleanRedirectLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" || strings.ContainsAny(location, "\r\n") {
		return ""
	}
	return location
}

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// Route 描述一个 Web 路由。
type Route struct {
	Method  string
	Pattern string
	Handler arkweb.Handler
	// Owners 保存实际装配的路由贡献者身份，空值表示手写匿名路由。
	Owners []string
}

// NewRoute 创建标准化路由描述。
func NewRoute(method, pattern string, handler arkweb.Handler) (Route, error) {
	method = normalizeMethod(method)
	if method == "" || pattern == "" || handler == nil {
		return Route{}, ErrInvalidRoute
	}
	return Route{
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	}, nil
}
