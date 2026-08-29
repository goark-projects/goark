package web

import (
	"context"

	"goark.dev/goark/container"
	"goark.dev/goark/web/message"
)

// MessageConverter 是 Spring HttpMessageConverter 的 Go 化双向接口。
type MessageConverter = message.HTTPConverter

// MessageWriteConverter 表示响应体写出转换器。
type MessageWriteConverter = message.Converter

// MessageReadConverter 表示请求体读取转换器。
type MessageReadConverter = message.ReadConverter

// RegisterMessageConverter 注册同时支持读写的消息转换器贡献点。
func RegisterMessageConverter(registry *container.Registry, name string, converter message.HTTPConverter, options ...container.Option) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.AddMessageReadConverter(converter)
		webRegistry.AddMessageConverter(converter)
		return nil
	}), options...)
}

// RegisterMessageReadConverter 注册请求体读取转换器贡献点。
func RegisterMessageReadConverter(registry *container.Registry, name string, converter message.ReadConverter, options ...container.Option) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.AddMessageReadConverter(converter)
		return nil
	}), options...)
}

// RegisterMessageWriteConverter 注册响应体写出转换器贡献点。
func RegisterMessageWriteConverter(registry *container.Registry, name string, converter message.Converter, options ...container.Option) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(registry, name, ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if webRegistry == nil {
			return ErrNilRegistry
		}
		webRegistry.AddMessageConverter(converter)
		return nil
	}), options...)
}

func isNilMessageConverter(converter any) bool {
	return isNilWebValue(converter)
}
