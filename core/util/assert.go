package util

import (
	"strings"

	arkerrors "goark.dev/goark/errors"
)

// AssertTrue 校验表达式必须为 true。
func AssertTrue(condition bool, message string) error {
	if condition {
		return nil
	}
	return arkerrors.New(arkerrors.CodeInvalidArgument, message)
}

// AssertNotBlank 校验字符串不能是空白。
func AssertNotBlank(value string, message string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return arkerrors.New(arkerrors.CodeInvalidArgument, message)
}

// RequireNotBlank 返回去除首尾空白后的字符串，空白时报错。
func RequireNotBlank(value string, message string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", arkerrors.New(arkerrors.CodeInvalidArgument, message)
	}
	return value, nil
}
