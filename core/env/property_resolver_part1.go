package env

import (
	"reflect"
	"strings"
	"sync"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
)

func (r *PropertySourcesPropertyResolver) resolvePlaceholders(
	text string,
	required bool,
	depth int,
) (string, error) {
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
				return "", arkerrors.Newf(
					arkerrors.CodeInvalidArgument,
					"unclosed property placeholder in %q",
					text,
				)
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
		return zero, true, arkerrors.Newf(
			arkerrors.CodeTypeMismatch,
			"property %q is %T, expected %s",
			key,
			value,
			lang.TypeOf[T](),
		)
	}
	return typed, true, nil
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
		return arkerrors.Newf(
			arkerrors.CodeNotFound,
			"required properties not found: %s",
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func (r *PropertySourcesPropertyResolver) valueToString(value any) (string, error) {
	converted, err := r.conversionServiceSnapshot().Convert(value, reflect.TypeOf(""))
	if err != nil {
		return "", err
	}
	text, ok := converted.(string)
	if !ok {
		return "", arkerrors.Newf(
			arkerrors.CodeTypeMismatch,
			"property value is %T, expected string",
			converted,
		)
	}
	return text, nil
}

// NewPropertySourcesPropertyResolver 创建属性解析器。
func NewPropertySourcesPropertyResolver(
	propertySources PropertySources,
) (*PropertySourcesPropertyResolver, error) {
	if propertySources == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property sources is nil")
	}
	return &PropertySourcesPropertyResolver{
		propertySources:   propertySources,
		conversionService: convert.DefaultService(),
	}, nil
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

func (r *PropertySourcesPropertyResolver) GetPropertyOrDefault(
	key string,
	defaultValue string,
) string {
	if value, ok := r.GetProperty(key); ok {
		return value
	}
	return defaultValue
}

// ConfigurablePropertyResolver 表示可配置的属性解析器。
type ConfigurablePropertyResolver interface {
	PropertyResolver
	ConversionService() *convert.Service
	SetConversionService(service *convert.Service) error
	SetRequiredProperties(keys ...string)
	ValidateRequiredProperties() error
}

func (r *PropertySourcesPropertyResolver) conversionServiceSnapshot() *convert.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.conversionService == nil {
		return convert.DefaultService()
	}
	return r.conversionService
}

// PropertySourcesPropertyResolver 基于 PropertySources 实现属性解析。
type PropertySourcesPropertyResolver struct {
	mu                 sync.RWMutex
	propertySources    PropertySources
	conversionService  *convert.Service
	requiredProperties []string
}

const (
	placeholderPrefix = "${"
	placeholderSuffix = "}"
	defaultSeparator  = ":"
	maxResolveDepth   = 16
)

func (r *PropertySourcesPropertyResolver) requiredPropertiesSnapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.requiredProperties...)
}

// ResolvePlaceholders 尽量解析占位符，无法解析的占位符保持原样。
func (r *PropertySourcesPropertyResolver) ResolvePlaceholders(text string) (string, error) {
	return r.resolvePlaceholders(text, false, 0)
}
