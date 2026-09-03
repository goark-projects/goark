package mvc

import (
	"net/url"
	"reflect"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

func modelAttributeBindingValues(ctx *arkweb.Context, target any) (url.Values, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = url.Values{}
	}
	fallbackNames := modelAttributeFallbackNames(target)
	appendModelAttributePathValues(values, ctx, fallbackNames)
	appendModelAttributeHeaderValues(values, ctx, fallbackNames)
	return values, nil
}

func appendModelAttributePathValues(values url.Values, ctx *arkweb.Context, fallbackNames map[string]struct{}) {
	if ctx == nil || len(fallbackNames) == 0 {
		return
	}
	for name, value := range ctx.PathValues() {
		addFallbackModelAttributeValue(values, name, stripMatrixSegment(value), fallbackNames)
	}
}

func appendModelAttributeHeaderValues(values url.Values, ctx *arkweb.Context, fallbackNames map[string]struct{}) {
	if ctx == nil || ctx.Request() == nil || len(fallbackNames) == 0 {
		return
	}
	header := ctx.Request().Header()
	for _, name := range servlet.HeaderNames(header) {
		list := header.Values(name)
		addFallbackModelAttributeValues(values, name, list, fallbackNames)
		if lowerName := strings.ToLower(name); lowerName != name {
			addFallbackModelAttributeValues(values, lowerName, list, fallbackNames)
		}
		if fieldName := modelAttributeHeaderFieldName(name); fieldName != "" && fieldName != name {
			addFallbackModelAttributeValues(values, fieldName, list, fallbackNames)
		}
	}
}

func addFallbackModelAttributeValue(values url.Values, name string, value string, fallbackNames map[string]struct{}) {
	addFallbackModelAttributeValues(values, name, []string{value}, fallbackNames)
}

func addFallbackModelAttributeValues(values url.Values, name string, list []string, fallbackNames map[string]struct{}) {
	if values == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" || len(list) == 0 {
		return
	}
	if _, ok := fallbackNames[name]; !ok {
		return
	}
	if _, exists := values[name]; exists {
		return
	}
	values[name] = append([]string(nil), list...)
}

func modelAttributeFallbackNames(target any) map[string]struct{} {
	targetType := reflect.TypeOf(target)
	for targetType != nil && targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}
	if targetType == nil || targetType.Kind() != reflect.Struct {
		return nil
	}
	names := make(map[string]struct{})
	collectModelAttributeFallbackNames(targetType, "", names)
	if len(names) == 0 {
		return nil
	}
	return names
}

func collectModelAttributeFallbackNames(targetType reflect.Type, prefix string, names map[string]struct{}) {
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldType := modelAttributeDerefType(field.Type)
		if shouldRecurseModelAttributeFallbackField(field, fieldType) {
			collectModelAttributeFallbackNames(fieldType, prefix, names)
			continue
		}
		name, ok := modelAttributeFieldName(field)
		if !ok {
			continue
		}
		name = prefixedModelAttributeName(prefix, name)
		names[name] = struct{}{}
		if shouldCollectNestedModelAttributeFallbackNames(fieldType) {
			collectModelAttributeFallbackNames(fieldType, name, names)
		}
	}
}

func shouldRecurseModelAttributeFallbackField(field reflect.StructField, fieldType reflect.Type) bool {
	return field.Anonymous &&
		field.Tag.Get("form") == "" &&
		field.Tag.Get("multipart") == "" &&
		fieldType.Kind() == reflect.Struct &&
		!isScalarModelAttributeStruct(fieldType)
}

func shouldCollectNestedModelAttributeFallbackNames(fieldType reflect.Type) bool {
	return fieldType.Kind() == reflect.Struct && !isScalarModelAttributeStruct(fieldType)
}

func modelAttributeHeaderFieldName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(name))
	upperNext := false
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '-' {
			upperNext = true
			continue
		}
		if builder.Len() == 0 {
			builder.WriteByte(toLowerASCII(ch))
			upperNext = false
			continue
		}
		if upperNext {
			builder.WriteByte(toUpperASCII(ch))
			upperNext = false
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func toLowerASCII(ch byte) byte {
	if ch >= 'A' && ch <= 'Z' {
		return ch + ('a' - 'A')
	}
	return ch
}

func toUpperASCII(ch byte) byte {
	if ch >= 'a' && ch <= 'z' {
		return ch - ('a' - 'A')
	}
	return ch
}
