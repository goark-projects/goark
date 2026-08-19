package config

import (
	"encoding"
	"reflect"
	"strconv"
	"strings"
	"time"

	arkerrors "github.com/goark-projects/goark/errors"
)

var (
	durationType        = reflect.TypeOf(time.Duration(0))
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func isNestedStruct(value reflect.Value) bool {
	typ := value.Type()
	if typ == durationType {
		return false
	}
	if typ.Kind() == reflect.Pointer {
		elem := typ.Elem()
		return elem.Kind() == reflect.Struct &&
			elem != durationType &&
			!typ.Implements(textUnmarshalerType) &&
			!elem.Implements(textUnmarshalerType)
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	if typ.Implements(textUnmarshalerType) {
		return false
	}
	return !reflect.PointerTo(typ).Implements(textUnmarshalerType)
}

func setValue(value reflect.Value, text string) error {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		return setValue(value.Elem(), text)
	}
	if value.CanAddr() && value.Addr().Type().Implements(textUnmarshalerType) {
		return value.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(text))
	}
	if value.Type() == durationType {
		duration, err := time.ParseDuration(text)
		if err != nil {
			return err
		}
		value.SetInt(int64(duration))
		return nil
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(text)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(text)
		if err != nil {
			return err
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(text, 10, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(text, 10, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(text, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetFloat(parsed)
	case reflect.Slice:
		return setSlice(value, text)
	default:
		return arkerrors.Newf(arkerrors.CodeTypeMismatch, "unsupported bind target type %s", value.Type())
	}
	return nil
}

func setSlice(value reflect.Value, text string) error {
	parts := strings.Split(text, ",")
	slice := reflect.MakeSlice(value.Type(), 0, len(parts))
	for _, part := range parts {
		elem := reflect.New(value.Type().Elem()).Elem()
		if err := setValue(elem, strings.TrimSpace(part)); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}
	value.Set(slice)
	return nil
}
