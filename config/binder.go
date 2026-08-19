package config

import (
	"reflect"
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Binder 将 Environment 中的扁平配置绑定到结构体。
type Binder struct {
	env *Environment
}

// NewBinder 创建配置绑定器。
func NewBinder(env *Environment) (*Binder, error) {
	if env == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return &Binder{env: env}, nil
}

// Bind 使用临时绑定器将配置绑定到目标结构体。
func Bind(env *Environment, prefix string, target any) error {
	binder, err := NewBinder(env)
	if err != nil {
		return err
	}
	return binder.Bind(prefix, target)
}

// Bind 将指定前缀下的配置绑定到结构体指针。
func (b *Binder) Bind(prefix string, target any) error {
	if b == nil || b.env == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "binder environment is nil")
	}
	if target == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bind target is nil")
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bind target must be a non-nil pointer")
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bind target must point to a struct")
	}
	return b.bindStruct(cleanPrefix(prefix), elem)
}

func (b *Binder) bindStruct(prefix string, value reflect.Value) error {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldValue := value.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		key, skip := fieldKey(field)
		if skip {
			continue
		}
		fullKey := joinKey(prefix, key)
		if isNestedStruct(fieldValue) {
			if err := b.bindNested(fullKey, fieldValue); err != nil {
				return err
			}
			continue
		}

		text, ok := b.lookupField(prefix, field)
		if !ok {
			continue
		}
		if err := setValue(fieldValue, text); err != nil {
			return arkerrors.Wrapf(arkerrors.CodeTypeMismatch, err, "failed to bind property %q", fullKey)
		}
	}
	return nil
}

func (b *Binder) bindNested(prefix string, value reflect.Value) error {
	if value.Kind() == reflect.Pointer {
		if !b.hasDescendant(prefix) {
			return nil
		}
		value.Set(reflect.New(value.Type().Elem()))
		value = value.Elem()
	}
	return b.bindStruct(prefix, value)
}

func (b *Binder) lookupField(prefix string, field reflect.StructField) (string, bool) {
	for _, candidate := range fieldCandidates(prefix, field) {
		if value, ok := b.env.Get(candidate); ok {
			return value, true
		}
	}
	return "", false
}

func (b *Binder) hasDescendant(prefix string) bool {
	if prefix == "" {
		return len(b.env.Keys()) > 0
	}
	childPrefix := prefix + "."
	for _, key := range b.env.Keys() {
		if strings.HasPrefix(key, childPrefix) {
			return true
		}
	}
	return false
}
