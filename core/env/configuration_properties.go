package env

import (
	"sort"
	"strings"

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
		if _, exists := allowedSet[name]; !exists {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return arkerrors.Newf(arkerrors.CodeInvalidArgument, "unknown configuration properties: %s", strings.Join(unknown, ", "))
}

func propertyHasPrefix(name string, prefix string) bool {
	if prefix == "" {
		return name != ""
	}
	return strings.HasPrefix(name, prefix+".")
}
