package mvc

import (
	"reflect"
	"unicode"
	"unicode/utf8"
)

const fallbackModelAttributeName = "value"

func inferModelAttributeName(value any) (string, bool) {
	modelType := reflect.TypeOf(value)
	if modelType == nil {
		return "", false
	}
	suffix := ""
	modelType = dereferenceModelType(modelType)
	if modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array {
		suffix = "List"
		modelType = dereferenceModelType(modelType.Elem())
	}
	name := modelType.Name()
	if name == "" {
		return fallbackModelAttributeName, true
	}
	return lowerFirstRune(name) + suffix, true
}

func dereferenceModelType(modelType reflect.Type) reflect.Type {
	for modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}
	return modelType
}

func lowerFirstRune(value string) string {
	first, size := utf8.DecodeRuneInString(value)
	if first == utf8.RuneError && size == 0 {
		return value
	}
	return string(unicode.ToLower(first)) + value[size:]
}
