package mvc

import (
	"strings"

	arkweb "goark.dev/arkarta/web"
)

// PathVariableMap 绑定全部路径变量。
func PathVariableMap(ctx *arkweb.Context) (map[string]string, error) {
	if ctx == nil {
		return nil, arkweb.ErrNilContext
	}
	values := ctx.PathValues()
	if len(values) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(values))
	for name, value := range values {
		out[name] = stripMatrixSegment(value)
	}
	return out, nil
}

// RequestParamMap 绑定全部请求参数，每个名称取第一个值。
func RequestParamMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// RequestParamValuesMap 绑定全部请求参数，保留每个名称的全部值。
func RequestParamValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

// RequestHeaderMap 绑定全部请求头，每个名称取第一个值。
func RequestHeaderMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestHeaders(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// RequestHeaderValuesMap 绑定全部请求头，保留每个名称的全部值。
func RequestHeaderValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestHeaders(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

// CookieValueMap 绑定全部 Cookie，每个名称取第一个值。
func CookieValueMap(ctx *arkweb.Context) (map[string]string, error) {
	values, err := requestCookieValues(ctx)
	if err != nil {
		return nil, err
	}
	return firstStringValueMap(values), nil
}

// CookieValueValuesMap 绑定全部 Cookie，保留每个名称的全部值。
func CookieValueValuesMap(ctx *arkweb.Context) (map[string][]string, error) {
	values, err := requestCookieValues(ctx)
	if err != nil {
		return nil, err
	}
	return cloneStringValuesMap(values), nil
}

func requestHeaders(ctx *arkweb.Context) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	return cloneStringValuesMap(ctx.Request().Header()), nil
}

func requestCookieValues(ctx *arkweb.Context) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	cookies := ctx.Request().HTTPRequest().Cookies()
	if len(cookies) == 0 {
		return map[string][]string{}, nil
	}
	out := make(map[string][]string, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		out[name] = append(out[name], cookie.Value)
	}
	return out, nil
}

func firstStringValueMap(values map[string][]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for name, list := range values {
		if len(list) > 0 {
			out[name] = list[0]
		}
	}
	return out
}

func cloneStringValuesMap(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(values))
	for name, list := range values {
		out[name] = append([]string(nil), list...)
	}
	return out
}
