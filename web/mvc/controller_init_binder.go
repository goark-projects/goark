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
		return handleWithInitBinders(ctx, handler, copied)
	})
}

func initBinderInterceptor(initializers []BinderInitializer) arkweb.Interceptor {
	if len(initializers) == 0 {
		return nil
	}
	copied := append([]BinderInitializer(nil), initializers...)
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		if next == nil {
			return nil, arkweb.ErrNilHandler
		}
		return handleWithInitBinders(ctx, next, copied)
	})
}

func handleWithInitBinders(ctx *arkweb.Context, handler arkweb.Handler, initializers []BinderInitializer) (arkweb.Result, error) {
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
