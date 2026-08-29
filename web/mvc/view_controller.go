package mvc

import (
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// ViewControllerOption 定制简单视图控制器。
type ViewControllerOption func(*viewControllerOptions)

type viewControllerOptions struct {
	status int
}

// ViewController 创建只渲染固定逻辑视图名的 GET 路由。
func ViewController(pattern string, viewName string, options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, strings.TrimSpace(viewName), options...)
}

// RedirectViewController 创建只执行重定向的 GET 路由。
func RedirectViewController(pattern string, location string, options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, prefixedViewControllerName(RedirectViewNamePrefix, location), options...)
}

// ForwardViewController 创建只执行服务端转发的 GET 路由。
func ForwardViewController(pattern string, target string, options ...ViewControllerOption) Route {
	return viewControllerRoute(pattern, prefixedViewControllerName(ForwardViewNamePrefix, target), options...)
}

// WithViewControllerStatus 设置简单视图控制器响应状态码。
func WithViewControllerStatus(statusCode int) ViewControllerOption {
	return func(options *viewControllerOptions) {
		options.status = normalizeResponseStatus(statusCode, 0)
	}
}

func viewControllerRoute(pattern string, viewName string, options ...ViewControllerOption) Route {
	config := newViewControllerOptions(options)
	return GET(pattern, arkweb.HandlerFunc(func(*arkweb.Context) (arkweb.Result, error) {
		return NewModelAndView(viewName, nil, WithViewStatus(config.status)), nil
	}))
}

func newViewControllerOptions(options []ViewControllerOption) viewControllerOptions {
	var config viewControllerOptions
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}

func prefixedViewControllerName(prefix string, value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, prefix) {
		return value
	}
	return prefix + value
}
