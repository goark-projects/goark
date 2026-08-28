package mvc

import (
	"errors"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

// ExceptionFunc 表示可决定是否处理当前错误的 MVC 异常函数。
type ExceptionFunc func(ctx *arkweb.Context, err error) (arkweb.Result, bool)

// ExceptionPredicate 判断当前错误是否应由异常处理器处理。
type ExceptionPredicate func(err error) bool

// ExceptionResultFunc 将指定错误转换为 Web 响应。
type ExceptionResultFunc[E error] func(ctx *arkweb.Context, err E) arkweb.Result

// ExceptionHandler 将 MVC 异常函数适配为 Web 错误映射器。
func ExceptionHandler(fn ExceptionFunc) goweb.ErrorMapper {
	return goweb.ErrorMapperFunc(func(ctx *arkweb.Context, err error) arkweb.Result {
		if fn == nil {
			return nil
		}
		result, handled := fn(ctx, err)
		if !handled {
			return nil
		}
		return result
	})
}

// ExceptionHandlerAs 使用 errors.As 匹配指定错误类型。
func ExceptionHandlerAs[E error](fn ExceptionResultFunc[E]) goweb.ErrorMapper {
	return ExceptionHandler(func(ctx *arkweb.Context, err error) (arkweb.Result, bool) {
		if fn == nil || err == nil {
			return nil, false
		}
		var target E
		if !errors.As(err, &target) {
			return nil, false
		}
		return fn(ctx, target), true
	})
}

// ExceptionHandlerIf 使用谓词匹配哨兵错误或自定义错误条件。
func ExceptionHandlerIf(match ExceptionPredicate, fn func(ctx *arkweb.Context, err error) arkweb.Result) goweb.ErrorMapper {
	return ExceptionHandler(func(ctx *arkweb.Context, err error) (arkweb.Result, bool) {
		if match == nil || fn == nil || !match(err) {
			return nil, false
		}
		return fn(ctx, err), true
	})
}
