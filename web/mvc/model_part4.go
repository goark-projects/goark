package mvc

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
)

func (b modelAttributeBinder) bindIndexedSliceField(name string, field reflect.Value,
	values url.Values) error {
	sliceType := modelAttributeDerefType(field.Type())
	bindings, maxIndex, err := collectIndexedModelAttributeValues(values, name)
	if err != nil || maxIndex < 0 {
		return err
	}
	slice := reflect.MakeSlice(sliceType, maxIndex+1, maxIndex+1)
	for i := 0; i <= maxIndex; i++ {
		binding := bindings[i]
		if binding == nil {
			continue
		}
		elementName := fmt.Sprintf("%s[%d]", name, i)
		element := slice.Index(i)
		if len(binding.direct) > 0 {
			if err := b.setIndexedElement(elementName, element, binding.direct); err != nil {
				return err
			}
		}
		if len(binding.nested) > 0 {
			if err := b.bindIndexedElement(elementName, element, binding.nested); err != nil {
				return err
			}
		}
	}
	return setModelAttributeField(name, "", field, slice.Interface())
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

func modelAttributeFieldValue(targetType reflect.Type, converted any) (reflect.Value, error) {
	if util.IsNil(converted) {
		if modelAttributeCanSetNil(targetType) {
			return reflect.Zero(targetType), nil
		}
		return reflect.Value{}, fmt.Errorf("nil cannot be assigned to %s", targetType)
	}
	value := reflect.ValueOf(converted)
	if value.Type().AssignableTo(targetType) {
		return value, nil
	}
	if targetType.Kind() == reflect.Pointer {
		elem, err := modelAttributeFieldValue(targetType.Elem(), converted)
		if err != nil {
			return reflect.Value{}, err
		}
		pointer := reflect.New(targetType.Elem())
		pointer.Elem().Set(elem)
		return pointer, nil
	}
	if value.Type().ConvertibleTo(targetType) {
		return value.Convert(targetType), nil
	}
	return reflect.Value{}, fmt.Errorf("converted value is %T, expected %s", converted, targetType)
}

func collectModelAttributeFallbackNames(targetType reflect.Type, prefix string,
	names map[string]struct{}) {
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

func (b modelAttributeBinder) convertMapSliceValue(name string, values []string,
	targetType reflect.Type) (reflect.Value, error) {
	sliceType := modelAttributeDerefType(targetType)
	items := splitParamValues(values)
	slice := reflect.MakeSlice(sliceType, 0, len(items))
	for _, item := range items {
		converted, err := b.convertString(name, item, sliceType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		value, err := modelAttributeFieldValue(sliceType.Elem(), converted)
		if err != nil {
			return reflect.Value{}, invalidParameterError(name, item, sliceType.String(), err)
		}
		slice = reflect.Append(slice, value)
	}
	return modelAttributeFieldValue(targetType, slice.Interface())
}

func initializeModelAttributes(ctx *arkweb.Context,
	initializers []ModelAttributeInitializer) error {
	model := CurrentModel(ctx)
	for _, initializer := range initializers {
		if initializer == nil {
			return ErrNilModelAttributeInitializer
		}
		next, err := initializer.InitializeModelAttribute(ctx, model)
		if err != nil {
			return err
		}
		model = next
		setCurrentModel(ctx, model)
	}
	return nil
}

func parseModelAttributeMapKey(name string, key string) (string, error) {
	offset := len(name)
	if len(key) <= offset+2 || key[offset] != '[' {
		return "", fmt.Errorf("缺少 map 键")
	}
	end := strings.IndexByte(key[offset+1:], ']')
	if end < 1 {
		return "", fmt.Errorf("缺少 map 键")
	}
	mapKey := key[offset+1 : offset+1+end]
	if suffix := key[offset+2+end:]; suffix != "" {
		return "", fmt.Errorf("map 属性暂不支持嵌套路径 %q", suffix)
	}
	return mapKey, nil
}

// NewModelAndView 创建 MVC 模型视图结果。
func NewModelAndView(viewName string, model any, options ...ModelAndViewOption) ModelAndView {
	out := ModelAndView{
		viewName: strings.TrimSpace(viewName),
		model:    normalizeModel(model),
		flash:    flashModelFrom(model),
	}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

func (b modelAttributeBinder) convertString(name string, raw string, targetType reflect.Type) (
	any, error) {
	source := raw
	if modelAttributeShouldTrim(targetType) {
		source = strings.TrimSpace(raw)
	}
	converted, err := b.converter.Convert(source, modelAttributeConversionTargetType(targetType))
	if err != nil {
		return nil, invalidParameterError(name, raw, targetType.String(), err)
	}
	return converted, nil
}

func (b modelAttributeBinder) convertMapValue(name string, values []string,
	targetType reflect.Type) (reflect.Value, error) {
	if modelAttributeDerefType(targetType).Kind() == reflect.Slice {
		return b.convertMapSliceValue(name, values, targetType)
	}
	raw := firstIndexedModelAttributeValue(values)
	converted, err := b.convertString(name, raw, targetType)
	if err != nil {
		return reflect.Value{}, err
	}
	return modelAttributeFieldValue(targetType, converted)
}

// ModelAttributeGroups 绑定模型属性并按显式分组执行结构体验证。
func ModelAttributeGroups[T any](ctx *arkweb.Context, groups ...string) (T, error) {
	var out T
	if ctx == nil {
		return out, arkweb.ErrNilContext
	}
	if _, err := bindModelAttribute(ctx, &out); err != nil {
		return out, err
	}
	return out, validateBound(ctx, &out, groups)
}

func wrapModelAttributeInitializers(handler arkweb.Handler,
	initializers []ModelAttributeInitializer) arkweb.Handler {
	if handler == nil || len(initializers) == 0 {
		return handler
	}
	copied := append([]ModelAttributeInitializer(nil), initializers...)
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		return handleWithModelAttributeInitializers(ctx, handler, copied)
	})
}

func hasIndexedModelAttributePrefix(values url.Values, name string) bool {
	prefix := name + "["
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func appendModelAttributePathValues(values url.Values, ctx *arkweb.Context,
	fallbackNames map[string]struct{}) {
	if ctx == nil || len(fallbackNames) == 0 {
		return
	}
	for name, value := range ctx.PathValues() {
		addFallbackModelAttributeValue(values, name, stripMatrixSegment(value), fallbackNames)
	}
}

func shouldRecurseModelAttributeFallbackField(field reflect.StructField,
	fieldType reflect.Type) bool {
	return field.Anonymous &&
		field.Tag.Get("form") == "" &&
		field.Tag.Get("multipart") == "" &&
		fieldType.Kind() == reflect.Struct &&
		!isScalarModelAttributeStruct(fieldType)
}

func shouldRecurseModelAttributeField(field reflect.StructField, value reflect.Value) bool {
	if !field.Anonymous || field.Tag.Get("form") != "" || field.Tag.Get("multipart") != "" {
		return false
	}
	value = indirectModelAttributeStruct(value)
	return value.IsValid() && value.Kind() == reflect.Struct
}

func (b modelAttributeBinder) hasFieldMarker(name string) bool {
	if len(b.fieldMarkers) == 0 {
		return false
	}
	_, ok := b.fieldMarkers[name]
	return ok
}

func lowerFirstRune(value string) string {
	first, size := utf8.DecodeRuneInString(value)
	if first == utf8.RuneError && size == 0 {
		return value
	}
	return string(unicode.ToLower(first)) + value[size:]
}

func prefixedModelAttributeName(prefix string, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func dereferenceModelType(modelType reflect.Type) reflect.Type {
	for modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}
	return modelType
}

type modelAttributeBinder struct {
	value        reflect.Value
	converter    *convert.Service
	fieldMarkers map[string]struct{}
}

type mappedModelAttributeValue struct {
	key    string
	values []string
}

// ModelAttribute 绑定模型属性并执行结构体验证。
func ModelAttribute[T any](ctx *arkweb.Context) (T, error) {
	return ModelAttributeGroups[T](ctx)
}

// Model 返回视图模型副本。
func (v ModelAndView) Model() map[string]any {
	return v.model.Values()
}

// NewModel 创建 MVC 视图模型。
func NewModel() Model {
	return Model{attributes: make(map[string]any)}
}

func shouldCollectNestedModelAttributeFallbackNames(fieldType reflect.Type) bool {
	return fieldType.Kind() == reflect.Struct && !isScalarModelAttributeStruct(fieldType)
}

const modelAttributeFormContentType = "application/x-www-form-urlencoded"

const fallbackModelAttributeName = "value"
