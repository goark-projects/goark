package convert

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
)

var (
	durationType        = reflect.TypeOf(time.Duration(0))
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func canUseBuiltin(sourceType reflect.Type, targetType reflect.Type) bool {
	if targetType.Kind() == reflect.Pointer {
		return canUseBuiltin(sourceType, targetType.Elem())
	}
	if targetType.Kind() == reflect.Slice {
		return sourceType.Kind() == reflect.String || sourceType.Kind() == reflect.Slice
	}
	if textUnmarshalerTarget(targetType) {
		return sourceType.Kind() == reflect.String
	}
	switch sourceType.Kind() {
	case reflect.String:
		return isStringTarget(targetType)
	default:
		return targetType.Kind() == reflect.String
	}
}

func builtinConvert(value any, targetType reflect.Type, service *Service) (any, error) {
	if targetType.Kind() == reflect.Pointer {
		converted, err := builtinConvert(value, targetType.Elem(), service)
		if err != nil {
			return nil, err
		}
		pointer := reflect.New(targetType.Elem())
		pointer.Elem().Set(reflect.ValueOf(converted))
		return pointer.Interface(), nil
	}

	if targetType.Kind() == reflect.Slice {
		return convertSlice(value, targetType, service)
	}

	if text, ok := value.(string); ok {
		return convertString(text, targetType)
	}
	if targetType.Kind() == reflect.String {
		return fmt.Sprint(value), nil
	}
	return nil, arkerrors.Newf(arkerrors.CodeConversion, "unsupported conversion %T -> %s", value, targetType)
}

func convertString(text string, targetType reflect.Type) (any, error) {
	if textUnmarshalerTarget(targetType) {
		return unmarshalText(text, targetType)
	}
	if targetType == durationType {
		return time.ParseDuration(text)
	}
	switch targetType.Kind() {
	case reflect.String:
		return text, nil
	case reflect.Bool:
		return strconv.ParseBool(text)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(text, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetInt(parsed)
		return out.Interface(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(text, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetUint(parsed)
		return out.Interface(), nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(text, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetFloat(parsed)
		return out.Interface(), nil
	default:
		return nil, arkerrors.Newf(arkerrors.CodeConversion, "unsupported string target %s", targetType)
	}
}

func convertSlice(value any, targetType reflect.Type, service *Service) (any, error) {
	var parts []any
	if text, ok := value.(string); ok {
		rawParts := strings.Split(text, ",")
		parts = make([]any, 0, len(rawParts))
		for _, part := range rawParts {
			parts = append(parts, strings.TrimSpace(part))
		}
	} else {
		sourceValue := reflect.ValueOf(value)
		if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
			return nil, arkerrors.Newf(arkerrors.CodeConversion, "source %T is not slice-compatible", value)
		}
		parts = make([]any, 0, sourceValue.Len())
		for i := 0; i < sourceValue.Len(); i++ {
			parts = append(parts, sourceValue.Index(i).Interface())
		}
	}

	slice := reflect.MakeSlice(targetType, 0, len(parts))
	for _, part := range parts {
		converted, err := service.Convert(part, targetType.Elem())
		if err != nil {
			return nil, err
		}
		convertedValue, err := sliceElementValue(converted, targetType.Elem())
		if err != nil {
			return nil, err
		}
		slice = reflect.Append(slice, convertedValue)
	}
	return slice.Interface(), nil
}

func sliceElementValue(value any, elemType reflect.Type) (reflect.Value, error) {
	if util.IsNil(value) {
		switch elemType.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return reflect.Zero(elemType), nil
		default:
			return reflect.Value{}, arkerrors.Newf(arkerrors.CodeConversion, "nil cannot be appended to %s slice", elemType)
		}
	}
	elemValue := reflect.ValueOf(value)
	if elemValue.Type().AssignableTo(elemType) {
		return elemValue, nil
	}
	if elemValue.Type().ConvertibleTo(elemType) {
		return elemValue.Convert(elemType), nil
	}
	return reflect.Value{}, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted slice element is %T, expected %s", value, elemType)
}

func textUnmarshalerTarget(targetType reflect.Type) bool {
	if targetType.Implements(textUnmarshalerType) {
		return true
	}
	return reflect.PointerTo(targetType).Implements(textUnmarshalerType)
}

func unmarshalText(text string, targetType reflect.Type) (any, error) {
	value := reflect.New(targetType)
	unmarshaler, ok := value.Interface().(encoding.TextUnmarshaler)
	if !ok {
		if value.Elem().CanAddr() {
			unmarshaler, ok = value.Elem().Addr().Interface().(encoding.TextUnmarshaler)
		}
		if !ok {
			return nil, arkerrors.Newf(arkerrors.CodeConversion, "%s does not implement encoding.TextUnmarshaler", targetType)
		}
	}
	if err := unmarshaler.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return value.Elem().Interface(), nil
}

func isStringTarget(targetType reflect.Type) bool {
	if textUnmarshalerTarget(targetType) {
		return true
	}
	if targetType == durationType {
		return true
	}
	switch targetType.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func normalizeConverted(value any, targetType reflect.Type) (any, error) {
	if util.IsNil(value) {
		switch targetType.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return value, nil
		default:
			return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted value is nil, expected %s", targetType)
		}
	}
	converted, ok := assignValue(value, targetType)
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted value is %T, expected %s", value, targetType)
	}
	return converted, nil
}
