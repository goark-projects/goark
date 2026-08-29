package web

import "errors"

var (
	// ErrNilConfigurer 表示 Web 配置器为空。
	ErrNilConfigurer = errors.New("goark/web: configurer is nil")
	// ErrNilRegistry 表示 Web 注册表为空。
	ErrNilRegistry = errors.New("goark/web: registry is nil")
	// ErrInvalidRoute 表示路由描述非法。
	ErrInvalidRoute = errors.New("goark/web: invalid route")
	// ErrNilErrorMapper 表示错误映射器为空。
	ErrNilErrorMapper = errors.New("goark/web: error mapper is nil")
	// ErrNilResponseAdvice 表示响应增强器为空。
	ErrNilResponseAdvice = errors.New("goark/web: response advice is nil")
	// ErrNilMessageConverter 表示消息转换器为空。
	ErrNilMessageConverter = errors.New("goark/web: message converter is nil")
	// ErrNilInterceptor 表示 Web 拦截器为空。
	ErrNilInterceptor = errors.New("goark/web: interceptor is nil")
	// ErrInvalidInterceptorMapping 表示拦截器路径映射非法。
	ErrInvalidInterceptorMapping = errors.New("goark/web: invalid interceptor mapping")
	// ErrNilFilter 表示 Servlet 过滤器为空。
	ErrNilFilter = errors.New("goark/web: filter is nil")
	// ErrNilServlet 表示 Servlet 为空。
	ErrNilServlet = errors.New("goark/web: servlet is nil")
	// ErrNilDownloadReader 表示下载响应体为空。
	ErrNilDownloadReader = errors.New("goark/web: download reader is nil")
	// ErrInvalidRedirectLocation 表示重定向 Location 非法。
	ErrInvalidRedirectLocation = errors.New("goark/web: invalid redirect location")
)
