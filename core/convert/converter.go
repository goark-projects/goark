package convert

import (
	"reflect"

	"github.com/goark-projects/goark/core/lang"
	arkerrors "github.com/goark-projects/goark/errors"
)

// Converter 描述一个源类型到目标类型的转换器。
type Converter interface {
	SourceType() reflect.Type
	TargetType() reflect.Type
	Convert(value any) (any, error)
}

// ConverterFunc 将类型安全函数适配为 Converter。
type ConverterFunc[S any, T any] func(S) (T, error)

// SourceType 返回转换器接受的源类型。
func (f ConverterFunc[S, T]) SourceType() reflect.Type {
	return lang.TypeOf[S]()
}

// TargetType 返回转换器输出的目标类型。
func (f ConverterFunc[S, T]) TargetType() reflect.Type {
	return lang.TypeOf[T]()
}

// Convert 执行类型安全转换。
func (f ConverterFunc[S, T]) Convert(value any) (any, error) {
	var zero T
	if f == nil {
		return zero, arkerrors.New(arkerrors.CodeInvalidArgument, "converter function is nil")
	}
	source, ok := value.(S)
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converter source is %T, expected %s", value, lang.TypeOf[S]())
	}
	return f(source)
}

// Register 注册类型安全转换函数。
func Register[S any, T any](service *Service, fn func(S) (T, error)) error {
	if service == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "conversion service is nil")
	}
	return service.Register(ConverterFunc[S, T](fn))
}
