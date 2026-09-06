package web

import (
	"context"
	"reflect"

	arkweb "goark.dev/arkarta/web"
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
func RegisterMessageConverter(
	registry *container.Registry,
	name string,
	converter message.HTTPConverter,
	options ...container.Option,
) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(
		registry,
		name,
		ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if webRegistry == nil {
				return ErrNilRegistry
			}
			webRegistry.AddMessageReadConverter(converter)
			webRegistry.AddMessageConverter(converter)
			return nil
		}),
		options...)
}

// RegisterMessageReadConverter 注册请求体读取转换器贡献点。
func RegisterMessageReadConverter(
	registry *container.Registry,
	name string,
	converter message.ReadConverter,
	options ...container.Option,
) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(
		registry,
		name,
		ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if webRegistry == nil {
				return ErrNilRegistry
			}
			webRegistry.AddMessageReadConverter(converter)
			return nil
		}),
		options...)
}

// RegisterMessageWriteConverter 注册响应体写出转换器贡献点。
func RegisterMessageWriteConverter(
	registry *container.Registry,
	name string,
	converter message.Converter,
	options ...container.Option,
) error {
	if isNilMessageConverter(converter) {
		return ErrNilMessageConverter
	}
	return RegisterConfigurer(
		registry,
		name,
		ConfigurerFunc(func(ctx context.Context, webRegistry *Registry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if webRegistry == nil {
				return ErrNilRegistry
			}
			webRegistry.AddMessageConverter(converter)
			return nil
		}),
		options...)
}

func isNilMessageConverter(converter any) bool {
	return isNilWebValue(converter)
}

type messageResult struct {
	statusCode int
	value      any
	mediaTypes []string
}

// Message 创建基于消息转换器和 Accept 协商的响应结果。
func Message(statusCode int, value any, mediaTypes ...string) arkweb.Result {
	return messageResult{
		statusCode: statusCode,
		value:      value,
		mediaTypes: append([]string(nil), mediaTypes...),
	}
}

// Write 将消息响应写入 Arkarta Web 上下文。
func (r messageResult) Write(ctx *arkweb.Context) error {
	return message.WriterFromContext(ctx).Write(ctx, r.statusCode, r.value, r.mediaTypes...)
}

func isNilWebValue(value any) bool {
	if value == nil {
		return true
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflectValue.IsNil()
	default:
		return false
	}
}
