package mvc

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strings"

	appcontext "goark.dev/goark/context"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/core/util"
	"goark.dev/goark/web/cors"
)

// ConfigureWeb 注册 advice 异常处理链。
func (a ControllerAdvice) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	if len(a.binders) > 0 {
		registry.Use(initBinderInterceptor(a.binders))
	}
	if len(a.models) > 0 {
		registry.Use(modelAttributeInterceptor(a.models))
	}
	for _, advice := range a.requestBodyAdvice {
		if util.IsNil(advice) {
			continue
		}
		registry.UseRequestBodyAdvice(advice)
	}
	for _, advice := range a.responseAdvice {
		if util.IsNil(advice) {
			continue
		}
		registry.UseResponseAdvice(advice)
	}
	for _, handler := range a.handlers {
		if util.IsNil(handler) {
			continue
		}
		registry.UseErrorMapper(a.wrapExceptionHandler(handler))
	}
	return nil
}

func bindControllerKind(kind ControllerKind, handler arkweb.Handler) arkweb.Handler {
	if handler == nil {
		return nil
	}
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if ctx == nil || ctx.Request() == nil {
			return handler.Handle(ctx)
		}
		request := ctx.Request()
		previous, existed := request.Attribute(AttributeControllerKind)
		request.SetAttribute(AttributeControllerKind, kind)
		defer func() {
			if existed {
				request.SetAttribute(AttributeControllerKind, previous)
				return
			}
			request.SetAttribute(AttributeControllerKind, nil)
		}()
		return handler.Handle(ctx)
	})
}

func matchConsumes(req *servlet.Request, consumes []string) error {
	if len(consumes) == 0 {
		return nil
	}
	contentType, _, err := mime.ParseMediaType(req.Header().Get("Content-Type"))
	if err != nil || contentType == "" {
		return servlet.NewHTTPError(http.StatusUnsupportedMediaType, http.StatusText(
			http.StatusUnsupportedMediaType), err)
	}
	for _, mediaType := range consumes {
		if mediaTypeMatches(mediaType, contentType) {
			return nil
		}
	}
	return servlet.NewHTTPError(http.StatusUnsupportedMediaType, http.StatusText(
		http.StatusUnsupportedMediaType), nil)
}

func joinControllerRoutePattern(prefix string, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "/"
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if prefix == "" {
		return pattern
	}
	if pattern == "/" {
		return prefix
	}
	return prefix + "/" + strings.TrimLeft(pattern, "/")
}

func cleanControllerPathPrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	seen := make(map[string]struct{}, len(prefixes))
	for _, value := range prefixes {
		for _, item := range strings.Split(value, ",") {
			prefix := normalizeControllerPathPrefix(item)
			if _, exists := seen[prefix]; exists {
				continue
			}
			seen[prefix] = struct{}{}
			out = append(out, prefix)
		}
	}
	return out
}

func selectedProducesMediaType(ctx *arkweb.Context) (string, bool) {
	if ctx == nil || ctx.Request() == nil {
		return "", false
	}
	value, ok := ctx.Request().Attribute(AttributeProducesMediaType)
	if !ok {
		return "", false
	}
	mediaType, ok := value.(string)
	if !ok {
		return "", false
	}
	mediaType = strings.TrimSpace(mediaType)
	return mediaType, mediaType != ""
}

func buildRouteRegistrationHandler(key routeRegistrationKey,
	registrations []routeRegistration) (arkweb.Handler, error) {
	if len(registrations) == 0 {
		return nil, fmt.Errorf("goark/web/mvc: empty route registration %s %s", key.method, key.pattern)
	}
	if len(registrations) == 1 {
		registration := registrations[0]
		return registration.conditions.wrap(registration.handler), nil
	}
	if err := rejectAmbiguousRouteConditions(key, registrations); err != nil {
		return nil, err
	}
	return newConditionDispatchHandler(registrations), nil
}

func matchProduces(req *servlet.Request, produces []string) error {
	if len(produces) == 0 {
		return nil
	}
	selected, ok := req.NegotiateContentType(produces...)
	if !ok {
		return servlet.NewHTTPError(http.StatusNotAcceptable, http.StatusText(
			http.StatusNotAcceptable), nil)
	}
	req.SetAttribute(AttributeProducesMediaType, selected)
	return nil
}

func cloneCrossOriginConfig(config cors.Config) *cors.Config {
	copied := cors.Config{
		AllowedOrigins:        append([]string(nil), config.AllowedOrigins...),
		AllowedOriginPatterns: append([]string(nil), config.AllowedOriginPatterns...),
		AllowedMethods:        append([]string(nil), config.AllowedMethods...),
		AllowedHeaders:        append([]string(nil), config.AllowedHeaders...),
		ExposedHeaders:        append([]string(nil), config.ExposedHeaders...),
		AllowCredentials:      config.AllowCredentials,
		MaxAge:                config.MaxAge,
	}
	return &copied
}

// Configuration 将 MVC 控制器注册为 Goark 配置单元。
type Configuration struct {
	name                      string
	order                     int
	controllers               []Controller
	advices                   []ControllerAdvice
	exceptionHandlers         []goweb.ErrorMapper
	handlerInterceptors       []HandlerInterceptor
	mappedHandlerInterceptors []mappedHandlerInterceptor
}

// JSON 将普通返回值写为 JSON 响应。
func JSON[T any](statusCode int, fn ValueFunc[T]) arkweb.Handler {
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		value, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResult(ctx, statusCode, value), nil
	})
}

func (c Controller) crossOriginFor(route Route) (*cors.Config, bool) {
	if route.crossOrigin != nil {
		return route.crossOrigin, true
	}
	if c.crossOrigin != nil {
		return c.crossOrigin, true
	}
	return nil, false
}

// Configurer 将 MVC 控制器注册到 Web 注册表。
type Configurer struct {
	controllers               []Controller
	advices                   []ControllerAdvice
	exceptionHandlers         []goweb.ErrorMapper
	handlerInterceptors       []HandlerInterceptor
	mappedHandlerInterceptors []mappedHandlerInterceptor
}

func conditionSignature(conditions Conditions) string {
	return strings.Join([]string{
		conditionSignaturePart(conditions.Consumes),
		conditionSignaturePart(conditions.Produces),
		conditionSignaturePart(conditions.Params),
		conditionSignaturePart(conditions.Headers),
	}, "\x01")
}

func mediaTypeMatches(pattern string, actual string) bool {
	mediaRange, ok := servlet.NewMediaType(pattern)
	if !ok {
		return false
	}
	return mediaRange.Matches(actual)
}

// Register 注册控制器路由。
func (c Controller) Register(registry *goweb.Registry) error {
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	return registerControllers(registry, []Controller{c})
}

// WithParams 设置请求参数条件，支持 name、!name、name=value、name!=value。
func WithParams(expressions ...string) RouteOption {
	copied := cleanRouteValues(expressions)
	return func(route *Route) {
		route.Conditions.Params = copied
	}
}

func (c Controller) registrationPathPrefixes() []string {
	if len(c.pathPrefixes) == 0 {
		return []string{""}
	}
	return append([]string(nil), c.pathPrefixes...)
}

// HandlerInterceptor 提供 Spring MVC HandlerInterceptor 的 Go 化生命周期。
type HandlerInterceptor interface {
	PreHandle(ctx *arkweb.Context) (bool, error)
	PostHandle(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error)
	AfterCompletion(ctx *arkweb.Context, err error)
}

func normalizeRequestMappingMethods(methods []string) []string {
	if len(methods) == 0 {
		return defaultRequestMappingMethods[:]
	}
	return normalizeExplicitRequestMethods(methods)
}

func (c Configurer) withMappedHandlerInterceptors(
	interceptors ...mappedHandlerInterceptor) Configurer {
	c.mappedHandlerInterceptors = append(c.mappedHandlerInterceptors, interceptors...)
	return c
}

// WithInitBinders 追加全局绑定器初始化器，对齐 Spring ControllerAdvice @InitBinder。
func (a ControllerAdvice) WithInitBinders(initializers ...BinderInitializer) ControllerAdvice {
	a.binders = append(a.binders, initializers...)
	return a
}

// WithParams 设置控制器级请求参数条件。
func (c Controller) WithParams(expressions ...string) Controller {
	c.conditions.Params = cleanRouteValues(expressions)
	return c
}

// WithCrossOrigin 设置控制器级 CORS 策略，对齐 Spring 类级 @CrossOrigin。
func (c Controller) WithCrossOrigin(config cors.Config) Controller {
	c.crossOrigin = cloneCrossOriginConfig(config)
	return c
}

// Register 注册 MVC Web 配置器 Bean。
func (c Configuration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, appcontext.NewConfigurationContext(nil, registry))
}

const (
	// AttributeControllerAdviceKind 保存当前 MVC advice 的默认返回值策略。
	AttributeControllerAdviceKind = "goark.web.mvc.controller_advice.kind"
)

// ResponseAdvice 返回响应增强器快照。
func (a ControllerAdvice) ResponseAdvice() []goweb.ResponseAdvice {
	return append([]goweb.ResponseAdvice(nil), a.responseAdvice...)
}

type routeRegistrationKey struct {
	method  string
	pattern string
}

// Routes 返回控制器路由快照。
func (c Controller) Routes() []Route {
	return append([]Route(nil), c.routes...)
}

// BindBody 按 Content-Type 绑定并校验请求体，再将返回值写为 JSON 响应。
func BindBody[In any, Out any](statusCode int, fn BindFunc[In, Out]) arkweb.Handler {
	return bindBody(statusCode, fn, nil, nil)
}

// HEAD 创建 HEAD 路由描述。
func HEAD(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodHead, pattern, handler, options...)
}

// OPTIONS 创建 OPTIONS 路由描述。
func OPTIONS(pattern string, handler arkweb.Handler, options ...RouteOption) Route {
	return Handle(http.MethodOptions, pattern, handler, options...)
}

// BindEntityFunc 表示绑定并校验请求体后返回 Goark 响应实体的处理函数。
type BindEntityFunc[In any, Out any] func(ctx *arkweb.Context, input In) (
	goweb.ResponseEntity[Out], error)

// AfterCompletionFunc 在处理器链完成后接收处理错误。
type AfterCompletionFunc func(ctx *arkweb.Context, err error)

// RouteOption 定制 MVC 路由描述。
type RouteOption func(*Route)
