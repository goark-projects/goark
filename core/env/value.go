package env

import (
	"reflect"
	"strings"

	"github.com/goark-projects/goark/core/convert"
	"github.com/goark-projects/goark/core/lang"
	arkerrors "github.com/goark-projects/goark/errors"
)

// ResolveValue 解析 goark:value 表达式，并转换为目标 Go 类型。
func ResolveValue(environment Environment, expression string, targetType reflect.Type) (any, error) {
	if environment == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	if targetType == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "value target type is nil")
	}
	if strings.Contains(expression, "#{") {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "SpEL expressions are not supported")
	}
	resolved := expression
	var err error
	if strings.Contains(expression, "${") {
		resolved, err = environment.ResolveRequiredPlaceholders(expression)
		if err != nil {
			return nil, err
		}
	}
	converted, err := conversionServiceOf(environment).Convert(resolved, targetType)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeConversion, err, "failed to resolve value %q as %s", expression, targetType)
	}
	return converted, nil
}

// ResolveValueAs 按泛型目标类型解析 goark:value 表达式。
func ResolveValueAs[T any](environment Environment, expression string) (T, error) {
	var zero T
	value, err := ResolveValue(environment, expression, lang.TypeOf[T]())
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeTypeMismatch, "resolved value is %T, expected %s", value, lang.TypeOf[T]())
	}
	return typed, nil
}

func conversionServiceOf(environment Environment) *convert.Service {
	if configurable, ok := environment.(ConfigurablePropertyResolver); ok {
		if service := configurable.ConversionService(); service != nil {
			return service
		}
	}
	return convert.DefaultService()
}
