package errors

import (
	stderrors "errors"
	"fmt"
)

// Code 表示框架级错误分类，便于上层做稳定分支判断。
type Code string

const (
	CodeInvalidArgument    Code = "INVALID_ARGUMENT"
	CodeAlreadyExists      Code = "ALREADY_EXISTS"
	CodeNotFound           Code = "NOT_FOUND"
	CodeTypeMismatch       Code = "TYPE_MISMATCH"
	CodeConversion         Code = "CONVERSION"
	CodeResource           Code = "RESOURCE"
	CodeCircularDependency Code = "CIRCULAR_DEPENDENCY"
	CodeCreation           Code = "CREATION"
	CodeConflict           Code = "CONFLICT"
	CodeLifecycle          Code = "LIFECYCLE"
	CodeClosed             Code = "CLOSED"
)

// Error 是 Goark 公开错误类型，保留错误码与底层原因。
type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e != nil && t != nil && e.Code == t.Code
}

// New 创建无底层原因的框架错误。
func New(code Code, message string) error {
	return &Error{Code: code, Message: message}
}

// Newf 创建带格式化消息的框架错误。
func Newf(code Code, format string, args ...any) error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap 创建包含底层原因的框架错误。
func Wrap(code Code, cause error, message string) error {
	if cause == nil {
		return New(code, message)
	}
	return &Error{Code: code, Message: message, Cause: cause}
}

// Wrapf 创建包含底层原因与格式化消息的框架错误。
func Wrapf(code Code, cause error, format string, args ...any) error {
	return Wrap(code, cause, fmt.Sprintf(format, args...))
}

// Is 判断错误链中是否包含指定错误码。
func Is(err error, code Code) bool {
	return stderrors.Is(err, &Error{Code: code})
}

// CodeOf 返回错误链中第一个 Goark 错误码。
func CodeOf(err error) (Code, bool) {
	var target *Error
	if stderrors.As(err, &target) && target != nil {
		return target.Code, true
	}
	return "", false
}
