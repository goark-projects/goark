package mvc

import (
	"net/http"
	"strings"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/cors"
)

// Route 描述 MVC 路由。
type Route struct {
	Method     string
	Pattern    string
	Handler    arkweb.Handler
	Conditions Conditions

	crossOrigin *cors.Config
}

// RouteOption 定制 MVC 路由描述。
type RouteOption func(*Route)

// Handle 创建 MVC 路由描述。
func Handle(method, pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	route := Route{
		Method:     method,
		Pattern:    pattern,
		Handler:    handler,
		Conditions: Conditions{},
	}
	for _, option := range options {
		if option != nil {
			option(&route)
		}
	}
	return route
}

// GET 创建 GET 路由描述。
func GET(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodGet, pattern, handler, options...)
}

// HEAD 创建 HEAD 路由描述。
func HEAD(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodHead, pattern, handler, options...)
}

// POST 创建 POST 路由描述。
func POST(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPost, pattern, handler, options...)
}

// PUT 创建 PUT 路由描述。
func PUT(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPut, pattern, handler, options...)
}

// PATCH 创建 PATCH 路由描述。
func PATCH(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPatch, pattern, handler, options...)
}

// DELETE 创建 DELETE 路由描述。
func DELETE(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodDelete, pattern, handler, options...)
}

// OPTIONS 创建 OPTIONS 路由描述。
func OPTIONS(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodOptions, pattern, handler, options...)
}

// TRACE 创建 TRACE 路由描述。
func TRACE(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodTrace, pattern, handler, options...)
}

// WithConsumes 设置请求 Content-Type 条件。
func WithConsumes(mediaTypes ...string) RouteOption {
	copied := cleanRouteValues(mediaTypes)
	return func(route *Route) {
		route.Conditions.Consumes = copied
	}
}

// WithProduces 设置请求 Accept 条件。
func WithProduces(mediaTypes ...string) RouteOption {
	copied := cleanRouteValues(mediaTypes)
	return func(route *Route) {
		route.Conditions.Produces = copied
	}
}

// WithParams 设置请求参数条件，支持 name、!name、name=value、name!=value。
func WithParams(expressions ...string) RouteOption {
	copied := cleanRouteValues(expressions)
	return func(route *Route) {
		route.Conditions.Params = copied
	}
}

// WithHeaders 设置请求头条件，支持 name、!name、name=value、name!=value。
func WithHeaders(expressions ...string) RouteOption {
	copied := cleanRouteValues(expressions)
	return func(route *Route) {
		route.Conditions.Headers = copied
	}
}

// WithCrossOrigin 设置路由级 CORS 策略，对齐 Spring 方法级 @CrossOrigin。
func WithCrossOrigin(config cors.Config) RouteOption {
	copied := cloneCrossOriginConfig(config)
	return func(route *Route) {
		route.crossOrigin = copied
	}
}

func cleanRouteValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}
