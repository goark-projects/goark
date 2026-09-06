package web

import (
	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

// Router 构造 Arkarta Web Router。
func (r *Registry) Router(options ...arkweb.Option) (*arkweb.Router, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}
	routerOptions := appendRouterOptions(r.errorMappers, r.fallbackErrorMapper, r.validator, options)
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

// RequestBodyAdvice 返回请求体读取增强器快照。
func (r *Registry) RequestBodyAdvice() []message.ReadAdvice {
	if r == nil {
		return nil
	}
	return append([]message.ReadAdvice(nil), r.requestBodyAdvice...)
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
	return mappedInterceptor{target: r.interceptor, mapping: r.mapping}
}

func hasProfile(profiles []servletcontainer.Profile, target servletcontainer.Profile) bool {
	for _, profile := range profiles {
		if profile == target {
			return true
		}
	}
	return false
}

func appendRouterOptions(mappers []arkweb.ErrorMapper, fallback arkweb.ErrorMapper, validator validation.Validator, options []arkweb.Option) []arkweb.Option {
	if len(mappers) == 0 && isNilErrorMapper(fallback) && isNilValidator(validator) {
		return options
	}
	routerOptions := make([]arkweb.Option, 0, len(options)+2)
	if len(mappers) > 0 || !isNilErrorMapper(fallback) {
		routerOptions = append(routerOptions, arkweb.WithErrorMapper(newErrorMapperChain(fallback, mappers)))
	}
	if !isNilValidator(validator) {
		routerOptions = append(routerOptions, arkweb.WithValidator(validator))
	}
	routerOptions = append(routerOptions, options...)
	return routerOptions
}

func (r *Registry) hasMessageIO() bool {
	return r.messageReader != nil || r.messageWriter != nil ||
		len(r.requestBodyAdvice) > 0 || len(r.readConverters) > 0 || len(r.writeConverters) > 0
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
	if len(r.requestBodyAdvice) > 0 {
		reader = message.NewReader(
			message.WithReadConverters(reader.ReadConverters()...),
			message.WithReadAdvice(reader.ReadAdvices()...),
			message.WithAppendedReadAdvice(r.requestBodyAdvice...),
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
