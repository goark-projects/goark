package mvc

import (
	arkweb "goark.dev/arkarta/web"
)

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

func requestHeaders(ctx *arkweb.Context) (map[string][]string, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	return cloneStringValuesMap(ctx.Request().Header()), nil
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
