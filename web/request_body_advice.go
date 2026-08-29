package web

import (
	"context"

	"goark.dev/goark/container"
	"goark.dev/goark/web/message"
)

// RequestBodyAdvice 表示请求体读取前后的增强器。
type RequestBodyAdvice = message.ReadAdvice

// RequestBodyAdviceFunc 将函数组适配为 RequestBodyAdvice。
type RequestBodyAdviceFunc = message.ReadAdviceFunc

// RequestBodyAdviceContext 描述一次请求体读取增强上下文。
type RequestBodyAdviceContext = message.ReadAdviceContext

// RegisterRequestBodyAdvice 注册请求体读取增强器贡献点。
func RegisterRequestBodyAdvice(registry *container.Registry, name string, advice message.ReadAdvice, options ...container.Option) error {
	if isNilMessageReadAdvice(advice) {
		return ErrNilRequestBodyAdvice
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseRequestBodyAdvice(advice)
		return nil
	}), options...)
}

func isNilMessageReadAdvice(advice message.ReadAdvice) bool {
	return isNilWebValue(advice)
}
