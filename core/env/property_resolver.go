package env

import (
	"reflect"
	"strings"
	"sync"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
)

const (
	placeholderPrefix = "${"
	placeholderSuffix = "}"
	defaultSeparator  = ":"
	maxResolveDepth   = 16
)

// PropertyResolver 负责配置读取、类型转换与占位符解析。
type PropertyResolver interface {
	ContainsProperty(key string) bool
	GetProperty(key string) (string, bool)
	GetPropertyOrDefault(key string, defaultValue string) string
	GetRequiredProperty(key string) (string, error)
	GetPropertyAs(key string, targetType reflect.Type) (any, bool, error)
	GetRequiredPropertyAs(key string, targetType reflect.Type) (any, error)
	ResolvePlaceholders(text string) (string, error)
	ResolveRequiredPlaceholders(text string) (string, error)
}

// ConfigurablePropertyResolver 表示可配置的属性解析器。
type ConfigurablePropertyResolver interface {
	PropertyResolver
	ConversionService() *convert.Service
	SetConversionService(service *convert.Service) error
	SetRequiredProperties(keys ...string)
	ValidateRequiredProperties() error
}

// PropertySourcesPropertyResolver 基于 PropertySources 实现属性解析。
type PropertySourcesPropertyResolver struct {
	mu                 sync.RWMutex
	propertySources    PropertySources
	conversionService  *convert.Service
	requiredProperties []string
}

// NewPropertySourcesPropertyResolver 创建属性解析器。
func NewPropertySourcesPropertyResolver(propertySources PropertySources) (*PropertySourcesPropertyResolver, error) {
	if propertySources == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property sources is nil")
	}
	return &PropertySourcesPropertyResolver{
		propertySources:   propertySources,
		conversionService: convert.DefaultService(),
	}, nil
}

func (r *PropertySourcesPropertyResolver) ContainsProperty(key string) bool {
	_, ok := r.findProperty(key)
	return ok
}

func (r *PropertySourcesPropertyResolver) GetProperty(key string) (string, bool) {
	value, ok := r.findProperty(key)
	if !ok {
		return "", false
	}
	text, err := r.valueToString(value)
	if err != nil {
		return "", false
	}
	resolved, err := r.ResolvePlaceholders(text)
	if err != nil {
		return "", false
	}
	return resolved, true
}

func (r *PropertySourcesPropertyResolver) GetPropertyOrDefault(key string, defaultValue string) string {
	if value, ok := r.GetProperty(key); ok {
		return value
	}
	return defaultValue
}

func (r *PropertySourcesPropertyResolver) GetRequiredProperty(key string) (string, error) {
	value, ok := r.GetProperty(key)
	if !ok {
		return "", arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
	}
	return value, nil
}

func (r *PropertySourcesPropertyResolver) GetPropertyAs(key string, targetType reflect.Type) (any, bool, error) {
	if targetType == nil {
		return nil, false, arkerrors.New(arkerrors.CodeInvalidArgument, "target type is nil")
	}
	value, ok := r.findProperty(key)
	if !ok {
		return nil, false, nil
	}
	if text, ok := value.(string); ok {
		resolved, err := r.ResolvePlaceholders(text)
		if err != nil {
			return nil, true, err
		}
		value = resolved
	}
	converted, err := r.conversionServiceSnapshot().Convert(value, targetType)
	if err != nil {
		return nil, true, err
	}
	return converted, true, nil
}

func (r *PropertySourcesPropertyResolver) GetRequiredPropertyAs(key string, targetType reflect.Type) (any, error) {
	value, ok, err := r.GetPropertyAs(key, targetType)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
	}
	return value, nil
}

// ResolvePlaceholders 尽量解析占位符，无法解析的占位符保持原样。
func (r *PropertySourcesPropertyResolver) ResolvePlaceholders(text string) (string, error) {
	return r.resolvePlaceholders(text, false, 0)
}

// ResolveRequiredPlaceholders 解析占位符，无法解析时报错。
func (r *PropertySourcesPropertyResolver) ResolveRequiredPlaceholders(text string) (string, error) {
	return r.resolvePlaceholders(text, true, 0)
}

func (r *PropertySourcesPropertyResolver) ConversionService() *convert.Service {
	if r == nil {
		return nil
	}
	return r.conversionServiceSnapshot()
}

func (r *PropertySourcesPropertyResolver) SetConversionService(service *convert.Service) error {
	if r == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver is nil")
	}
	if service == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "conversion service is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conversionService = service
	return nil
}

func (r *PropertySourcesPropertyResolver) SetRequiredProperties(keys ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requiredProperties = append([]string(nil), keys...)
}

func (r *PropertySourcesPropertyResolver) ValidateRequiredProperties() error {
	if r == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver is nil")
	}
	missing := make([]string, 0)
	for _, key := range r.requiredPropertiesSnapshot() {
		if !r.ContainsProperty(key) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return arkerrors.Newf(arkerrors.CodeNotFound, "required properties not found: %s", strings.Join(missing, ", "))
	}
	return nil
}

// GetPropertyAsValue 按泛型目标类型读取配置。
func GetPropertyAsValue[T any](resolver PropertyResolver, key string) (T, bool, error) {
	var zero T
	if resolver == nil {
		return zero, false, arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver is nil")
	}
	value, ok, err := resolver.GetPropertyAs(key, lang.TypeOf[T]())
	if err != nil || !ok {
		return zero, ok, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, true, arkerrors.Newf(arkerrors.CodeTypeMismatch, "property %q is %T, expected %s", key, value, lang.TypeOf[T]())
	}
	return typed, true, nil
}

// GetRequiredPropertyAsValue 按泛型目标类型读取必需配置。
func GetRequiredPropertyAsValue[T any](resolver PropertyResolver, key string) (T, error) {
	var zero T
	value, ok, err := GetPropertyAsValue[T](resolver, key)
	if err != nil {
		return zero, err
	}
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
	}
	return value, nil
}

func (r *PropertySourcesPropertyResolver) findProperty(key string) (any, bool) {
	if r == nil || r.propertySources == nil || key == "" {
		return nil, false
	}
	for _, source := range r.propertySources.Snapshot() {
		if value, ok := source.GetProperty(key); ok {
			return value, true
		}
	}
	return nil, false
}

func (r *PropertySourcesPropertyResolver) valueToString(value any) (string, error) {
	converted, err := r.conversionServiceSnapshot().Convert(value, reflect.TypeOf(""))
	if err != nil {
		return "", err
	}
	text, ok := converted.(string)
	if !ok {
		return "", arkerrors.Newf(arkerrors.CodeTypeMismatch, "property value is %T, expected string", converted)
	}
	return text, nil
}

func (r *PropertySourcesPropertyResolver) conversionServiceSnapshot() *convert.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.conversionService == nil {
		return convert.DefaultService()
	}
	return r.conversionService
}

func (r *PropertySourcesPropertyResolver) requiredPropertiesSnapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.requiredProperties...)
}

func (r *PropertySourcesPropertyResolver) resolvePlaceholders(text string, required bool, depth int) (string, error) {
	if depth > maxResolveDepth {
		return "", arkerrors.New(arkerrors.CodeConflict, "property placeholder nesting is too deep")
	}
	start := strings.Index(text, placeholderPrefix)
	if start < 0 {
		return text, nil
	}

	var builder strings.Builder
	offset := 0
	for start >= 0 {
		builder.WriteString(text[offset:start])
		end := findPlaceholderEnd(text, start)
		if end < 0 {
			if required {
				return "", arkerrors.Newf(arkerrors.CodeInvalidArgument, "unclosed property placeholder in %q", text)
			}
			builder.WriteString(text[start:])
			return builder.String(), nil
		}
		rawExpression := text[start : end+len(placeholderSuffix)]
		expression := text[start+len(placeholderPrefix) : end]
		value, ok, err := r.resolvePlaceholderExpression(expression, required, depth)
		if err != nil {
			return "", err
		}
		if ok {
			builder.WriteString(value)
		} else {
			builder.WriteString(rawExpression)
		}
		offset = end + len(placeholderSuffix)
		next := strings.Index(text[offset:], placeholderPrefix)
		if next < 0 {
			break
		}
		start = offset + next
	}
	builder.WriteString(text[offset:])

	resolved := builder.String()
	if resolved != text && strings.Contains(resolved, placeholderPrefix) {
		return r.resolvePlaceholders(resolved, required, depth+1)
	}
	return resolved, nil
}

func findPlaceholderEnd(text string, start int) int {
	index := start + len(placeholderPrefix)
	nested := 0
	for index < len(text) {
		switch {
		case strings.HasPrefix(text[index:], placeholderSuffix):
			if nested == 0 {
				return index
			}
			nested--
			index += len(placeholderSuffix)
		case strings.HasPrefix(text[index:], placeholderPrefix):
			nested++
			index += len(placeholderPrefix)
		default:
			index++
		}
	}
	return -1
}

func (r *PropertySourcesPropertyResolver) resolvePlaceholderExpression(expression string, required bool, depth int) (string, bool, error) {
	key, fallback := splitPlaceholderExpression(expression)
	key = strings.TrimSpace(key)
	if key == "" {
		if required {
			return "", false, arkerrors.New(arkerrors.CodeInvalidArgument, "property placeholder key is empty")
		}
		return "", false, nil
	}
	if value, ok := r.findProperty(key); ok {
		text, err := r.valueToString(value)
		if err != nil {
			return "", false, err
		}
		resolved, err := r.resolvePlaceholders(text, required, depth+1)
		return resolved, true, err
	}
	if fallback != nil {
		resolved, err := r.resolvePlaceholders(*fallback, required, depth+1)
		return resolved, true, err
	}
	if required {
		return "", false, arkerrors.Newf(arkerrors.CodeNotFound, "property placeholder %q not found", key)
	}
	return "", false, nil
}

func splitPlaceholderExpression(expression string) (string, *string) {
	index := strings.Index(expression, defaultSeparator)
	if index < 0 {
		return expression, nil
	}
	key := expression[:index]
	fallback := expression[index+len(defaultSeparator):]
	return key, &fallback
}
