package mvc

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	arkweb "goark.dev/arkarta/web"

	"goark.dev/arkarta/servlet"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
	"goark.dev/goark/web/mvc/view"
)

func collectIndexedModelAttributeValues(values url.Values, name string) (
	map[int]*indexedModelAttributeValue, int, error) {
	prefix := name + "["
	bindings := make(map[int]*indexedModelAttributeValue)
	maxIndex := -1
	for key, list := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		index, suffix, err := parseModelAttributeIndex(name, key)
		if err != nil {
			return nil, -1, invalidParameterError(key, firstIndexedModelAttributeValue(list),
				"indexed model attribute", err)
		}
		binding := bindings[index]
		if binding == nil {
			binding = &indexedModelAttributeValue{}
			bindings[index] = binding
		}
		switch {
		case suffix == "":
			binding.direct = append(binding.direct, list...)
		case strings.HasPrefix(suffix, ".") && len(suffix) > 1:
			if binding.nested == nil {
				binding.nested = make(url.Values)
			}
			binding.nested[suffix[1:]] = append(binding.nested[suffix[1:]], list...)
		default:
			return nil, -1, invalidParameterError(key, firstIndexedModelAttributeValue(list),
				"indexed model attribute", fmt.Errorf("非法索引属性路径 %q", key))
		}
		if index > maxIndex {
			maxIndex = index
		}
	}
	return bindings, maxIndex, nil
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

func normalizeModel(model any) Model {
	switch value := model.(type) {
	case nil:
		return NewModel()
	case Model:
		return value
	case *Model:
		if value == nil {
			return NewModel()
		}
		return *value
	case RedirectAttributes:
		return value.Model()
	case *RedirectAttributes:
		if value == nil {
			return NewModel()
		}
		return value.Model()
	case map[string]any:
		return NewModel().AddAllAttributes(value)
	default:
		return NewModel().AddAttributeValue(value)
	}
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

func appendModelAttributeHeaderValues(values url.Values, ctx *arkweb.Context,
	fallbackNames map[string]struct{}) {
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

// ModelAttributeResultGroups 绑定模型属性，并按显式分组返回绑定和验证结果。
func ModelAttributeResultGroups[T any](ctx *arkweb.Context, groups ...string) (T,
	BindingResult, error) {
	var out T
	if ctx == nil {
		return out, BindingResult{}, arkweb.ErrNilContext
	}
	suppressedFields, err := bindModelAttribute(ctx, &out)
	if err != nil {
		return out, newBindingErrorResult(err).withSuppressedFields(suppressedFields), nil
	}
	result, err := validateBindingResult(ctx, &out, groups)
	return out, result.withSuppressedFields(suppressedFields), err
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

func (b modelAttributeBinder) setIndexedElement(name string, element reflect.Value,
	values []string) error {
	if modelAttributeDerefType(element.Type()).Kind() == reflect.Slice {
		return b.setSliceField(name, element, values)
	}
	raw := values[0]
	converted, err := b.convertString(name, raw, element.Type())
	if err != nil {
		return err
	}
	return setModelAttributeField(name, raw, element, converted)
}

// AddAttribute 添加模型属性。
func (m Model) AddAttribute(name string, value any) Model {
	name = strings.TrimSpace(name)
	if name == "" {
		return m
	}
	if m.attributes == nil {
		m.attributes = make(map[string]any, 1)
	}
	m.attributes[name] = value
	return m
}

func mergeCurrentModel(ctx *arkweb.Context, model Model) Model {
	base, ok := currentModel(ctx)
	if !ok {
		setCurrentModel(ctx, model)
		return model
	}
	merged := base.AddAllAttributes(model.Values())
	setCurrentModel(ctx, merged)
	return merged
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

func setModelAttributeField(name string, raw string, field reflect.Value, converted any) error {
	value, err := modelAttributeFieldValue(field.Type(), converted)
	if err != nil {
		return invalidParameterError(name, raw, field.Type().String(), err)
	}
	field.Set(value)
	return nil
}

func (b modelAttributeBinder) convertMapKey(name string, raw string, targetType reflect.Type) (
	reflect.Value, error) {
	converted, err := b.convertString(name, raw, targetType)
	if err != nil {
		return reflect.Value{}, err
	}
	return modelAttributeFieldValue(targetType, converted)
}

// AddAttributeValue 添加模型属性，并按值类型推导属性名。
func (m Model) AddAttributeValue(value any) Model {
	name, ok := inferModelAttributeName(value)
	if !ok {
		return m
	}
	return m.AddAttribute(name, value)
}

// ModelAttributeValue 将普通返回值加入当前请求模型。
func ModelAttributeValue[T any](name string, fn ValueFunc[T]) ModelAttributeInitializer {
	return modelAttributeValue[T]{
		name: strings.TrimSpace(name),
		fn:   fn,
	}
}

func lowerModelAttributeFieldName(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func firstIndexedModelAttributeValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func toUpperASCII(ch byte) byte {
	if ch >= 'a' && ch <= 'z' {
		return ch - ('a' - 'A')
	}
	return ch
}

// WithViewResolver 设置模型视图使用的视图解析器。
func WithViewResolver(resolver view.Resolver) ModelAndViewOption {
	return func(out *ModelAndView) {
		out.resolver = resolver
	}
}

// ModelAttributeInitializer 表示方法级 @ModelAttribute 的 Go 化模型初始化器。
type ModelAttributeInitializer interface {
	InitializeModelAttribute(ctx *arkweb.Context, model Model) (Model, error)
}

// ModelAttributeResult 绑定模型属性，并返回可由调用方处理的绑定和验证结果。
func ModelAttributeResult[T any](ctx *arkweb.Context) (T, BindingResult, error) {
	return ModelAttributeResultGroups[T](ctx)
}

func implicitModelResult(ctx *arkweb.Context, statusCode int, model Model) arkweb.Result {
	return NewModelAndView(DefaultViewName(ctx), model, WithViewStatus(resolveResponseStatus(ctx,
		statusCode, http.StatusOK)))
}

// Len 返回模型属性数量。
func (m Model) Len() int {
	return len(m.attributes)
}

// maxModelAttributeCollectionAutoGrow 对齐 Spring DataBinder 默认自动增长上限，避免大索引触发过量分配。
const maxModelAttributeCollectionAutoGrow = 256

// ModelAttributeInitializerFunc 将函数适配为模型初始化器。
type ModelAttributeInitializerFunc func(ctx *arkweb.Context, model Model) (Model, error)
