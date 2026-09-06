package env

import (
	"reflect"
	"strings"

	"goark.dev/goark/core/convert"
	arkerrors "goark.dev/goark/errors"
)

func (r *PropertySourcesPropertyResolver) resolvePlaceholderExpression(
	expression string,
	required bool,
	depth int,
) (string, bool, error) {
	key, fallback := splitPlaceholderExpression(expression)
	key = strings.TrimSpace(key)
	if key == "" {
		if required {
			return "", false, arkerrors.New(
				arkerrors.CodeInvalidArgument,
				"property placeholder key is empty",
			)
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
		return "", false, arkerrors.Newf(
			arkerrors.CodeNotFound,
			"property placeholder %q not found",
			key,
		)
	}
	return "", false, nil
}

func (r *PropertySourcesPropertyResolver) GetPropertyAs(
	key string,
	targetType reflect.Type,
) (any, bool, error) {
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

func (r *PropertySourcesPropertyResolver) GetRequiredPropertyAs(
	key string,
	targetType reflect.Type,
) (any, error) {
	value, ok, err := r.GetPropertyAs(key, targetType)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
	}
	return value, nil
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

func (r *PropertySourcesPropertyResolver) SetRequiredProperties(keys ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requiredProperties = append([]string(nil), keys...)
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
func (r *PropertySourcesPropertyResolver) GetRequiredProperty(key string) (string, error) {
	value, ok := r.GetProperty(key)
	if !ok {
		return "", arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
	}
	return value, nil
}

func (r *PropertySourcesPropertyResolver) ConversionService() *convert.Service {
	if r == nil {
		return nil
	}
	return r.conversionServiceSnapshot()
}

func (r *PropertySourcesPropertyResolver) ContainsProperty(key string) bool {
	_, ok := r.findProperty(key)
	return ok
}

// ResolveRequiredPlaceholders 解析占位符，无法解析时报错。
func (r *PropertySourcesPropertyResolver) ResolveRequiredPlaceholders(text string) (string, error) {
	return r.resolvePlaceholders(text, true, 0)
}
