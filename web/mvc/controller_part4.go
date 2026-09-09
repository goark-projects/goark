package mvc

import (
	"context"
	"net/http"
	"strings"

	servletmultipart "goark.dev/arkarta/servlet/multipart"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/goark/core/util"
	"goark.dev/goark/web/cors"
)

// ConfigureWeb 注册控制器路由。
func (c Configurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	for _, interceptor := range c.handlerInterceptors {
		if util.IsNil(interceptor) {
			continue
		}
		registry.Use(HandlerInterceptorAdapter(interceptor))
	}
	for _, registration := range c.mappedHandlerInterceptors {
		if util.IsNil(registration.interceptor) {
			continue
		}
		registry.UseMapped(HandlerInterceptorAdapter(registration.interceptor), registration.mapping)
	}
	for _, handler := range c.exceptionHandlers {
		registry.UseErrorMapper(handler)
	}
	for _, advice := range c.advices {
		if err := advice.ConfigureWeb(ctx, registry); err != nil {
			return err
		}
	}
	return registerControllers(registry, c.controllers)
}

func registerControllers(registry *goweb.Registry, controllers []Controller) error {
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	groups := make(map[routeRegistrationKey][]routeRegistration)
	keys := make([]routeRegistrationKey, 0)
	for _, controller := range controllers {
		if err := appendControllerRegistrations(registry, controller, groups, &keys); err != nil {
			return err
		}
	}
	for _, key := range keys {
		handler, err := buildRouteRegistrationHandler(key, groups[key])
		if err != nil {
			return err
		}
		owners := make([]string, 0, len(groups[key]))
		for _, registration := range groups[key] {
			owners = append(owners, registration.owner)
		}
		if err := registry.HandleOwned(key.method, key.pattern, handler, owners...); err != nil {
			return err
		}
	}
	return nil
}

// RequestMappingMethods 创建限定到指定 HTTP method 集合的 MVC 路由。
func RequestMappingMethods(methods []string, pattern string, handler arkweb.Handler,
	options ...RouteOption) []Route {
	implicitMethods := len(methods) == 0
	normalized := normalizeRequestMappingMethods(methods)
	groupID := uint64(0)
	if implicitMethods {
		groupID = requestMappingSequence.Add(1)
	}
	routes := make([]Route, 0, len(normalized))
	for _, method := range normalized {
		route := Handle(method, pattern, handler, options...)
		route.implicitMethods = implicitMethods
		route.methodGroupID = groupID
		routes = append(routes, route)
	}
	return routes
}

func (e conditionExpression) matches(actual string, exists bool) bool {
	if e.negated {
		return !exists
	}
	if !exists {
		return false
	}
	if !e.hasValue {
		return true
	}
	matched := actual == e.value
	if e.valueNegate {
		return !matched
	}
	return matched
}

func normalizeExplicitRequestMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		method = normalizeSingleRouteMethod(method)
		if method == "" {
			continue
		}
		if _, exists := seen[method]; exists {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	return out
}

func bindMultipartEntity[In any, Out any](fn BindEntityFunc[In, Out], groups []string,
	options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := MultipartGroups[In](ctx, validationGroups, options...)
		if err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}

func normalizeControllerPathPrefix(prefix string) string {
	prefix = strings.TrimSpace(strings.ReplaceAll(prefix, "\\", "/"))
	if prefix == "" || prefix == "/" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return ""
	}
	return prefix
}

// Controller 描述一组同属一个控制器的路由。
type Controller struct {
	name         string
	routes       []Route
	kind         ControllerKind
	methods      []string
	conditions   Conditions
	crossOrigin  *cors.Config
	modelAttrs   []ModelAttributeInitializer
	sessionAttrs []string
	binders      []BinderInitializer
	pathPrefixes []string
}

// ControllerAdvice 描述一组全局 MVC 异常处理器、请求/响应增强器、模型初始化器和绑定器初始化器。
type ControllerAdvice struct {
	name              string
	order             int
	kind              ControllerKind
	handlers          []goweb.ErrorMapper
	requestBodyAdvice []goweb.RequestBodyAdvice
	responseAdvice    []goweb.ResponseAdvice
	binders           []BinderInitializer
	models            []ModelAttributeInitializer
}

func cleanRouteValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}
func selectConditionError(current error, candidate error) error {
	if current == nil {
		return candidate
	}
	if conditionErrorRank(candidate) > conditionErrorRank(current) {
		return candidate
	}
	return current
}

// WithMappedHandlerInterceptor 添加带路径映射的 MVC 处理器拦截器。
func (c Configurer) WithMappedHandlerInterceptor(interceptor HandlerInterceptor,
	mapping goweb.InterceptorMapping) Configurer {
	c.mappedHandlerInterceptors = append(c.mappedHandlerInterceptors, mappedHandlerInterceptor{
		interceptor: interceptor,
		mapping:     mapping,
	})
	return c
}

var defaultRequestMappingMethods = [...]string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// NewController 创建控制器描述。
func NewController(name string, routes ...Route) Controller {
	return Controller{
		name:   name,
		routes: append([]Route(nil), routes...),
		kind:   ControllerKindView,
	}
}

// Name 返回配置名称。
func (c Configuration) Name() string {
	if c.name == "" {
		return "goark.web.mvc"
	}
	return c.name
}

func jsonResult(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	statusCode = resolveResponseStatus(ctx, statusCode, http.StatusOK)
	if mediaType, ok := selectedProducesMediaType(ctx); ok {
		return goweb.Message(statusCode, value, mediaType)
	}
	return arkweb.JSON(statusCode, value)
}

// WithCrossOrigin 设置路由级 CORS 策略，对齐 Spring 方法级 @CrossOrigin。
func WithCrossOrigin(config cors.Config) RouteOption {
	copied := cloneCrossOriginConfig(config)
	return func(route *Route) {
		route.crossOrigin = copied
	}
}

const (
	// ControllerKindView 表示普通 Controller，字符串默认解析为逻辑视图名。
	ControllerKindView ControllerKind = iota
	// ControllerKindREST 表示 REST Controller，普通返回值默认写为响应体。
	ControllerKindREST
)

// AfterCompletion 执行完成回调；未设置时无操作。
func (f HandlerInterceptorFuncs) AfterCompletion(ctx *arkweb.Context, err error) {
	if f.AfterCompletionFunc != nil {
		f.AfterCompletionFunc(ctx, err)
	}
}

// WithExceptionHandlers 添加 MVC 全局异常处理器。
func (c Configuration) WithExceptionHandlers(handlers ...goweb.ErrorMapper) Configuration {
	c.exceptionHandlers = append(c.exceptionHandlers, handlers...)
	return c
}

// WithControllerAdvices 添加 MVC 全局 advice。
func (c Configuration) WithControllerAdvices(advices ...ControllerAdvice) Configuration {
	c.advices = append(c.advices, advices...)
	return c
}

// WithOrder 设置 advice 配置和 Web 注册顺序。
func (a ControllerAdvice) WithOrder(order int) ControllerAdvice {
	a.order = order
	return a
}

// WithConditions 设置控制器级请求映射条件。
func (c Controller) WithConditions(conditions Conditions) Controller {
	c.conditions = cloneConditions(conditions)
	return c
}

// WithInitBinders 设置控制器级绑定器初始化器，对齐 Spring 方法级 @InitBinder。
func (c Controller) WithInitBinders(initializers ...BinderInitializer) Controller {
	c.binders = append([]BinderInitializer(nil), initializers...)
	return c
}

// BindMultipartEntity 绑定并校验 multipart/form-data 请求体，再写出响应实体。
func BindMultipartEntity[In any, Out any](fn BindEntityFunc[In, Out],
	options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartEntity(fn, nil, options...)
}

// Kind 返回 advice 默认返回值策略。
func (a ControllerAdvice) Kind() ControllerKind {
	return a.kind
}

// ModelAttributes 返回全局模型初始化器快照。
func (a ControllerAdvice) ModelAttributes() []ModelAttributeInitializer {
	return append([]ModelAttributeInitializer(nil), a.models...)
}

const (
	// AttributeControllerKind 保存当前请求命中的 MVC 控制器类型。
	AttributeControllerKind = "goark.web.mvc.controller.kind"
)

// SessionAttributes 返回控制器级 Session 模型属性名快照。
func (c Controller) SessionAttributes() []string {
	return append([]string(nil), c.sessionAttrs...)
}

// BindBodyEntity 按 Content-Type 绑定并校验请求体，再写出响应实体。
func BindBodyEntity[In any, Out any](fn BindEntityFunc[In, Out]) arkweb.Handler {
	return bindBodyEntity(fn, nil, nil)
}

// PUT 创建 PUT 路由描述。
func PUT(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPut, pattern, handler, options...)
}

type conditionDispatchHandler struct {
	registrations []routeRegistration
}

func newConditionDispatchHandler(registrations []routeRegistration) arkweb.Handler {
	return conditionDispatchHandler{registrations: append([]routeRegistration(nil), registrations...)}
}

// ControllerKind 表示控制器默认返回值策略。
type ControllerKind uint8

// ValueFunc 表示返回普通值并由 MVC 写为 JSON 的处理函数。
type ValueFunc[T any] func(ctx *arkweb.Context) (T, error)
