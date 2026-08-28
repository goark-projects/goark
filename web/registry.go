package web

import (
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	arkweb "goark.dev/arkarta/web"
)

// Registry 收集 Web 路由、拦截器和部署选项。
type Registry struct {
	routes            []Route
	interceptors      []arkweb.Interceptor
	advice            []arkweb.ResponseAdvice
	errorMappers      []arkweb.ErrorMapper
	filters           []servlet.Filter
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

// Use 注册全局 Web 拦截器。
func (r *Registry) Use(interceptor arkweb.Interceptor) {
	if !isNilInterceptor(interceptor) {
		r.interceptors = append(r.interceptors, interceptor)
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

// AddFilter 添加 Servlet 过滤器。
func (r *Registry) AddFilter(filter servlet.Filter) {
	if !isNilFilter(filter) {
		r.filters = append(r.filters, filter)
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
	routerOptions := appendRouterOptions(r.errorMappers, options)
	router := arkweb.NewRouter(routerOptions...)
	for _, interceptor := range r.interceptors {
		router.Use(interceptor)
	}
	for _, advice := range r.advice {
		router.UseResponseAdvice(advice)
	}
	for _, route := range r.routes {
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

// Filters 返回 Servlet 过滤器快照。
func (r *Registry) Filters() []servlet.Filter {
	if r == nil {
		return nil
	}
	return append([]servlet.Filter(nil), r.filters...)
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

func hasProfile(profiles []servletcontainer.Profile, target servletcontainer.Profile) bool {
	for _, profile := range profiles {
		if profile == target {
			return true
		}
	}
	return false
}

func appendRouterOptions(mappers []arkweb.ErrorMapper, options []arkweb.Option) []arkweb.Option {
	if len(mappers) == 0 {
		return options
	}
	routerOptions := make([]arkweb.Option, 0, len(options)+1)
	routerOptions = append(routerOptions, arkweb.WithErrorMapper(NewErrorMapperChain(mappers...)))
	routerOptions = append(routerOptions, options...)
	return routerOptions
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}
