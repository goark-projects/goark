package env

import (
	"context"
	"reflect"
	"strings"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/expression"
)

// ResolveValue 解析 goark:value 表达式，并转换为目标 Go 类型。
func ResolveValue(environment Environment, expression string, targetType reflect.Type) (any, error) {
	if environment == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	if targetType == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "value target type is nil")
	}
	resolved := expression
	var err error
	if strings.Contains(resolved, "${") {
		resolved, err = environment.ResolveRequiredPlaceholders(resolved)
		if err != nil {
			return nil, err
		}
	}
	value := any(resolved)
	trimmed := strings.TrimSpace(resolved)
	if strings.HasPrefix(trimmed, "#{") && strings.HasSuffix(trimmed, "}") {
		value, err = evaluateGaEL(environment, trimmed[2:len(trimmed)-1])
		if err != nil {
			return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "failed to evaluate GaEL expression %q", expression)
		}
	} else if strings.Contains(resolved, "#{") {
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "GaEL expression must occupy the complete value: %q", expression)
	}
	converted, err := conversionServiceOf(environment).Convert(value, targetType)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeConversion, err, "failed to resolve value %q as %s", expression, targetType)
	}
	return converted, nil
}

func evaluateGaEL(environment Environment, source string) (any, error) {
	evaluationContext, err := expression.NewEvaluationContext(environment)
	if err != nil {
		return nil, err
	}
	parsed, err := expression.NewParser().Parse(source)
	if err != nil {
		return nil, err
	}
	return parsed.Evaluate(context.Background(), evaluationContext)
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
