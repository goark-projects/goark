package mvc

import (
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/cors"
)

const (
	// AttributeControllerKind 保存当前请求命中的 MVC 控制器类型。
	AttributeControllerKind = "goark.web.mvc.controller.kind"
)

// ControllerKind 表示控制器默认返回值策略。
type ControllerKind uint8

const (
	// ControllerKindView 表示普通 Controller，字符串默认解析为逻辑视图名。
	ControllerKindView ControllerKind = iota
	// ControllerKindREST 表示 REST Controller，普通返回值默认写为响应体。
	ControllerKindREST
)

// Controller 描述一组同属一个控制器的路由。
type Controller struct {
	name        string
	routes      []Route
	kind        ControllerKind
	methods     []string
	conditions  Conditions
	crossOrigin *cors.Config
}

// NewController 创建控制器描述。
func NewController(name string, routes ...Route) Controller {
	return Controller{
		name:   name,
		routes: append([]Route(nil), routes...),
		kind:   ControllerKindView,
	}
}

// NewRestController 创建 REST 控制器描述。
func NewRestController(name string, routes ...Route) Controller {
	controller := NewController(name, routes...)
	controller.kind = ControllerKindREST
	return controller
}

// Name 返回控制器名称。
func (c Controller) Name() string {
	return c.name
}

// Kind 返回控制器默认返回值策略。
func (c Controller) Kind() ControllerKind {
	return c.kind
}

// Routes 返回控制器路由快照。
func (c Controller) Routes() []Route {
	return append([]Route(nil), c.routes...)
}

// Methods 返回控制器级 HTTP method 限定快照。
func (c Controller) Methods() []string {
	return append([]string(nil), c.methods...)
}

// WithCrossOrigin 设置控制器级 CORS 策略，对齐 Spring 类级 @CrossOrigin。
func (c Controller) WithCrossOrigin(config cors.Config) Controller {
	c.crossOrigin = cloneCrossOriginConfig(config)
	return c
}

// Register 注册控制器路由。
func (c Controller) Register(registry *goweb.Registry) error {
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	var implicitGroups map[uint64]struct{}
	for _, route := range c.routes {
		if c.skipImplicitRoute(route, implicitGroups) {
			continue
		}
		if route.implicitMethods && len(c.methods) > 0 && route.methodGroupID != 0 {
			if implicitGroups == nil {
				implicitGroups = make(map[uint64]struct{})
			}
			implicitGroups[route.methodGroupID] = struct{}{}
		}
		for _, method := range c.routeMethods(route) {
			resolvedRoute := route
			resolvedRoute.Method = method
			if err := c.registerRoute(registry, resolvedRoute); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Controller) registerRoute(registry *goweb.Registry, route Route) error {
	if config, ok := c.crossOriginFor(route); ok {
		if err := registry.AddCORSMapping(route.Pattern, crossOriginMethods(route.Method), *config); err != nil {
			return err
		}
	}
	conditions := mergeControllerRouteConditions(c.conditions, route.Conditions)
	handler := conditions.wrap(bindControllerKind(c.kind, route.Handler))
	return registry.Handle(route.Method, route.Pattern, handler)
}

func (c Controller) skipImplicitRoute(route Route, groups map[uint64]struct{}) bool {
	if !route.implicitMethods || len(c.methods) == 0 || route.methodGroupID == 0 {
		return false
	}
	_, exists := groups[route.methodGroupID]
	return exists
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

func (c Controller) crossOriginFor(route Route) (*cors.Config, bool) {
	if route.crossOrigin != nil {
		return route.crossOrigin, true
	}
	if c.crossOrigin != nil {
		return c.crossOrigin, true
	}
	return nil, false
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
