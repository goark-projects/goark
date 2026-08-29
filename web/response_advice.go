package web

import (
	"context"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/container"
)

// ResponseAdvice 表示响应写出前的增强器。
type ResponseAdvice = arkweb.ResponseAdvice

// ResponseAdviceFunc 将普通函数适配为 ResponseAdvice。
type ResponseAdviceFunc = arkweb.ResponseAdviceFunc

// RegisterResponseAdvice 注册 Web 响应增强器贡献点。
func RegisterResponseAdvice(registry *container.Registry, name string, advice arkweb.ResponseAdvice, options ...container.Option) error {
	if isNilResponseAdvice(advice) {
		return ErrNilResponseAdvice
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseResponseAdvice(advice)
		return nil
	}), options...)
}

func isNilResponseAdvice(advice arkweb.ResponseAdvice) bool {
	return isNilWebValue(advice)
}
