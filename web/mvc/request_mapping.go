package mvc

import (
	"net/http"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

var defaultRequestMappingMethods = [...]string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// RequestMapping 创建无 HTTP method 限定的 MVC 路由集合，对齐 Spring @RequestMapping 默认语义。
func RequestMapping(pattern string, handler arkweb.Handler, options ...RouteOption) []Route {
	return RequestMappingMethods(nil, pattern, handler, options...)
}

// RequestMappingMethods 创建限定到指定 HTTP method 集合的 MVC 路由。
func RequestMappingMethods(methods []string, pattern string, handler arkweb.Handler, options ...RouteOption) []Route {
	normalized := normalizeRequestMappingMethods(methods)
	routes := make([]Route, 0, len(normalized))
	for _, method := range normalized {
		routes = append(routes, Handle(method, pattern, handler, options...))
	}
	return routes
}

func normalizeRequestMappingMethods(methods []string) []string {
	if len(methods) == 0 {
		return defaultRequestMappingMethods[:]
	}
	return normalizeExplicitRequestMethods(methods)
}

func normalizeExplicitRequestMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		method = strings.ToUpper(strings.TrimSpace(method))
		if _, exists := seen[method]; exists {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	return out
}

func hasRequestMethod(methods []string, method string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	for _, item := range methods {
		if item == method {
			return true
		}
	}
	return false
}
