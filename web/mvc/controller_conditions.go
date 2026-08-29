package mvc

// WithConditions 设置控制器级请求映射条件。
func (c Controller) WithConditions(conditions Conditions) Controller {
	c.conditions = cloneConditions(conditions)
	return c
}

// WithConsumes 设置控制器级 Content-Type 条件。
func (c Controller) WithConsumes(mediaTypes ...string) Controller {
	c.conditions.Consumes = cleanRouteValues(mediaTypes)
	return c
}

// WithProduces 设置控制器级 Accept 条件。
func (c Controller) WithProduces(mediaTypes ...string) Controller {
	c.conditions.Produces = cleanRouteValues(mediaTypes)
	return c
}

// WithParams 设置控制器级请求参数条件。
func (c Controller) WithParams(expressions ...string) Controller {
	c.conditions.Params = cleanRouteValues(expressions)
	return c
}

// WithHeaders 设置控制器级请求头条件。
func (c Controller) WithHeaders(expressions ...string) Controller {
	c.conditions.Headers = cleanRouteValues(expressions)
	return c
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

func cloneConditions(conditions Conditions) Conditions {
	return Conditions{
		Consumes: append([]string(nil), conditions.Consumes...),
		Produces: append([]string(nil), conditions.Produces...),
		Params:   append([]string(nil), conditions.Params...),
		Headers:  append([]string(nil), conditions.Headers...),
	}
}
