package mvc

import arkweb "goark.dev/arkarta/web"

// WithInitBinders 设置控制器级绑定器初始化器，对齐 Spring 方法级 @InitBinder。
func (c Controller) WithInitBinders(initializers ...BinderInitializer) Controller {
	c.binders = append([]BinderInitializer(nil), initializers...)
	return c
}

// InitBinders 返回控制器级绑定器初始化器快照。
func (c Controller) InitBinders() []BinderInitializer {
	return append([]BinderInitializer(nil), c.binders...)
}

func wrapInitBinders(handler arkweb.Handler, initializers []BinderInitializer) arkweb.Handler {
	if handler == nil || len(initializers) == 0 {
		return handler
	}
	copied := append([]BinderInitializer(nil), initializers...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if ctx == nil || ctx.Request() == nil {
			return handler.Handle(ctx)
		}
		base := ConversionServiceFromContext(ctx)
		scoped, err := base.Clone()
		if err != nil {
			return nil, err
		}
		binder := newDataBinder(scoped)
		request := ctx.Request()
		previous, existed := request.Attribute(AttributeConversionService)
		request.SetAttribute(AttributeConversionService, binder.ConversionService())
		defer restoreConversionServiceAttribute(ctx, previous, existed)
		for _, initializer := range copied {
			if initializer == nil {
				return nil, ErrNilBinderInitializer
			}
			if err := initializer.InitializeBinder(ctx, binder); err != nil {
				return nil, err
			}
		}
		return handler.Handle(ctx)
	})
}

func restoreConversionServiceAttribute(ctx *arkweb.Context, previous any, existed bool) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	if existed {
		ctx.Request().SetAttribute(AttributeConversionService, previous)
		return
	}
	ctx.Request().SetAttribute(AttributeConversionService, nil)
}
