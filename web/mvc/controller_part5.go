package mvc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	appcontext "goark.dev/goark/context"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/web/cors"
)

func (h conditionDispatchHandler) Handle(ctx *arkweb.Context) (arkweb.Result, error) {
	bestIndex := -1
	bestScore := -1
	var selectedErr error
	for i := range h.registrations {
		registration := h.registrations[i]
		if err := registration.conditions.match(ctx); err != nil {
			selectedErr = selectConditionError(selectedErr, err)
			continue
		}
		score := registration.conditions.specificity()
		if bestIndex < 0 || score > bestScore {
			bestIndex = i
			bestScore = score
		}
	}
	if bestIndex < 0 {
		if selectedErr != nil {
			return nil, selectedErr
		}
		return nil, servlet.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil)
	}
	registration := h.registrations[bestIndex]
	if err := registration.conditions.match(ctx); err != nil {
		return nil, err
	}
	return registration.handler.Handle(ctx)
}

func parseConditionExpression(text string) (conditionExpression, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return conditionExpression{}, false
	}
	if name, value, ok := strings.Cut(text, "!="); ok {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		return conditionExpression{name: name, value: value, hasValue: true, valueNegate: true},
			name != ""
	}
	if name, value, ok := strings.Cut(text, "="); ok {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		return conditionExpression{name: name, value: value, hasValue: true}, name != ""
	}
	if strings.HasPrefix(text, "!") {
		name := strings.TrimSpace(strings.TrimPrefix(text, "!"))
		return conditionExpression{name: name, negated: true}, name != ""
	}
	return conditionExpression{name: text}, true
}

func (c Controller) routeMethods(route Route) []string {
	if len(c.methods) == 0 {
		return []string{route.Method}
	}
	if route.implicitMethods {
		return c.methods
	}
	methods := make([]string, 0, len(c.methods)+1)
	if method := normalizeSingleRouteMethod(route.Method); method != "" {
		methods = append(methods, method)
	}
	for _, method := range c.methods {
		if !hasRequestMethod(methods, method) {
			methods = append(methods, method)
		}
	}
	return methods
}

func matchParameterExpressions(req *servlet.Request, expressions []string) error {
	for _, text := range expressions {
		expr, ok := parseConditionExpression(text)
		if !ok {
			continue
		}
		value, exists, err := req.Parameter(expr.name)
		if err != nil {
			return err
		}
		if !expr.matches(value, exists) {
			return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("请求参数条件不匹配 %q", text), nil)
		}
	}
	return nil
}

func bindBodyEntity[In any, Out any](fn BindEntityFunc[In, Out], groups []string,
	mediaTypes []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	readMediaTypes := cleanRouteValues(mediaTypes)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateBody(ctx, &input, validationGroups, readMediaTypes); err != nil {
			return nil, err
		}
		entity, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}

func bindJSON[In any, Out any](statusCode int, fn BindFunc[In, Out],
	groups []string) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		var input In
		if err := bindAndValidateJSON(ctx, &input, validationGroups); err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

func (c Conditions) wrap(handler arkweb.Handler) arkweb.Handler {
	if handler == nil {
		return nil
	}
	if c.empty() {
		return handler
	}
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := c.match(ctx); err != nil {
			return nil, err
		}
		return handler.Handle(ctx)
	})
}

func matchHeaderExpressions(req *servlet.Request, expressions []string) error {
	for _, text := range expressions {
		expr, ok := parseConditionExpression(text)
		if !ok {
			continue
		}
		value, exists := req.HeaderValue(expr.name)
		if !expr.matches(value, exists) {
			return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("请求头条件不匹配 %q", text), nil)
		}
	}
	return nil
}

func rejectAmbiguousRouteConditions(key routeRegistrationKey,
	registrations []routeRegistration) error {
	seen := make(map[string]struct{}, len(registrations))
	for _, registration := range registrations {
		signature := conditionSignature(registration.conditions)
		if _, exists := seen[signature]; exists {
			return fmt.Errorf("goark/web/mvc: ambiguous route conditions for %s %s", key.method, key.pattern)
		}
		seen[signature] = struct{}{}
	}
	return nil
}

// Route 描述 MVC 路由。
type Route struct {
	Method     string
	Pattern    string
	Handler    arkweb.Handler
	Conditions Conditions

	crossOrigin     *cors.Config
	implicitMethods bool
	methodGroupID   uint64
}

// Entity 将响应实体处理函数适配为 Arkarta Web Handler。
func Entity[T any](fn EntityFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		entity, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return entityResult(ctx, entity), nil
	})
}

func wrapInitBinders(handler arkweb.Handler, initializers []BinderInitializer) arkweb.Handler {
	if handler == nil || len(initializers) == 0 {
		return handler
	}
	copied := append([]BinderInitializer(nil), initializers...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return handleWithInitBinders(ctx, handler, copied)
	})
}

func hasRequestMethod(methods []string, method string) bool {
	method = normalizeSingleRouteMethod(method)
	for _, item := range methods {
		if item == method {
			return true
		}
	}
	return false
}

// PostHandle 执行后置处理；未设置时保留原响应结果。
func (f HandlerInterceptorFuncs) PostHandle(ctx *arkweb.Context, result arkweb.Result) (
	arkweb.Result, error) {
	if f.PostHandleFunc == nil {
		return result, nil
	}
	return f.PostHandleFunc(ctx, result)
}

// Name 返回 advice 配置名称。
func (a ControllerAdvice) Name() string {
	if a.name == "" {
		return "goark.web.mvc.controller-advice"
	}
	return a.name
}

// WithConsumes 设置请求 Content-Type 条件。
func WithConsumes(mediaTypes ...string) RouteOption {
	copied := cleanRouteValues(mediaTypes)
	return func(route *Route) {
		route.Conditions.Consumes = copied
	}
}

// NewRestControllerAdvice 创建 REST MVC advice，普通返回值默认写入响应体。
func NewRestControllerAdvice(name string, handlers ...goweb.ErrorMapper) ControllerAdvice {
	advice := NewControllerAdvice(name, handlers...)
	advice.kind = ControllerKindREST
	return advice
}

// WithRequestBodyAdvice 追加请求体读取增强器，对齐 Spring ControllerAdvice RequestBodyAdvice。
func (a ControllerAdvice) WithRequestBodyAdvice(
	advice ...goweb.RequestBodyAdvice) ControllerAdvice {
	a.requestBodyAdvice = append(a.requestBodyAdvice, advice...)
	return a
}

// NewRestController 创建 REST 控制器描述。
func NewRestController(name string, routes ...Route) Controller {
	controller := NewController(name, routes...)
	controller.kind = ControllerKindREST
	return controller
}

// Handler 将 ResultFunc 适配为 Arkarta Web Handler。
func Handler(fn ResultFunc) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return fn(ctx)
	})
}

// WithExceptionHandlers 添加 MVC 全局异常处理器。
func (c Configurer) WithExceptionHandlers(handlers ...goweb.ErrorMapper) Configurer {
	c.exceptionHandlers = append(c.exceptionHandlers, handlers...)
	return c
}

// WithExceptionHandlers 追加全局异常处理器。
func (a ControllerAdvice) WithExceptionHandlers(handlers ...goweb.ErrorMapper) ControllerAdvice {
	a.handlers = append(a.handlers, handlers...)
	return a
}

// WithConsumes 设置控制器级 Content-Type 条件。
func (c Controller) WithConsumes(mediaTypes ...string) Controller {
	c.conditions.Consumes = cleanRouteValues(mediaTypes)
	return c
}

// WithModelAttributes 设置控制器级模型初始化器，对齐 Spring 方法级 @ModelAttribute。
func (c Controller) WithModelAttributes(initializers ...ModelAttributeInitializer) Controller {
	c.modelAttrs = append([]ModelAttributeInitializer(nil), initializers...)
	return c
}

const (
	// AttributeProducesMediaType 保存 MVC produces 条件协商后的媒体类型。
	AttributeProducesMediaType = "goark.web.mvc.produces.media_type"
)

type mappedHandlerInterceptor struct {
	interceptor HandlerInterceptor
	mapping     goweb.InterceptorMapping
}

// ExceptionHandlers 返回异常处理器快照。
func (a ControllerAdvice) ExceptionHandlers() []goweb.ErrorMapper {
	return append([]goweb.ErrorMapper(nil), a.handlers...)
}

// Register 注册 advice Web 配置器 Bean。
func (a ControllerAdvice) Register(ctx context.Context, registry *container.Registry) error {
	return a.RegisterWithContext(ctx, appcontext.NewConfigurationContext(nil, registry))
}

// Name 返回控制器名称。
func (c Controller) Name() string {
	return c.name
}

// Methods 返回控制器级 HTTP method 限定快照。
func (c Controller) Methods() []string {
	return append([]string(nil), c.methods...)
}

// RequestMapping 创建无 HTTP method 限定的 MVC 路由集合，对齐 Spring @RequestMapping 默认语义。
func RequestMapping(pattern string, handler arkweb.Handler, options ...RouteOption) []Route {
	return RequestMappingMethods(nil, pattern, handler, options...)
}

// PATCH 创建 PATCH 路由描述。
func PATCH(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPatch, pattern, handler, options...)
}

func (c Conditions) empty() bool {
	return len(c.Consumes) == 0 && len(c.Produces) == 0 && len(c.Params) == 0 && len(c.Headers) == 0
}

// PreHandleFunc 在处理器执行前决定请求是否继续。
type PreHandleFunc func(ctx *arkweb.Context) (bool, error)

// EntityFunc 表示返回 Goark 响应实体的处理函数。
type EntityFunc[T any] func(ctx *arkweb.Context) (goweb.ResponseEntity[T], error)
