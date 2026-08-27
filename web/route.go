package web

import arkweb "goark.dev/arkarta/web"

// Route 描述一个 Web 路由。
type Route struct {
	Method  string
	Pattern string
	Handler arkweb.Handler
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
