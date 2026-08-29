package mvc

import (
	"net/url"
	"reflect"
	"strings"
)

func prefixedModelAttributeName(prefix string, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func shouldBindNestedModelAttributeField(value reflect.Value, values url.Values, name string) bool {
	if name == "" || !hasModelAttributePrefix(values, name) {
		return false
	}
	targetType := modelAttributeDerefType(value.Type())
	return targetType.Kind() == reflect.Struct && !isScalarModelAttributeStruct(targetType)
}

func hasModelAttributePrefix(values url.Values, name string) bool {
	prefix := name + "."
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func isScalarModelAttributeStruct(targetType reflect.Type) bool {
	return targetType.PkgPath() == "time" && targetType.Name() == "Time"
}
