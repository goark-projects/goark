package mvc

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	mvcflash "goark.dev/goark/web/mvc/flash"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/goark/web/mvc/view"
)

func (b modelAttributeBinder) bindStruct(value reflect.Value, values url.Values,
	prefix string) error {
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

func parseModelAttributeIndex(name string, key string) (int, string, error) {
	offset := len(name)
	if len(key) <= offset+2 || key[offset] != '[' {
		return 0, "", fmt.Errorf("缺少集合索引")
	}
	end := strings.IndexByte(key[offset+1:], ']')
	if end < 1 {
		return 0, "", fmt.Errorf("缺少集合索引")
	}
	rawIndex := key[offset+1 : offset+1+end]
	for _, ch := range rawIndex {
		if ch < '0' || ch > '9' {
			return 0, "", fmt.Errorf("集合索引 %q 非法", rawIndex)
		}
	}
	index, err := strconv.ParseUint(rawIndex, 10, 31)
	if err != nil {
		return 0, "", fmt.Errorf("集合索引 %q 非法: %w", rawIndex, err)
	}
	if index >= maxModelAttributeCollectionAutoGrow {
		return 0, "", fmt.Errorf("集合索引 %d 超过自动增长上限 %d", index, maxModelAttributeCollectionAutoGrow)
	}
	return int(index), key[offset+2+end:], nil
}

func (b modelAttributeBinder) bindMappedField(name string, field reflect.Value,
	values url.Values) error {
	mapType := modelAttributeDerefType(field.Type())
	entries, err := collectMappedModelAttributeValues(values, name)
	if err != nil {
		return err
	}
	out := reflect.MakeMapWithSize(mapType, len(entries))
	for _, entry := range entries {
		entryName := name + "[" + entry.key + "]"
		key, err := b.convertMapKey(entryName, entry.key, mapType.Key())
		if err != nil {
			return err
		}
		value, err := b.convertMapValue(entryName, entry.values, mapType.Elem())
		if err != nil {
			return err
		}
		out.SetMapIndex(key, value)
	}
	return setModelAttributeField(name, "", field, out.Interface())
}

func currentModel(ctx *arkweb.Context) (Model, bool) {
	if ctx == nil || ctx.Request() == nil {
		return NewModel(), false
	}
	value, ok := ctx.Request().Attribute(AttributeModel)
	if !ok {
		return NewModel(), false
	}
	switch typed := value.(type) {
	case Model:
		return typed, true
	case *Model:
		if typed != nil {
			return *typed, true
		}
	}
	return NewModel(), false
}

// AddAllAttributes 添加多个模型属性。
func (m Model) AddAllAttributes(attributes map[string]any) Model {
	if len(attributes) == 0 {
		return m
	}
	if m.attributes == nil {
		m.attributes = make(map[string]any, len(attributes))
	}
	for name, value := range attributes {
		name = strings.TrimSpace(name)
		if name != "" {
			m.attributes[name] = value
		}
	}
	return m
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

func handleWithModelAttributeInitializers(
	ctx *arkweb.Context,
	handler arkweb.Handler,
	initializers []ModelAttributeInitializer,
) (arkweb.Result, error) {
	if handler == nil {
		return nil, arkweb.ErrNilHandler
	}
	if err := initializeModelAttributes(ctx, initializers); err != nil {
		return nil, err
	}
	return handler.Handle(ctx)
}

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

func hasModelAttributeMarkerPrefix(markers map[string]struct{}, name string) bool {
	if len(markers) == 0 {
		return false
	}
	prefix := name + "."
	for marker := range markers {
		if strings.HasPrefix(marker, prefix) {
			return true
		}
	}
	return false
}

// CurrentModel 返回当前 MVC 请求模型；没有模型时创建空模型。
func CurrentModel(ctx *arkweb.Context) Model {
	model, ok := currentModel(ctx)
	if ok {
		return model
	}
	input := mvcflash.Input(ctx)
	model = NewModel().AddAllAttributes((&input).Values())
	setCurrentModel(ctx, model)
	return model
}

func modelAttributeInterceptor(initializers []ModelAttributeInitializer) arkweb.Interceptor {
	if len(initializers) == 0 {
		return nil
	}
	copied := append([]ModelAttributeInitializer(nil), initializers...)
	return arkweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result,
		error) {
		return handleWithModelAttributeInitializers(ctx, next, copied)
	})
}

func hasBracketedModelAttributePrefix(values url.Values, name string) bool {
	prefix := name + "["
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func modelAttributeMethodAllowsBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func modelAttributeCanSetNil(targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

// ModelAndView 表示视图名、模型和状态码的组合结果。
type ModelAndView struct {
	viewName string
	model    Model
	flash    Model
	status   int
	resolver view.Resolver
}

func shouldBindIndexedModelAttributeField(value reflect.Value, values url.Values,
	name string) bool {
	if name == "" || modelAttributeDerefType(value.Type()).Kind() != reflect.Slice {
		return false
	}
	return hasIndexedModelAttributePrefix(values, name)
}

// DefaultViewName 按请求路径推导默认逻辑视图名。
func DefaultViewName(ctx *arkweb.Context) string {
	if ctx == nil || ctx.Request() == nil {
		return defaultViewName
	}
	return defaultViewNameFromPath(ctx.Request().Path())
}

func toLowerASCII(ch byte) byte {
	if ch >= 'A' && ch <= 'Z' {
		return ch + ('a' - 'A')
	}
	return ch
}

// WithViewStatus 设置视图响应状态码。
func WithViewStatus(statusCode int) ModelAndViewOption {
	return func(out *ModelAndView) {
		out.status = normalizeResponseStatus(statusCode, 0)
	}
}

func modelAttributeShouldTrim(targetType reflect.Type) bool {
	targetType = modelAttributeDerefType(targetType)
	return targetType.Kind() != reflect.String
}

type indexedModelAttributeValue struct {
	direct []string
	nested url.Values
}

func addFallbackModelAttributeValue(values url.Values, name string, value string,
	fallbackNames map[string]struct{}) {
	addFallbackModelAttributeValues(values, name, []string{value}, fallbackNames)
}

// ViewName 返回逻辑视图名。
func (v ModelAndView) ViewName() string {
	return v.viewName
}

// Model 表示 MVC 视图模型。
type Model struct {
	attributes map[string]any
}

func modelAttributeConversionTargetType(targetType reflect.Type) reflect.Type {
	return modelAttributeDerefType(targetType)
}

// maxModelAttributeMapEntries 限制表单 map 自动增长规模，防止单请求过量分配。
const maxModelAttributeMapEntries = 256

const defaultViewName = "index"
