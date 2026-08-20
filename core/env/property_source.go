package env

import (
	"reflect"
	"sort"
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
)

// PropertySource 表示一个命名配置源。
type PropertySource interface {
	Name() string
	Source() any
	ContainsProperty(name string) bool
	GetProperty(name string) (any, bool)
}

// EnumerablePropertySource 表示可枚举属性名的配置源。
type EnumerablePropertySource interface {
	PropertySource
	PropertyNames() []string
}

// MapPropertySource 是基于 map 的不可变配置源。
type MapPropertySource struct {
	name   string
	source map[string]any
}

// NewMapPropertySource 创建 map 配置源，并复制输入数据。
func NewMapPropertySource(name string, source map[string]any) (*MapPropertySource, error) {
	if strings.TrimSpace(name) == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	copied := make(map[string]any, len(source))
	for key, value := range source {
		if strings.TrimSpace(key) == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property key is empty")
		}
		copied[key] = clonePropertyValue(value)
	}
	return &MapPropertySource{name: name, source: copied}, nil
}

func (s *MapPropertySource) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

func (s *MapPropertySource) Source() any {
	if s == nil {
		return nil
	}
	return clonePropertyMap(s.source)
}

func (s *MapPropertySource) ContainsProperty(name string) bool {
	if s == nil || name == "" {
		return false
	}
	_, ok := s.source[name]
	return ok
}

func (s *MapPropertySource) GetProperty(name string) (any, bool) {
	if s == nil || name == "" {
		return nil, false
	}
	value, ok := s.source[name]
	return clonePropertyValue(value), ok
}

func (s *MapPropertySource) PropertyNames() []string {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.source))
	for name := range s.source {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PropertiesPropertySource 语义上表示传统 properties 配置源。
type PropertiesPropertySource = MapPropertySource

// NewPropertiesPropertySource 创建 properties 风格配置源。
func NewPropertiesPropertySource(name string, source map[string]any) (*PropertiesPropertySource, error) {
	return NewMapPropertySource(name, source)
}

// ConfigPropertySource 语义上表示文件配置源。
type ConfigPropertySource = MapPropertySource

// NewConfigPropertySource 创建文件配置源。
func NewConfigPropertySource(name string, source map[string]any) (*ConfigPropertySource, error) {
	return NewMapPropertySource(name, source)
}

func clonePropertyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	copied := make(map[string]any, len(source))
	for key, value := range source {
		copied[key] = clonePropertyValue(value)
	}
	return copied
}

func clonePropertyValue(value any) any {
	if value == nil {
		return nil
	}
	cloned := cloneReflectValue(reflect.ValueOf(value))
	if !cloned.IsValid() {
		return nil
	}
	return cloned.Interface()
}

func cloneReflectValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		return cloneReflectValue(value.Elem())
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copied := reflect.MakeMapWithSize(value.Type(), value.Len())
		for _, key := range value.MapKeys() {
			copied.SetMapIndex(key, cloneReflectValue(value.MapIndex(key)))
		}
		return copied
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copied := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			copied.Index(i).Set(cloneReflectValue(value.Index(i)))
		}
		return copied
	case reflect.Array:
		copied := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			copied.Index(i).Set(cloneReflectValue(value.Index(i)))
		}
		return copied
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copied := reflect.New(value.Type().Elem())
		copied.Elem().Set(cloneReflectValue(value.Elem()))
		return copied
	default:
		return value
	}
}
