package web

import (
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

// Registry 收集 Web 路由、拦截器和部署选项。
type Registry struct {
	routes              []Route
	interceptors        []interceptorRegistration
	advice              []arkweb.ResponseAdvice
	errorMappers        []arkweb.ErrorMapper
	fallbackErrorMapper arkweb.ErrorMapper
	messageReader       *message.Reader
	messageWriter       *message.Writer
	requestBodyAdvice   []message.ReadAdvice
	readConverters      []message.ReadConverter
	writeConverters     []message.Converter
	validator           validation.Validator
	filters             []filterRegistration
	corsMappings        []CORSMapping
	profiles            []servletcontainer.Profile
	servlets            []servletMapping
	deploymentOptions   []servletcontainer.DeploymentOption
}

// NewRegistry 创建空 Web 注册表。
func NewRegistry() *Registry {
	return &Registry{}
}

// Handle 注册 HTTP 方法路由。
func (r *Registry) Handle(method, pattern string, handler arkweb.Handler) error {
	return r.HandleOwned(method, pattern, handler)
}

// HandleOwned 注册路由及其贡献者身份，不改变请求分派语义。
func (r *Registry) HandleOwned(method, pattern string, handler arkweb.Handler, owners ...string) error {
	route, err := NewRoute(method, pattern, handler)
	if err != nil {
		return err
	}
	route.Owners = append([]string(nil), owners...)
	r.routes = append(r.routes, route)
	return nil
}

// GET 注册 GET 路由。
func (r *Registry) GET(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodGet, pattern, handler)
}

// HEAD 注册 HEAD 路由。
func (r *Registry) HEAD(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodHead, pattern, handler)
}

// POST 注册 POST 路由。
func (r *Registry) POST(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodPost, pattern, handler)
}

// PUT 注册 PUT 路由。
func (r *Registry) PUT(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodPut, pattern, handler)
}

// PATCH 注册 PATCH 路由。
func (r *Registry) PATCH(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodPatch, pattern, handler)
}

// DELETE 注册 DELETE 路由。
func (r *Registry) DELETE(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodDelete, pattern, handler)
}

// OPTIONS 注册 OPTIONS 路由。
func (r *Registry) OPTIONS(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodOptions, pattern, handler)
}

// TRACE 注册 TRACE 路由。
func (r *Registry) TRACE(pattern string, handler arkweb.Handler) error {
	return r.Handle(http.MethodTrace, pattern, handler)
}

// Use 注册全局 Web 拦截器。
func (r *Registry) Use(interceptor arkweb.Interceptor) {
	if !isNilInterceptor(interceptor) {
		r.interceptors = append(r.interceptors, interceptorRegistration{interceptor: interceptor})
	}
}

// UseMapped 注册带路径映射的 Web 拦截器。
func (r *Registry) UseMapped(interceptor arkweb.Interceptor, mapping InterceptorMapping) {
	if !isNilInterceptor(interceptor) {
		r.interceptors = append(r.interceptors, interceptorRegistration{
			interceptor: interceptor,
			mapping:     mapping,
		})
	}
}

// UseResponseAdvice 注册全局响应增强器。
func (r *Registry) UseResponseAdvice(advice arkweb.ResponseAdvice) {
	if !isNilResponseAdvice(advice) {
		r.advice = append(r.advice, advice)
	}
}

// UseRequestBodyAdvice 注册请求体读取增强器。
func (r *Registry) UseRequestBodyAdvice(advice message.ReadAdvice) {
	if r != nil && !isNilMessageReadAdvice(advice) {
		r.requestBodyAdvice = append(r.requestBodyAdvice, advice)
	}
}

// UseErrorMapper 注册全局错误映射器。
func (r *Registry) UseErrorMapper(mapper arkweb.ErrorMapper) {
	if !isNilErrorMapper(mapper) {
		r.errorMappers = append(r.errorMappers, mapper)
	}
}

// UseFallbackErrorMapper 设置普通错误映射器均未命中时执行的兜底映射器。
func (r *Registry) UseFallbackErrorMapper(mapper arkweb.ErrorMapper) {
	if r != nil && !isNilErrorMapper(mapper) {
		r.fallbackErrorMapper = mapper
	}
}

// UseMessageReader 设置当前 Web 注册表的请求体读取器。
func (r *Registry) UseMessageReader(reader message.Reader) {
	if r != nil {
		r.messageReader = &reader
	}
}

// UseMessageWriter 设置当前 Web 注册表的响应体写出器。
func (r *Registry) UseMessageWriter(writer message.Writer) {
	if r != nil {
		r.messageWriter = &writer
	}
}

// AddMessageReadConverter 添加请求体读取转换器，优先级高于默认转换器。
func (r *Registry) AddMessageReadConverter(converter message.ReadConverter) {
	if r != nil && !isNilMessageConverter(converter) {
		r.readConverters = append(r.readConverters, converter)
	}
}

// AddMessageConverter 添加响应体写出转换器，优先级高于默认转换器。
func (r *Registry) AddMessageConverter(converter message.Converter) {
	if r != nil && !isNilMessageConverter(converter) {
		r.writeConverters = append(r.writeConverters, converter)
	}
}

// UseValidator 设置当前 Web 注册表的请求校验器。
func (r *Registry) UseValidator(validator validation.Validator) {
	if r != nil && !isNilValidator(validator) {
		r.validator = validator
	}
}

// AddFilter 添加 Servlet 过滤器。
func (r *Registry) AddFilter(filter servlet.Filter) {
	if !isNilFilter(filter) {
		r.filters = append(r.filters, filterRegistration{filter: filter})
	}
}

// AddMappedFilter 添加带路径映射的 Servlet 过滤器。
func (r *Registry) AddMappedFilter(filter servlet.Filter, mapping FilterMapping) {
	if !isNilFilter(filter) {
		r.filters = append(r.filters, filterRegistration{
			filter:  filter,
			mapping: mapping,
		})
	}
}

// RequireProfile 声明部署需要的 Arkarta Servlet Profile。
func (r *Registry) RequireProfile(profile servletcontainer.Profile) {
	if profile == "" || hasProfile(r.profiles, profile) {
		return
	}
	r.profiles = append(r.profiles, profile)
}

// AddDeploymentOption 添加底层 Servlet 部署选项。
func (r *Registry) AddDeploymentOption(option servletcontainer.DeploymentOption) {
	if option != nil {
		r.deploymentOptions = append(r.deploymentOptions, option)
	}
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}
