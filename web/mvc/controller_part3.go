package mvc

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	servletmultipart "goark.dev/arkarta/servlet/multipart"
	appcontext "goark.dev/goark/context"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/core/util"
)

func handleWithInitBinders(ctx *arkweb.Context, handler arkweb.Handler,
	initializers []BinderInitializer) (arkweb.Result, error) {
	if handler == nil {
		return nil, arkweb.ErrNilHandler
	}
	if ctx == nil || ctx.Request() == nil {
		return handler.Handle(ctx)
	}
	base := ConversionServiceFromContext(ctx)
	scoped, err := base.Clone()
	if err != nil {
		return nil, err
	}
	binder := newDataBinder(scoped)
	if parent, ok := dataBinderFromContext(ctx); ok {
		binder.inheritFieldRules(parent)
	}
	request := ctx.Request()
	previous, existed := request.Attribute(AttributeConversionService)
	request.SetAttribute(AttributeConversionService, binder.ConversionService())
	defer restoreRequestAttribute(ctx, AttributeConversionService, previous, existed)
	previousBinder, binderExisted := request.Attribute(attributeDataBinder)
	request.SetAttribute(attributeDataBinder, binder)
	defer restoreRequestAttribute(ctx, attributeDataBinder, previousBinder, binderExisted)
	for _, initializer := range initializers {
		if initializer == nil {
			return nil, ErrNilBinderInitializer
		}
		if err := initializer.InitializeBinder(ctx, binder); err != nil {
			return nil, err
		}
	}
	return handler.Handle(ctx)
}

// HandlerInterceptorAdapter 将 MVC HandlerInterceptor 适配为底层 Web Interceptor。
func HandlerInterceptorAdapter(interceptor HandlerInterceptor) arkweb.Interceptor {
	if util.IsNil(interceptor) {
		return nil
	}
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (
		result arkweb.Result, err error) {
		proceed, err := interceptor.PreHandle(ctx)
		if err != nil || !proceed {
			return nil, err
		}
		defer func() {
			interceptor.AfterCompletion(ctx, err)
		}()
		result, err = next.Handle(ctx)
		if err != nil {
			return nil, err
		}
		return interceptor.PostHandle(ctx, result)
	})
}

func conditionErrorRank(err error) int {
	var statusErr servlet.StatusError
	if !errors.As(err, &statusErr) {
		return 1
	}
	switch statusErr.StatusCode() {
	case http.StatusUnsupportedMediaType:
		return 4
	case http.StatusNotAcceptable:
		return 3
	case http.StatusBadRequest:
		return 2
	default:
		return 1
	}
}

func mergeControllerRouteConditions(controller Conditions, route Conditions) Conditions {
	out := cloneConditions(route)
	if len(out.Consumes) == 0 {
		out.Consumes = append([]string(nil), controller.Consumes...)
	}
	if len(out.Produces) == 0 {
		out.Produces = append([]string(nil), controller.Produces...)
	}
	if len(controller.Params) > 0 {
		out.Params = append(append([]string(nil), controller.Params...), out.Params...)
	}
	if len(controller.Headers) > 0 {
		out.Headers = append(append([]string(nil), controller.Headers...), out.Headers...)
	}
	return out
}

// ControllerAdviceKindFromContext 返回当前执行中的 advice 默认返回值策略。
func ControllerAdviceKindFromContext(ctx *arkweb.Context) ControllerKind {
	if ctx == nil || ctx.Request() == nil {
		return ControllerKindView
	}
	value, ok := ctx.Request().Attribute(AttributeControllerAdviceKind)
	if !ok {
		return ControllerKindView
	}
	kind, ok := value.(ControllerKind)
	if !ok {
		return ControllerKindView
	}
	return kind
}

func bindMultipart[In any, Out any](statusCode int, fn BindFunc[In, Out], groups []string,
	options ...servletmultipart.Option) arkweb.Handler {
	validationGroups := cloneValidationGroups(groups)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		input, err := MultipartGroups[In](ctx, validationGroups, options...)
		if err != nil {
			return nil, err
		}
		value, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

// RegisterWithContext 注册 MVC Web 配置器 Bean。
func (c Configuration) RegisterWithContext(_ context.Context,
	config appcontext.ConfigurationContext) error {
	return goweb.RegisterConfigurer(
		config.Registry(),
		c.Name()+".configurer",
		NewConfigurer(c.controllers...).
			WithExceptionHandlers(c.exceptionHandlers...).
			WithHandlerInterceptors(c.handlerInterceptors...).
			WithControllerAdvices(c.advices...).
			withMappedHandlerInterceptors(c.mappedHandlerInterceptors...),
		container.WithOrder(c.order),
	)
}

func initBinderInterceptor(initializers []BinderInitializer) arkweb.Interceptor {
	if len(initializers) == 0 {
		return nil
	}
	copied := append([]BinderInitializer(nil), initializers...)
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result,
		error) {
		if next == nil {
			return nil, arkweb.ErrNilHandler
		}
		return handleWithInitBinders(ctx, next, copied)
	})
}

func crossOriginMethods(method string) []string {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case http.MethodGet:
		return []string{http.MethodGet, http.MethodHead}
	default:
		if method == "" {
			return nil
		}
		return []string{method}
	}
}

// RegisterWithContext 注册 advice Web 配置器 Bean。
func (a ControllerAdvice) RegisterWithContext(_ context.Context,
	config appcontext.ConfigurationContext) error {
	return goweb.RegisterConfigurer(
		config.Registry(),
		a.Name()+".configurer",
		a,
		container.WithOrder(a.order),
	)
}

// Text 将字符串写为文本响应。
func Text(statusCode int, fn func(ctx *arkweb.Context) (string, error)) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return arkweb.Text(statusCode, value), nil
	})
}

// NewHandlerInterceptor 创建函数型 HandlerInterceptor。
func NewHandlerInterceptor(pre PreHandleFunc, post PostHandleFunc,
	after AfterCompletionFunc) HandlerInterceptor {
	return HandlerInterceptorFuncs{
		PreHandleFunc:       pre,
		PostHandleFunc:      post,
		AfterCompletionFunc: after,
	}
}

// NewControllerAdvice 创建普通 MVC advice，字符串返回值默认解析为逻辑视图名。
func NewControllerAdvice(name string, handlers ...goweb.ErrorMapper) ControllerAdvice {
	return ControllerAdvice{
		name:     name,
		kind:     ControllerKindView,
		handlers: append([]goweb.ErrorMapper(nil), handlers...),
	}
}

func conditionSignaturePart(values []string) string {
	if len(values) == 0 {
		return ""
	}
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return strings.Join(copied, "\x00")
}

// NewConfiguration 创建 MVC 配置单元。
func NewConfiguration(name string, controllers ...Controller) Configuration {
	return Configuration{
		name:        name,
		controllers: append([]Controller(nil), controllers...),
	}
}

// PreHandle 执行前置处理；未设置时默认继续。
func (f HandlerInterceptorFuncs) PreHandle(ctx *arkweb.Context) (bool, error) {
	if f.PreHandleFunc == nil {
		return true, nil
	}
	return f.PreHandleFunc(ctx)
}

// WithHeaders 设置请求头条件，支持 name、!name、name=value、name!=value。
func WithHeaders(expressions ...string) RouteOption {
	copied := cleanRouteValues(expressions)
	return func(route *Route) {
		route.Conditions.Headers = copied
	}
}

func registerRouteCORS(registry *goweb.Registry, controller Controller, route Route) error {
	if config, ok := controller.crossOriginFor(route); ok {
		return registry.AddCORSMapping(route.Pattern, crossOriginMethods(route.Method), *config)
	}
	return nil
}

// HandlerInterceptorFuncs 用函数组快速声明 HandlerInterceptor。
type HandlerInterceptorFuncs struct {
	PreHandleFunc       PreHandleFunc
	PostHandleFunc      PostHandleFunc
	AfterCompletionFunc AfterCompletionFunc
}

// WithOrder 设置配置单元顺序。
func (c Configuration) WithOrder(order int) Configuration {
	c.order = order
	return c
}

// WithHandlerInterceptors 添加 MVC 全局处理器拦截器。
func (c Configuration) WithHandlerInterceptors(interceptors ...HandlerInterceptor) Configuration {
	c.handlerInterceptors = append(c.handlerInterceptors, interceptors...)
	return c
}

// WithControllerAdvices 添加 MVC 全局 advice。
func (c Configurer) WithControllerAdvices(advices ...ControllerAdvice) Configurer {
	c.advices = append(c.advices, advices...)
	return c
}

// WithRequestMethods 设置控制器级 HTTP method 条件。
func (c Controller) WithRequestMethods(methods ...string) Controller {
	c.methods = normalizeExplicitRequestMethods(methods)
	return c
}

// WithHeaders 设置控制器级请求头条件。
func (c Controller) WithHeaders(expressions ...string) Controller {
	c.conditions.Headers = cleanRouteValues(expressions)
	return c
}

// BindMultipart 绑定并校验 multipart/form-data 请求体，再将返回值写为 JSON 响应。
func BindMultipart[In any, Out any](statusCode int, fn BindFunc[In, Out],
	options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipart(statusCode, fn, nil, options...)
}

// Order 返回 advice 顺序。
func (a ControllerAdvice) Order() int {
	return a.order
}

// InitBinders 返回全局绑定器初始化器快照。
func (a ControllerAdvice) InitBinders() []BinderInitializer {
	return append([]BinderInitializer(nil), a.binders...)
}

type routeRegistration struct {
	handler    arkweb.Handler
	conditions Conditions
	owner      string
}

// ModelAttributes 返回控制器级模型初始化器快照。
func (c Controller) ModelAttributes() []ModelAttributeInitializer {
	return append([]ModelAttributeInitializer(nil), c.modelAttrs...)
}

// BindEntity 绑定并校验 JSON 请求体，再写出响应实体。
func BindEntity[In any, Out any](fn BindEntityFunc[In, Out]) arkweb.Handler {
	return bindEntity(fn, nil)
}

// POST 创建 POST 路由描述。
func POST(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodPost, pattern, handler, options...)
}

// TRACE 创建 TRACE 路由描述。
func TRACE(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodTrace, pattern, handler, options...)
}

func normalizeSingleRouteMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

// ResultFunc 表示直接返回 Arkarta Web Result 的处理函数。
type ResultFunc func(ctx *arkweb.Context) (arkweb.Result, error)

var requestMappingSequence atomic.Uint64
