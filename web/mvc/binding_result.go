package mvc

import (
	"errors"

	"goark.dev/arkarta/validation"
)

// BindingResult 表示绑定成功后的结构体验证结果。
type BindingResult struct {
	result           validation.Result
	bindingError     error
	suppressedFields []string
}

// NewBindingResult 创建绑定结果。
func NewBindingResult(result validation.Result, bindingErrors ...error) BindingResult {
	return BindingResult{result: result, bindingError: errors.Join(bindingErrors...)}
}

func newBindingErrorResult(err error) BindingResult {
	return BindingResult{bindingError: err}
}

// Valid 返回绑定对象是否通过验证。
func (r BindingResult) Valid() bool {
	return r.bindingError == nil && r.result.Valid()
}

// HasErrors 返回绑定对象是否存在验证错误。
func (r BindingResult) HasErrors() bool {
	return !r.Valid()
}

// Result 返回底层 Arkarta 验证结果。
func (r BindingResult) Result() validation.Result {
	return r.result
}

// Violations 返回全部验证失败项。
func (r BindingResult) Violations() []validation.Violation {
	return r.result.Violations()
}

// FieldError 返回指定字段的第一个验证失败项。
func (r BindingResult) FieldError(path string) (validation.Violation, bool) {
	for _, violation := range r.result.Violations() {
		if violation.Path() == path {
			return violation, true
		}
	}
	return validation.Violation{}, false
}

// FieldErrors 返回指定字段的全部验证失败项。
func (r BindingResult) FieldErrors(path string) []validation.Violation {
	violations := r.result.Violations()
	if len(violations) == 0 {
		return nil
	}
	matched := make([]validation.Violation, 0, 1)
	for _, violation := range violations {
		if violation.Path() == path {
			matched = append(matched, violation)
		}
	}
	return matched
}

// BindingError 返回绑定阶段错误；没有绑定错误时返回 nil。
func (r BindingResult) BindingError() error {
	return r.bindingError
}

// SuppressedFields 返回被字段绑定规则拒绝的字段名快照。
func (r BindingResult) SuppressedFields() []string {
	if len(r.suppressedFields) == 0 {
		return nil
	}
	return append([]string(nil), r.suppressedFields...)
}

// Err 将验证失败结果转换为 error。
func (r BindingResult) Err() error {
	return errors.Join(r.bindingError, r.result.Error())
}

func (r BindingResult) withSuppressedFields(fields []string) BindingResult {
	if len(fields) == 0 {
		return r
	}
	r.suppressedFields = append([]string(nil), fields...)
	return r
}
