package env

import (
	"reflect"
	"strconv"
	"strings"

	arkerrors "goark.dev/goark/errors"
)

func flattenConfigMap(values map[string]any) (map[string]any, error) {
	out := make(map[string]any)
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(key, value, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func flattenConfigValue(prefix string, value any, out map[string]any) error {
	switch typed := value.(type) {
	case map[string]any:
		return flattenStringMap(prefix, typed, out)
	case map[any]any:
		return flattenAnyMap(prefix, typed, out)
	case []any:
		return flattenSlice(prefix, typed, out)
	default:
		return flattenReflectValue(prefix, value, out)
	}
}

func flattenStringMap(prefix string, values map[string]any, out map[string]any) error {
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(prefix+"."+key, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenAnyMap(prefix string, values map[any]any, out map[string]any) error {
	for key, value := range values {
		keyText, ok := key.(string)
		if !ok {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "config key %v is not a string", key)
		}
		keyText = strings.TrimSpace(keyText)
		if keyText == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(prefix+"."+keyText, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenSlice(prefix string, values []any, out map[string]any) error {
	out[prefix] = values
	for index, value := range values {
		key := prefix + "[" + strconv.Itoa(index) + "]"
		if err := flattenConfigValue(key, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenReflectValue(prefix string, value any, out map[string]any) error {
	if value == nil {
		out[prefix] = nil
		return nil
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Map:
		if reflectValue.Type().Key().Kind() != reflect.String {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "config key under %q is not string", prefix)
		}
		for _, key := range reflectValue.MapKeys() {
			keyText := strings.TrimSpace(key.String())
			if keyText == "" {
				return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
			}
			if err := flattenConfigValue(prefix+"."+keyText, reflectValue.MapIndex(key).Interface(), out); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		values := make([]any, 0, reflectValue.Len())
		for index := 0; index < reflectValue.Len(); index++ {
			values = append(values, reflectValue.Index(index).Interface())
		}
		return flattenSlice(prefix, values, out)
	default:
		out[prefix] = normalizeConfigScalar(value)
		return nil
	}
}

func normalizeConfigScalar(value any) any {
	return value
}
