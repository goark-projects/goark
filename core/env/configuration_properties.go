package env

import (
	"reflect"
	"sort"
	"strings"

	"goark.dev/goark/core/lang"
	arkerrors "goark.dev/goark/errors"
)

// EnumerablePropertyResolver 表示可以枚举全部属性名的解析器。
type EnumerablePropertyResolver interface {
	PropertyResolver
	PropertyNames() []string
}

// ConfigurationPropertiesValidator 由配置属性结构体实现，用于绑定后校验。
type ConfigurationPropertiesValidator interface {
	Validate() error
}

// ConfigurationProperty 描述一个编译期生成的配置属性。
type ConfigurationProperty struct {
	Name         string
	Type         string
	DefaultValue string
	Required     bool
}

// ValidateConfigurationPropertyNames 校验指定前缀下不存在未声明属性。
func ValidateConfigurationPropertyNames(resolver PropertyResolver, prefix string, allowed []string) error {
	if resolver == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver is nil")
	}
	enumerable, ok := resolver.(EnumerablePropertyResolver)
	if !ok {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver does not support property name enumeration")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), ".")
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		name = strings.TrimSpace(name)
		if name != "" {
			allowedSet[name] = struct{}{}
		}
	}
	unknown := make([]string, 0)
	for _, name := range enumerable.PropertyNames() {
		if !propertyHasPrefix(name, prefix) {
			continue
		}
		if !configurationPropertyAllowed(name, allowedSet) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return arkerrors.Newf(arkerrors.CodeInvalidArgument, "unknown configuration properties: %s", strings.Join(unknown, ", "))
}

// GetPropertyMapAsValue 按属性名前缀绑定字符串键映射。
func GetPropertyMapAsValue[V any](resolver PropertyResolver, prefix string) (map[string]V, bool, error) {
	if resolver == nil {
		return nil, false, arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver is nil")
	}
	enumerable, ok := resolver.(EnumerablePropertyResolver)
	if !ok {
		return nil, false, arkerrors.New(arkerrors.CodeInvalidArgument, "property resolver does not support property name enumeration")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), ".")
	propertyPrefix := prefix
	if propertyPrefix != "" {
		propertyPrefix += "."
	}
	names := enumerable.PropertyNames()
	sort.Strings(names)
	result := make(map[string]V)
	for _, name := range names {
		if !strings.HasPrefix(name, propertyPrefix) || len(name) == len(propertyPrefix) {
			continue
		}
		key := strings.TrimPrefix(name, propertyPrefix)
		value, found, err := resolver.GetPropertyAs(name, lang.TypeOf[V]())
		if err != nil {
			return nil, false, arkerrors.Wrapf(arkerrors.CodeConversion, err, "failed to bind map configuration property %q", name)
		}
		if !found {
			continue
		}
		typed, ok := value.(V)
		if !ok {
			return nil, false, arkerrors.Newf(arkerrors.CodeTypeMismatch, "configuration property %q is %T, expected %s", name, value, reflect.TypeOf((*V)(nil)).Elem())
		}
		result[key] = typed
	}
	if len(result) == 0 {
		return nil, false, nil
	}
	return result, true, nil
}

func configurationPropertyAllowed(name string, allowed map[string]struct{}) bool {
	if _, exists := allowed[name]; exists {
		return true
	}
	for candidate := range allowed {
		if strings.HasSuffix(candidate, ".*") && propertyHasPrefix(name, strings.TrimSuffix(candidate, ".*")) {
			return true
		}
	}
	return false
}

func propertyHasPrefix(name string, prefix string) bool {
	if prefix == "" {
		return name != ""
	}
	return strings.HasPrefix(name, prefix+".")
}
