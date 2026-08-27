package mvc

import (
	"net/http"

	arkweb "goark.dev/arkarta/web"
)

// Route 描述 MVC 路由。
type Route struct {
	Method  string
	Pattern string
	Handler arkweb.Handler
}

// Handle 创建 MVC 路由描述。
func Handle(method, pattern string, handler arkweb.Handler) Route {
	return Route{
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	}
}

// GET 创建 GET 路由描述。
func GET(pattern string, handler arkweb.Handler) Route {
	return Handle(http.MethodGet, pattern, handler)
}

// POST 创建 POST 路由描述。
func POST(pattern string, handler arkweb.Handler) Route {
	return Handle(http.MethodPost, pattern, handler)
}

// PUT 创建 PUT 路由描述。
func PUT(pattern string, handler arkweb.Handler) Route {
	return Handle(http.MethodPut, pattern, handler)
}

// PATCH 创建 PATCH 路由描述。
func PATCH(pattern string, handler arkweb.Handler) Route {
	return Handle(http.MethodPatch, pattern, handler)
}

// DELETE 创建 DELETE 路由描述。
func DELETE(pattern string, handler arkweb.Handler) Route {
	return Handle(http.MethodDelete, pattern, handler)
}
