package container

import (
	"context"
	"fmt"

	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/internal/reflectx"
)

// Get 按名称解析并转换为目标类型。
func Get[T any](ctx context.Context, resolver Resolver, name string) (T, error) {
	var zero T
	if resolver == nil {
		return zero, arkerrors.New(arkerrors.CodeInvalidArgument, "bean resolver is nil")
	}
	value, err := resolver.Get(ctx, name)
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean %q is %T, expected %s", name, value, typeName[T]())
	}
	return typed, nil
}

// GetByType 按类型解析并转换为目标类型。
func GetByType[T any](ctx context.Context, resolver Resolver, options ...ResolveOption) (T, error) {
	var zero T
	if resolver == nil {
		return zero, arkerrors.New(arkerrors.CodeInvalidArgument, "bean resolver is nil")
	}
	value, err := resolver.GetByType(ctx, reflectx.TypeOf[T](), options...)
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean type result is %T, expected %s", value, typeName[T]())
	}
	return typed, nil
}

// GetAllByType 按类型解析全部 Bean，并转换为目标类型切片。
func GetAllByType[T any](ctx context.Context, resolver Resolver) ([]T, error) {
	if resolver == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean resolver is nil")
	}
	values, err := resolver.GetAllByType(ctx, reflectx.TypeOf[T]())
	if err != nil {
		return nil, err
	}
	typed := make([]T, 0, len(values))
	for _, value := range values {
		item, ok := value.(T)
		if !ok {
			return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "bean type result is %T, expected %s", value, typeName[T]())
		}
		typed = append(typed, item)
	}
	return typed, nil
}

// MustGet 是 Get 的 panic 版本，适合初始化期快速失败。
func MustGet[T any](ctx context.Context, resolver Resolver, name string) T {
	value, err := Get[T](ctx, resolver, name)
	if err != nil {
		panic(err)
	}
	return value
}

// MustGetByType 是 GetByType 的 panic 版本。
func MustGetByType[T any](ctx context.Context, resolver Resolver, options ...ResolveOption) T {
	value, err := GetByType[T](ctx, resolver, options...)
	if err != nil {
		panic(err)
	}
	return value
}

func typeName[T any]() string {
	return fmt.Sprint(reflectx.TypeOf[T]())
}
