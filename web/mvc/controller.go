package mvc

import goweb "goark.dev/goark/web"

// Controller 描述一组同属一个控制器的路由。
type Controller struct {
	name   string
	routes []Route
}

// NewController 创建控制器描述。
func NewController(name string, routes ...Route) Controller {
	return Controller{
		name:   name,
		routes: append([]Route(nil), routes...),
	}
}

// Name 返回控制器名称。
func (c Controller) Name() string {
	return c.name
}

// Routes 返回控制器路由快照。
func (c Controller) Routes() []Route {
	return append([]Route(nil), c.routes...)
}

// Register 注册控制器路由。
func (c Controller) Register(registry *goweb.Registry) error {
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	for _, route := range c.routes {
		if err := registry.Handle(route.Method, route.Pattern, route.Conditions.wrap(route.Handler)); err != nil {
			return err
		}
	}
	return nil
}
