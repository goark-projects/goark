package problem

import (
	"errors"
	"fmt"
	"net/http"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
)

// Mapper 将处理错误映射为 Problem Details 响应。
type Mapper struct{}

// NewMapper 创建 Problem Details 错误映射器。
func NewMapper() Mapper {
	return Mapper{}
}

// MapError 将常见 Web 错误映射为 Problem Details。
func (Mapper) MapError(ctx *arkweb.Context, err error) arkweb.Result {
	return FromError(ctx, err)
}

// FromError 从错误创建 Problem Detail。
func FromError(ctx *arkweb.Context, err error, options ...Option) Detail {
	statusCode, message, extensions := classifyError(err)
	all := make([]Option, 0, 3+len(options))
	all = append(all,
		WithDetail(message),
		WithInstance(errorInstance(ctx)),
		WithExtension("error", statusCodeName(statusCode)),
	)
	if len(extensions) > 0 {
		all = append(all, WithExtensions(extensions))
	}
	all = append(all, options...)
	return New(statusCode, all...)
}

func classifyError(err error) (int, string, map[string]any) {
	if err == nil {
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), nil
	}
	var validationErr validation.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusUnprocessableEntity, "请求参数校验失败", map[string]any{
			"violations": violationDetails(validationErr.Result()),
		}
	}
	if errors.Is(err, arkjson.ErrPayloadTooLarge) {
		return http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge), nil
	}
	if errors.Is(err, arkweb.ErrUnsupportedMediaType) {
		return http.StatusUnsupportedMediaType, http.StatusText(http.StatusUnsupportedMediaType), nil
	}
	var parameterErr *arkweb.ParameterError
	if errors.As(err, &parameterErr) {
		return http.StatusBadRequest, "请求参数格式非法", nil
	}
	var bindErr *arkweb.BindError
	if errors.As(err, &bindErr) {
		return http.StatusBadRequest, "请求体格式非法", nil
	}
	var statusErr servlet.StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode(), statusErr.PublicMessage(), nil
	}
	return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), nil
}

type violationDetail struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func violationDetails(result validation.Result) []violationDetail {
	violations := result.Violations()
	details := make([]violationDetail, 0, len(violations))
	for _, violation := range violations {
		details = append(details, violationDetail{
			Path:    violation.Path(),
			Code:    violation.Code(),
			Message: violation.Message(),
		})
	}
	return details
}

func errorInstance(ctx *arkweb.Context) string {
	if ctx == nil || ctx.Request() == nil {
		return ""
	}
	return ctx.Request().Path()
}

func statusCodeName(statusCode int) string {
	return fmt.Sprintf("HTTP_%d", normalizeStatus(statusCode))
}
