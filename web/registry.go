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
	routes            []Route
	interceptors      []interceptorRegistration
	advice            []arkweb.ResponseAdvice
	errorMappers      []arkweb.ErrorMapper
	messageReader     *message.Reader
	messageWriter     *message.Writer
	readConverters    []message.ReadConverter
	writeConverters   []message.Converter
	validator         validation.Validator
	filters           []filterRegistration
	corsMappings      []CORSMapping
	profiles          []servletcontainer.Profile
	servlets          []servletMapping
	deploymentOptions []servletcontainer.DeploymentOption
}

// NewRegistry 创建空 Web 注册表。
func NewRegistry() *Registry {
	return &Registry{}
}

// Handle 注册 HTTP 方法路由。
func (r *Registry) Handle(method, pattern string, handler arkweb.Handler) error {
	route, err := NewRoute(method, pattern, handler)
	if err != nil {
		return err
	}
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
	if advice != nil {
		r.advice = append(r.advice, advice)
	}
}

// UseErrorMapper 注册全局错误映射器。
func (r *Registry) UseErrorMapper(mapper arkweb.ErrorMapper) {
	if !isNilErrorMapper(mapper) {
		r.errorMappers = append(r.errorMappers, mapper)
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

// Router 构造 Arkarta Web Router。
func (r *Registry) Router(options ...arkweb.Option) (*arkweb.Router, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}
	routerOptions := appendRouterOptions(r.errorMappers, r.validator, options)
	router := arkweb.NewRouter(routerOptions...)
	if r.hasMessageIO() {
		router.Use(message.ContextInterceptor(r.currentMessageReader(), r.currentMessageWriter()))
	}
	for _, registration := range r.interceptors {
		router.Use(registration.Interceptor())
	}
	for _, advice := range r.advice {
		router.UseResponseAdvice(advice)
	}
	routes, err := applyCORSMappings(r.routes, r.corsMappings)
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if err := router.Handle(route.Method, route.Pattern, route.Handler); err != nil {
			return nil, err
		}
	}
	return router, nil
}

// Routes 返回路由快照。
func (r *Registry) Routes() []Route {
	if r == nil {
		return nil
	}
	return append([]Route(nil), r.routes...)
}

// ErrorMappers 返回错误映射器快照。
func (r *Registry) ErrorMappers() []arkweb.ErrorMapper {
	if r == nil {
		return nil
	}
	return append([]arkweb.ErrorMapper(nil), r.errorMappers...)
}

// MessageReadConverters 返回请求体读取转换器快照。
func (r *Registry) MessageReadConverters() []message.ReadConverter {
	if r == nil {
		return nil
	}
	return append([]message.ReadConverter(nil), r.readConverters...)
}

// MessageConverters 返回响应体写出转换器快照。
func (r *Registry) MessageConverters() []message.Converter {
	if r == nil {
		return nil
	}
	return append([]message.Converter(nil), r.writeConverters...)
}

// Validator 返回当前 Web 注册表的校验器。
func (r *Registry) Validator() validation.Validator {
	if r == nil {
		return nil
	}
	return r.validator
}

// Filters 返回 Servlet 过滤器快照。
func (r *Registry) Filters() []servlet.Filter {
	if r == nil {
		return nil
	}
	filters := make([]servlet.Filter, 0, len(r.filters))
	for _, registration := range r.filters {
		filters = append(filters, registration.Filter())
	}
	return filters
}

// Profiles 返回 Arkarta Servlet Profile 快照。
func (r *Registry) Profiles() []servletcontainer.Profile {
	if r == nil {
		return nil
	}
	return append([]servletcontainer.Profile(nil), r.profiles...)
}

// DeploymentOptions 返回部署选项快照。
func (r *Registry) DeploymentOptions() []servletcontainer.DeploymentOption {
	if r == nil {
		return nil
	}
	return append([]servletcontainer.DeploymentOption(nil), r.deploymentOptions...)
}

type interceptorRegistration struct {
	interceptor arkweb.Interceptor
	mapping     InterceptorMapping
}

func (r interceptorRegistration) Interceptor() arkweb.Interceptor {
	if len(r.mapping.includes) == 0 && len(r.mapping.excludes) == 0 {
		return r.interceptor
	}
	return mappedInterceptor{
		target:  r.interceptor,
		mapping: r.mapping,
	}
}

func hasProfile(profiles []servletcontainer.Profile, target servletcontainer.Profile) bool {
	for _, profile := range profiles {
		if profile == target {
			return true
		}
	}
	return false
}

func appendRouterOptions(mappers []arkweb.ErrorMapper, validator validation.Validator, options []arkweb.Option) []arkweb.Option {
	if len(mappers) == 0 && isNilValidator(validator) {
		return options
	}
	routerOptions := make([]arkweb.Option, 0, len(options)+2)
	if len(mappers) > 0 {
		routerOptions = append(routerOptions, arkweb.WithErrorMapper(NewErrorMapperChain(mappers...)))
	}
	if !isNilValidator(validator) {
		routerOptions = append(routerOptions, arkweb.WithValidator(validator))
	}
	routerOptions = append(routerOptions, options...)
	return routerOptions
}

func (r *Registry) hasMessageIO() bool {
	return r.messageReader != nil || r.messageWriter != nil || len(r.readConverters) > 0 || len(r.writeConverters) > 0
}

func (r *Registry) currentMessageReader() message.Reader {
	reader := message.NewReader()
	if r.messageReader != nil {
		reader = *r.messageReader
	}
	if len(r.readConverters) > 0 {
		reader = message.NewReader(
			message.WithReadConverters(reader.ReadConverters()...),
			message.WithPrependedReadConverters(r.readConverters...),
		)
	}
	return reader
}

func (r *Registry) currentMessageWriter() message.Writer {
	writer := message.NewWriter()
	if r.messageWriter != nil {
		writer = *r.messageWriter
	}
	if len(r.writeConverters) > 0 {
		writer = message.NewWriter(
			message.WithConverters(writer.Converters()...),
			message.WithPrependedConverters(r.writeConverters...),
		)
	}
	return writer
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}
