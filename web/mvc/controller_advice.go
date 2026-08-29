package mvc

import (
	"context"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
	appcontext "goark.dev/goark/context"
	"goark.dev/goark/core/util"
	goweb "goark.dev/goark/web"
)

const (
	// AttributeControllerAdviceKind 保存当前 MVC advice 的默认返回值策略。
	AttributeControllerAdviceKind = "goark.web.mvc.controller_advice.kind"
)

// ControllerAdvice 描述一组全局 MVC 异常处理器和绑定器初始化器。
type ControllerAdvice struct {
	name     string
	order    int
	kind     ControllerKind
	handlers []goweb.ErrorMapper
	binders  []BinderInitializer
}

// NewControllerAdvice 创建普通 MVC advice，字符串返回值默认解析为逻辑视图名。
func NewControllerAdvice(name string, handlers ...goweb.ErrorMapper) ControllerAdvice {
	return ControllerAdvice{
		name:     name,
		kind:     ControllerKindView,
		handlers: append([]goweb.ErrorMapper(nil), handlers...),
	}
}

// NewRestControllerAdvice 创建 REST MVC advice，普通返回值默认写入响应体。
func NewRestControllerAdvice(name string, handlers ...goweb.ErrorMapper) ControllerAdvice {
	advice := NewControllerAdvice(name, handlers...)
	advice.kind = ControllerKindREST
	return advice
}

// WithOrder 设置 advice 配置和 Web 注册顺序。
func (a ControllerAdvice) WithOrder(order int) ControllerAdvice {
	a.order = order
	return a
}

// WithExceptionHandlers 追加全局异常处理器。
func (a ControllerAdvice) WithExceptionHandlers(handlers ...goweb.ErrorMapper) ControllerAdvice {
	a.handlers = append(a.handlers, handlers...)
	return a
}

// WithInitBinders 追加全局绑定器初始化器，对齐 Spring ControllerAdvice @InitBinder。
func (a ControllerAdvice) WithInitBinders(initializers ...BinderInitializer) ControllerAdvice {
	a.binders = append(a.binders, initializers...)
	return a
}

// Name 返回 advice 配置名称。
func (a ControllerAdvice) Name() string {
	if a.name == "" {
		return "goark.web.mvc.controller-advice"
	}
	return a.name
}

// Order 返回 advice 顺序。
func (a ControllerAdvice) Order() int {
	return a.order
}

// Kind 返回 advice 默认返回值策略。
func (a ControllerAdvice) Kind() ControllerKind {
	return a.kind
}

// ExceptionHandlers 返回异常处理器快照。
func (a ControllerAdvice) ExceptionHandlers() []goweb.ErrorMapper {
	return append([]goweb.ErrorMapper(nil), a.handlers...)
}

// InitBinders 返回全局绑定器初始化器快照。
func (a ControllerAdvice) InitBinders() []BinderInitializer {
	return append([]BinderInitializer(nil), a.binders...)
}

// Register 注册 advice Web 配置器 Bean。
func (a ControllerAdvice) Register(ctx context.Context, registry *container.Registry) error {
	return a.RegisterWithContext(ctx, appcontext.NewConfigurationContext(nil, registry))
}

// RegisterWithContext 注册 advice Web 配置器 Bean。
func (a ControllerAdvice) RegisterWithContext(_ context.Context, config appcontext.ConfigurationContext) error {
	return goweb.RegisterConfigurer(
		config.Registry(),
		a.Name()+".configurer",
		a,
		container.WithOrder(a.order),
	)
}

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
	for _, handler := range a.handlers {
		if util.IsNil(handler) {
			continue
		}
		registry.UseErrorMapper(a.wrapExceptionHandler(handler))
	}
	return nil
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
