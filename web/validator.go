package web

import (
	"context"

	"goark.dev/arkarta/validation"
	"goark.dev/goark/container"
)

// Validator 表示 Web 请求绑定后使用的校验器。
type Validator = validation.Validator

// GroupValidator 表示支持显式校验分组的 Web 校验器。
type GroupValidator = validation.GroupValidator

// RegisterValidator 注册 Web 校验器贡献点。
func RegisterValidator(registry *container.Registry, name string, validator validation.Validator, options ...container.Option) error {
	if isNilValidator(validator) {
		return ErrNilValidator
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.UseValidator(validator)
		return nil
	}), options...)
}

func isNilValidator(validator validation.Validator) bool {
	return isNilWebValue(validator)
}
