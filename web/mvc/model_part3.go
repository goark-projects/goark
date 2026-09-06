package mvc

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web/mvc/view"
)

func collectMappedModelAttributeValues(values url.Values, name string) (
	[]mappedModelAttributeValue, error) {
	prefix := name + "["
	entries := make([]mappedModelAttributeValue, 0)
	indexes := make(map[string]int)
	for key, list := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		mapKey, err := parseModelAttributeMapKey(name, key)
		if err != nil {
			return nil, invalidParameterError(key, firstIndexedModelAttributeValue(list),
				"mapped model attribute", err)
		}
		if position, exists := indexes[mapKey]; exists {
			entries[position].values = append(entries[position].values, list...)
			continue
		}
		if len(entries) >= maxModelAttributeMapEntries {
			return nil, invalidParameterError(key, firstIndexedModelAttributeValue(list),
				"mapped model attribute", fmt.Errorf("map 属性数量超过上限 %d", maxModelAttributeMapEntries))
		}
		indexes[mapKey] = len(entries)
		entries = append(entries, mappedModelAttributeValue{
			key:    mapKey,
			values: append([]string(nil), list...),
		})
	}
	return entries, nil
}

func defaultViewNameFromPath(rawPath string) string {
	rawPath = strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	if rawPath == "" || rawPath == "/" {
		return defaultViewName
	}
	segments := strings.Split(rawPath, "/")
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = stripPathSemicolon(segment)
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		out = append(out, segment)
	}
	if len(out) == 0 {
		return defaultViewName
	}
	last := out[len(out)-1]
	if ext := path.Ext(last); ext != "" {
		last = strings.TrimSuffix(last, ext)
	}
	out[len(out)-1] = last
	name := path.Clean(strings.Join(out, "/"))
	if name == "." || strings.HasPrefix(name, "../") || name == ".." {
		return defaultViewName
	}
	return name
}

// Write 渲染模型视图。
func (v ModelAndView) Write(ctx *arkweb.Context) error {
	viewName := v.viewName
	if viewName == "" {
		viewName = DefaultViewName(ctx)
	}
	if result, ok := forwardResultFromViewName(viewName); ok {
		return result.Write(ctx)
	}
	if result, location, ok, err := redirectResultAndLocationFromViewNameWithModel(ctx, v.status,
		viewName, v.model); err != nil {
		return err
	} else if ok {
		if err := saveRedirectFlash(ctx, location, v.flash); err != nil {
			return err
		}
		return result.Write(ctx)
	}
	return view.Using(
		v.resolver,
		viewName,
		v.model.Values(),
		view.WithStatus(resolveResponseStatus(ctx, v.status, http.StatusOK)),
	).Write(ctx)
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
	binder, err := newModelAttributeBinder(target, ConversionServiceFromContext(ctx),
		preparedValues.fieldMarkers)
	if err != nil {
		return preparedValues.suppressedFields, err
	}
	return preparedValues.suppressedFields, binder.bind(preparedValues.values)
}

func (b modelAttributeBinder) setSliceField(name string, field reflect.Value,
	values []string) error {
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

func addFallbackModelAttributeValues(values url.Values, name string, list []string,
	fallbackNames map[string]struct{}) {
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

func (v modelAttributeValue[T]) InitializeModelAttribute(ctx *arkweb.Context, model Model) (
	Model, error) {
	if v.name == "" {
		return model, ErrInvalidModelAttributeName
	}
	if v.fn == nil {
		return model, ErrNilModelAttributeInitializer
	}
	value, err := v.fn(ctx)
	if err != nil {
		return model, err
	}
	return model.AddAttribute(v.name, value), nil
}

func shouldBindNestedModelAttributeField(
	value reflect.Value,
	values url.Values,
	fieldMarkers map[string]struct{},
	name string,
) bool {
	if name == "" || (!hasModelAttributePrefix(values, name) && !hasModelAttributeMarkerPrefix(
		fieldMarkers, name)) {
		return false
	}
	targetType := modelAttributeDerefType(value.Type())
	return targetType.Kind() == reflect.Struct && !isScalarModelAttributeStruct(targetType)
}

func (b modelAttributeBinder) bindIndexedElement(name string, element reflect.Value,
	values url.Values) error {
	targetType := modelAttributeDerefType(element.Type())
	if targetType.Kind() != reflect.Struct || isScalarModelAttributeStruct(targetType) {
		return invalidParameterError(name, "", "struct", fmt.Errorf("索引嵌套属性需要结构体元素"))
	}
	nested := indirectModelAttributeStruct(element)
	if !nested.IsValid() {
		return invalidParameterError(name, "", "struct", fmt.Errorf("索引嵌套属性需要可设置元素"))
	}
	return b.bindStruct(nested, values, "")
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

// Values 返回模型属性副本。
func (m Model) Values() map[string]any {
	if len(m.attributes) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(m.attributes))
	for name, value := range m.attributes {
		out[name] = value
	}
	return out
}
func setModelAttributeMarkerField(name string, field reflect.Value) error {
	value, err := modelAttributeMarkerFieldValue(field.Type())
	if err != nil {
		return invalidParameterError(name, "", field.Type().String(), err)
	}
	field.Set(value)
	return nil
}

// InitializeModelAttribute 执行模型初始化函数。
func (f ModelAttributeInitializerFunc) InitializeModelAttribute(ctx *arkweb.Context,
	model Model) (Model, error) {
	if f == nil {
		return model, ErrNilModelAttributeInitializer
	}
	return f(ctx, model)
}

// Attribute 返回模型属性。
func (m Model) Attribute(name string) (any, bool) {
	if len(m.attributes) == 0 {
		return nil, false
	}
	value, ok := m.attributes[name]
	return value, ok
}

func modelForView(ctx *arkweb.Context) any {
	model, ok := currentModel(ctx)
	if !ok {
		return nil
	}
	return model.Values()
}

func modelAttributeDerefType(targetType reflect.Type) reflect.Type {
	for targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}
	return targetType
}

func shouldBindMappedModelAttributeField(value reflect.Value, values url.Values, name string) bool {
	if name == "" || modelAttributeDerefType(value.Type()).Kind() != reflect.Map {
		return false
	}
	return hasBracketedModelAttributePrefix(values, name)
}

func setCurrentModel(ctx *arkweb.Context, model Model) {
	if ctx == nil || ctx.Request() == nil {
		return
	}
	ctx.Request().SetAttribute(AttributeModel, model)
}

func stripPathSemicolon(segment string) string {
	if index := strings.IndexByte(segment, ';'); index >= 0 {
		return segment[:index]
	}
	return segment
}

type modelAttributeValue[T any] struct {
	name string
	fn   ValueFunc[T]
}

const (
	// AttributeModel 保存当前 MVC 请求的视图模型。
	AttributeModel = "goark.web.mvc.model"
)

func mergeModelAndView(ctx *arkweb.Context, value ModelAndView) ModelAndView {
	value.model = mergeCurrentModel(ctx, value.model)
	return value
}

func (b modelAttributeBinder) bind(values url.Values) error {
	return b.bindStruct(b.value, values, "")
}

func isScalarModelAttributeStruct(targetType reflect.Type) bool {
	return targetType.PkgPath() == "time" && targetType.Name() == "Time"
}

// ModelAndViewOption 定制 ModelAndView。
type ModelAndViewOption func(*ModelAndView)
