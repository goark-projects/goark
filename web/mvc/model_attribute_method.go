package mvc

import (
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// ModelAttributeInitializer 表示方法级 @ModelAttribute 的 Go 化模型初始化器。
type ModelAttributeInitializer interface {
	InitializeModelAttribute(ctx *arkweb.Context, model Model) (Model, error)
}

// ModelAttributeInitializerFunc 将函数适配为模型初始化器。
type ModelAttributeInitializerFunc func(ctx *arkweb.Context, model Model) (Model, error)

// InitializeModelAttribute 执行模型初始化函数。
func (f ModelAttributeInitializerFunc) InitializeModelAttribute(ctx *arkweb.Context, model Model) (Model, error) {
	if f == nil {
		return model, ErrNilModelAttributeInitializer
	}
	return f(ctx, model)
}

// ModelAttributeValue 将普通返回值加入当前请求模型。
func ModelAttributeValue[T any](name string, fn ValueFunc[T]) ModelAttributeInitializer {
	return modelAttributeValue[T]{
		name: strings.TrimSpace(name),
		fn:   fn,
	}
}

type modelAttributeValue[T any] struct {
	name string
	fn   ValueFunc[T]
}

func (v modelAttributeValue[T]) InitializeModelAttribute(ctx *arkweb.Context, model Model) (Model, error) {
	if v.name == "" {
		return model, ErrInvalidModelAttributeName
	}
	if v.fn == nil {
		return model, ErrNilModelAttributeInitializer
	}
	value, err := v.fn(ctx)
	if err != nil {
		return model, err
	}
	return model.AddAttribute(v.name, value), nil
}

func wrapModelAttributeInitializers(handler arkweb.Handler, initializers []ModelAttributeInitializer) arkweb.Handler {
	if handler == nil || len(initializers) == 0 {
		return handler
	}
	copied := append([]ModelAttributeInitializer(nil), initializers...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return handleWithModelAttributeInitializers(ctx, handler, copied)
	})
}

func modelAttributeInterceptor(initializers []ModelAttributeInitializer) arkweb.Interceptor {
	if len(initializers) == 0 {
		return nil
	}
	copied := append([]ModelAttributeInitializer(nil), initializers...)
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		return handleWithModelAttributeInitializers(ctx, next, copied)
	})
}

func handleWithModelAttributeInitializers(
	ctx *arkweb.Context,
	handler arkweb.Handler,
	initializers []ModelAttributeInitializer,
) (arkweb.Result, error) {
	if handler == nil {
		return nil, arkweb.ErrNilHandler
	}
	if err := initializeModelAttributes(ctx, initializers); err != nil {
		return nil, err
	}
	return handler.Handle(ctx)
}

func initializeModelAttributes(ctx *arkweb.Context, initializers []ModelAttributeInitializer) error {
	model := CurrentModel(ctx)
	for _, initializer := range initializers {
		if initializer == nil {
			return ErrNilModelAttributeInitializer
		}
		next, err := initializer.InitializeModelAttribute(ctx, model)
		if err != nil {
			return err
		}
		model = next
		setCurrentModel(ctx, model)
	}
	return nil
}
