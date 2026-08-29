package mvc

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
)

const modelAttributeFormContentType = "application/x-www-form-urlencoded"

type modelAttributeBinder struct {
	value        reflect.Value
	converter    *convert.Service
	fieldMarkers map[string]struct{}
}

func bindModelAttribute(ctx *arkweb.Context, target any) ([]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	if err := ensureModelAttributeContentType(ctx); err != nil {
		return nil, err
	}
	values, err := modelAttributeBindingValues(ctx, target)
	if err != nil {
		return nil, err
	}
	preparedValues := modelAttributeValuesForCurrentBinder(ctx, values)
	binder, err := newModelAttributeBinder(target, ConversionServiceFromContext(ctx), preparedValues.fieldMarkers)
	if err != nil {
		return preparedValues.suppressedFields, err
	}
	return preparedValues.suppressedFields, binder.bind(preparedValues.values)
}

func ensureModelAttributeContentType(ctx *arkweb.Context) error {
	contentType := strings.TrimSpace(ctx.Request().Header().Get("Content-Type"))
	if contentType == "" || !modelAttributeMethodAllowsBody(ctx.Request().Method()) {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return arkweb.ErrUnsupportedMediaType
	}
	if strings.EqualFold(mediaType, modelAttributeFormContentType) {
		return nil
	}
	return arkweb.ErrUnsupportedMediaType
}

func modelAttributeMethodAllowsBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func newModelAttributeBinder(
	target any,
	converter *convert.Service,
	fieldMarkers map[string]struct{},
) (modelAttributeBinder, error) {
	if util.IsNil(target) {
		return modelAttributeBinder{}, arkweb.ErrInvalidBindTarget
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return modelAttributeBinder{}, arkweb.ErrInvalidBindTarget
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return modelAttributeBinder{}, arkweb.ErrInvalidBindTarget
	}
	if converter == nil {
		converter = DefaultConversionService()
	}
	return modelAttributeBinder{
		value:        value,
		converter:    converter,
		fieldMarkers: fieldMarkers,
	}, nil
}

func (b modelAttributeBinder) bind(values url.Values) error {
	return b.bindStruct(b.value, values, "")
}

func (b modelAttributeBinder) bindStruct(value reflect.Value, values url.Values, prefix string) error {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if shouldRecurseModelAttributeField(field, fieldValue) {
			nested := indirectModelAttributeStruct(fieldValue)
			if nested.IsValid() {
				if err := b.bindStruct(nested, values, prefix); err != nil {
					return err
				}
			}
			continue
		}
		name, ok := modelAttributeFieldName(field)
		if !ok {
			continue
		}
		name = prefixedModelAttributeName(prefix, name)
		if shouldBindNestedModelAttributeField(fieldValue, values, b.fieldMarkers, name) {
			nested := indirectModelAttributeStruct(fieldValue)
			if nested.IsValid() {
				if err := b.bindStruct(nested, values, name); err != nil {
					return err
				}
			}
			continue
		}
		if !fieldValue.CanSet() {
			continue
		}
		if list, exists := values[name]; exists && len(list) > 0 {
			if err := b.setField(name, fieldValue, list); err != nil {
				return err
			}
			continue
		}
		if shouldBindIndexedModelAttributeField(fieldValue, values, name) {
			if err := b.bindIndexedSliceField(name, fieldValue, values); err != nil {
				return err
			}
			continue
		}
		if shouldBindMappedModelAttributeField(fieldValue, values, name) {
			if err := b.bindMappedField(name, fieldValue, values); err != nil {
				return err
			}
			continue
		}
		if b.hasFieldMarker(name) {
			if err := setModelAttributeMarkerField(name, fieldValue); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldRecurseModelAttributeField(field reflect.StructField, value reflect.Value) bool {
	if !field.Anonymous || field.Tag.Get("form") != "" || field.Tag.Get("multipart") != "" {
		return false
	}
	value = indirectModelAttributeStruct(value)
	return value.IsValid() && value.Kind() == reflect.Struct
}

func indirectModelAttributeStruct(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() && value.CanSet() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func modelAttributeFieldName(field reflect.StructField) (string, bool) {
	if field.Tag.Get("form") == "-" {
		return "", false
	}
	if name, ok := modelAttributeTagName(field.Tag.Get("form")); ok {
		return name, true
	}
	if field.Tag.Get("json") == "-" {
		return "", false
	}
	if name, ok := modelAttributeTagName(field.Tag.Get("json")); ok {
		return name, true
	}
	return lowerModelAttributeFieldName(field.Name), true
}

func modelAttributeTagName(tag string) (string, bool) {
	if tag == "-" {
		return "", false
	}
	name := strings.TrimSpace(strings.Split(tag, ",")[0])
	if name == "" {
		return "", false
	}
	return name, true
}

func lowerModelAttributeFieldName(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func (b modelAttributeBinder) setField(name string, field reflect.Value, values []string) error {
	if modelAttributeDerefType(field.Type()).Kind() == reflect.Slice {
		return b.setSliceField(name, field, values)
	}
	raw := values[0]
	converted, err := b.convertString(name, raw, field.Type())
	if err != nil {
		return err
	}
	return setModelAttributeField(name, raw, field, converted)
}

func (b modelAttributeBinder) setSliceField(name string, field reflect.Value, values []string) error {
	sliceType := modelAttributeDerefType(field.Type())
	items := splitParamValues(values)
	slice := reflect.MakeSlice(sliceType, 0, len(items))
	for _, item := range items {
		converted, err := b.convertString(name, item, sliceType.Elem())
		if err != nil {
			return err
		}
		convertedValue, err := modelAttributeFieldValue(sliceType.Elem(), converted)
		if err != nil {
			return invalidParameterError(name, item, sliceType.String(), err)
		}
		slice = reflect.Append(slice, convertedValue)
	}
	return setModelAttributeField(name, strings.Join(values, ","), field, slice.Interface())
}

func (b modelAttributeBinder) convertString(name string, raw string, targetType reflect.Type) (any, error) {
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

func modelAttributeShouldTrim(targetType reflect.Type) bool {
	targetType = modelAttributeDerefType(targetType)
	return targetType.Kind() != reflect.String
}

func modelAttributeDerefType(targetType reflect.Type) reflect.Type {
	for targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}
	return targetType
}

func modelAttributeConversionTargetType(targetType reflect.Type) reflect.Type {
	return modelAttributeDerefType(targetType)
}

func setModelAttributeField(name string, raw string, field reflect.Value, converted any) error {
	value, err := modelAttributeFieldValue(field.Type(), converted)
	if err != nil {
		return invalidParameterError(name, raw, field.Type().String(), err)
	}
	field.Set(value)
	return nil
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

func setModelAttributeMarkerField(name string, field reflect.Value) error {
	value, err := modelAttributeMarkerFieldValue(field.Type())
	if err != nil {
		return invalidParameterError(name, "", field.Type().String(), err)
	}
	field.Set(value)
	return nil
}

func modelAttributeMarkerFieldValue(targetType reflect.Type) (reflect.Value, error) {
	if targetType.Kind() == reflect.Pointer {
		value, err := modelAttributeMarkerFieldValue(targetType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		pointer := reflect.New(targetType.Elem())
		pointer.Elem().Set(value)
		return pointer, nil
	}
	switch targetType.Kind() {
	case reflect.Slice:
		return reflect.MakeSlice(targetType, 0, 0), nil
	case reflect.Map:
		return reflect.MakeMapWithSize(targetType, 0), nil
	default:
		return reflect.Zero(targetType), nil
	}
}

func (b modelAttributeBinder) hasFieldMarker(name string) bool {
	if len(b.fieldMarkers) == 0 {
		return false
	}
	_, ok := b.fieldMarkers[name]
	return ok
}

func modelAttributeCanSetNil(targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}
