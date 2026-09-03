package gael

import (
	"reflect"

	arkerrors "goark.dev/goark/errors"
)

func indexedValue(target any, index any) (any, error) {
	value := reflect.ValueOf(target)
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return nil, arkerrors.New(arkerrors.CodeNotFound, "GaEL indexed target is nil")
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return nil, arkerrors.New(arkerrors.CodeNotFound, "GaEL indexed target is nil")
	}
	switch value.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(index)
		if !key.IsValid() || !key.Type().AssignableTo(value.Type().Key()) {
			return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "GaEL map index is %T, expected %s", index, value.Type().Key())
		}
		result := value.MapIndex(key)
		if !result.IsValid() {
			return nil, arkerrors.Newf(arkerrors.CodeNotFound, "GaEL map key %v not found", index)
		}
		return result.Interface(), nil
	case reflect.Array, reflect.Slice, reflect.String:
		number, err := numeric(index)
		if err != nil {
			return nil, err
		}
		integer, ok := number.(int64)
		if !ok || integer < 0 || integer >= int64(value.Len()) {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "GaEL index %v is out of range", index)
		}
		result := value.Index(int(integer))
		if value.Kind() == reflect.String {
			return uint8(result.Uint()), nil
		}
		return result.Interface(), nil
	default:
		return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "GaEL value of type %T is not indexable", target)
	}
}
