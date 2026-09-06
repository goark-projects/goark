package mvc

import (
	"net/http"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/goark/container"
)

func appendControllerRegistrations(
	registry *goweb.Registry,
	controller Controller,
	groups map[routeRegistrationKey][]routeRegistration,
	keys *[]routeRegistrationKey,
) error {
	var implicitGroups map[uint64]struct{}
	for _, route := range controller.routes {
		if controller.skipImplicitRoute(route, implicitGroups) {
			continue
		}
		if route.implicitMethods && len(controller.methods) > 0 && route.methodGroupID != 0 {
			if implicitGroups == nil {
				implicitGroups = make(map[uint64]struct{})
			}
			implicitGroups[route.methodGroupID] = struct{}{}
		}
		for _, prefix := range controller.registrationPathPrefixes() {
			for _, method := range controller.routeMethods(route) {
				resolvedRoute := route
				resolvedRoute.Method = method
				resolvedRoute.Pattern = joinControllerRoutePattern(prefix, route.Pattern)
				if err := registerRouteCORS(registry, controller, resolvedRoute); err != nil {
					return err
				}
				key := routeRegistrationKey{method: resolvedRoute.Method, pattern: resolvedRoute.Pattern}
				if _, exists := groups[key]; !exists {
					*keys = append(*keys, key)
				}
				handler := wrapModelAttributeInitializers(resolvedRoute.Handler, controller.modelAttrs)
				handler = wrapInitBinders(handler, controller.binders)
				handler = wrapSessionAttributes(handler, controller.sessionAttrs)
				groups[key] = append(groups[key], routeRegistration{
					handler:    bindControllerKind(controller.kind, handler),
					conditions: mergeControllerRouteConditions(controller.conditions, resolvedRoute.Conditions),
				})
			}
		}
	}
	return nil
}

func (a ControllerAdvice) wrapExceptionHandler(handler goweb.ErrorMapper) goweb.ErrorMapper {
	return goweb.ErrorMapperFunc(func(ctx *arkweb.Context, err error) arkweb.Result {
		if ctx == nil || ctx.Request() == nil {
			return handler.MapError(ctx, err)
		}
		request := ctx.Request()
		previous, existed := request.Attribute(AttributeControllerAdviceKind)
		request.SetAttribute(AttributeControllerAdviceKind, a.kind)
		defer func() {
			if existed {
				request.SetAttribute(AttributeControllerAdviceKind, previous)
				return
			}
			request.SetAttribute(AttributeControllerAdviceKind, nil)
		}()
		return handler.MapError(ctx, err)
	})
}

func (c Conditions) match(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	req := ctx.Request()
	if err := matchConsumes(req, c.Consumes); err != nil {
		return err
	}
	if err := matchProduces(req, c.Produces); err != nil {
		return err
	}
	if err := matchParameterExpressions(req, c.Params); err != nil {
		return err
	}
	return matchHeaderExpressions(req, c.Headers)
}

func bindBody[In any, Out any](statusCode int, fn BindFunc[In, Out], groups []string,
	mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateBody(ctx, &input, validationGroups, readMediaTypes); err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

// ControllerKindFromContext 返回当前请求命中的控制器类型。
func ControllerKindFromContext(ctx *arkweb.Context) ControllerKind {
	if ctx == nil || ctx.Request() == nil {
		return ControllerKindView
	}
	value, ok := ctx.Request().Attribute(AttributeControllerKind)
	if !ok {
		return ControllerKindView
	}
	kind, ok := value.(ControllerKind)
	if !ok {
		return ControllerKindView
	}
	return kind
}

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

func bindEntity[In any, Out any](fn BindEntityFunc[In, Out], groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateJSON(ctx, &input, validationGroups); err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}

// PathPrefixes 返回控制器级路径前缀快照。
func (c Controller) PathPrefixes() []string {
	out := make([]string, 0, len(c.pathPrefixes))
	for _, prefix := range c.pathPrefixes {
		if prefix == "" {
			out = append(out, "/")
			continue
		}
		out = append(out, prefix)
	}
	return out
}

// RegisterMappedHandlerInterceptor 注册带路径映射的 MVC HandlerInterceptor 配置贡献点。
func RegisterMappedHandlerInterceptor(
	registry *container.Registry,
	name string,
	interceptor HandlerInterceptor,
	mapping goweb.InterceptorMapping,
	options ...container.Option,
) error {
	return goweb.RegisterMappedInterceptor(registry, name, HandlerInterceptorAdapter(interceptor),
		mapping, options...)
}

func restoreRequestAttribute(ctx *arkweb.Context, name string, previous any, existed bool) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	if existed {
		ctx.Request().SetAttribute(name, previous)
		return
	}
	ctx.Request().SetAttribute(name, nil)
}

// WithMappedHandlerInterceptor 添加带路径映射的 MVC 处理器拦截器。
func (c Configuration) WithMappedHandlerInterceptor(interceptor HandlerInterceptor,
	mapping goweb.InterceptorMapping) Configuration {
	c.mappedHandlerInterceptors = append(c.mappedHandlerInterceptors, mappedHandlerInterceptor{
		interceptor: interceptor,
		mapping:     mapping,
	})
	return c
}

// NoContent 执行无响应体处理函数。
func NoContent(fn func(ctx *arkweb.Context) error) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := fn(ctx); err != nil {
			return nil, err
		}
		return responseStatusResult{statusCode: resolveResponseStatus(ctx, 0, http.StatusNoContent)}, nil
	})
}

func cloneConditions(conditions Conditions) Conditions {
	return Conditions{
		Consumes: append([]string(nil), conditions.Consumes...),
		Produces: append([]string(nil), conditions.Produces...),
		Params:   append([]string(nil), conditions.Params...),
		Headers:  append([]string(nil), conditions.Headers...),
	}
}

// Conditions 描述 MVC 路由额外匹配条件。
type Conditions struct {
	Consumes []string
	Produces []string
	Params   []string
	Headers  []string
}

type conditionExpression struct {
	name        string
	value       string
	hasValue    bool
	negated     bool
	valueNegate bool
}

func (c Controller) skipImplicitRoute(route Route, groups map[uint64]struct{}) bool {
	if !route.implicitMethods || len(c.methods) == 0 || route.methodGroupID == 0 {
		return false
	}
	_, exists := groups[route.methodGroupID]
	return exists
}

// WithProduces 设置请求 Accept 条件。
func WithProduces(mediaTypes ...string) RouteOption {
	copied := cleanRouteValues(mediaTypes)
	return func(route *Route) {
		route.Conditions.Produces = copied
	}
}

// WithModelAttributes 追加全局模型初始化器，对齐 Spring ControllerAdvice @ModelAttribute。
func (a ControllerAdvice) WithModelAttributes(
	initializers ...ModelAttributeInitializer) ControllerAdvice {
	a.models = append(a.models, initializers...)
	return a
}

// RegisterHandlerInterceptor 注册 MVC HandlerInterceptor 配置贡献点。
func RegisterHandlerInterceptor(registry *container.Registry, name string,
	interceptor HandlerInterceptor, options ...container.Option) error {
	return goweb.RegisterInterceptor(registry, name, HandlerInterceptorAdapter(interceptor),
		options...)
}

func entityResult[T any](ctx *arkweb.Context, entity goweb.ResponseEntity[T]) arkweb.Result {
	if mediaType, ok := selectedProducesMediaType(ctx); ok && !entity.HasMediaTypes() {
		return entity.WithContentType(mediaType)
	}
	return entity
}

// WithHandlerInterceptors 添加 MVC 全局处理器拦截器。
func (c Configurer) WithHandlerInterceptors(interceptors ...HandlerInterceptor) Configurer {
	c.handlerInterceptors = append(c.handlerInterceptors, interceptors...)
	return c
}

// WithResponseAdvice 追加响应写出前增强器，对齐 Spring ControllerAdvice ResponseBodyAdvice。
func (a ControllerAdvice) WithResponseAdvice(advice ...goweb.ResponseAdvice) ControllerAdvice {
	a.responseAdvice = append(a.responseAdvice, advice...)
	return a
}

// WithProduces 设置控制器级 Accept 条件。
func (c Controller) WithProduces(mediaTypes ...string) Controller {
	c.conditions.Produces = cleanRouteValues(mediaTypes)
	return c
}

// WithPathPrefixes 设置控制器级路径前缀，对齐 Spring 类型级 @RequestMapping path。
func (c Controller) WithPathPrefixes(prefixes ...string) Controller {
	c.pathPrefixes = cleanControllerPathPrefixes(prefixes)
	return c
}

// Order 返回配置顺序。
func (c Configuration) Order() int {
	return c.order
}

// NewConfigurer 创建 MVC Web 配置器。
func NewConfigurer(controllers ...Controller) Configurer {
	return Configurer{controllers: append([]Controller(nil), controllers...)}
}

// RequestBodyAdvice 返回请求体读取增强器快照。
func (a ControllerAdvice) RequestBodyAdvice() []goweb.RequestBodyAdvice {
	return append([]goweb.RequestBodyAdvice(nil), a.requestBodyAdvice...)
}

// InitBinders 返回控制器级绑定器初始化器快照。
func (c Controller) InitBinders() []BinderInitializer {
	return append([]BinderInitializer(nil), c.binders...)
}

// Kind 返回控制器默认返回值策略。
func (c Controller) Kind() ControllerKind {
	return c.kind
}

// BindJSON 绑定并校验 JSON 请求体，再将返回值写为 JSON 响应。
func BindJSON[In any, Out any](statusCode int, fn BindFunc[In, Out]) arkweb.Handler {
	return bindJSON(statusCode, fn, nil)
}

// GET 创建 GET 路由描述。
func GET(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodGet, pattern, handler, options...)
}

// DELETE 创建 DELETE 路由描述。
func DELETE(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodDelete, pattern, handler, options...)
}

func (c Conditions) specificity() int {
	return len(c.Params)*8 + len(c.Headers)*8 + len(c.Consumes)*4 + len(c.Produces)*4
}

// PostHandleFunc 在处理器成功返回后调整响应结果。
type PostHandleFunc func(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error)

// BindFunc 表示绑定并校验请求体后返回普通值的处理函数。
type BindFunc[In any, Out any] func(ctx *arkweb.Context, input In) (Out, error)
