package mvc

import (
	"net/http"
	"net/url"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	gowebfilter "goark.dev/goark/web/filter"
)

func requestParameters(ctx *arkweb.Context) (url.Values, error) {
	if ctx == nil || ctx.Request() == nil {
		return nil, arkweb.ErrNilContext
	}
	values, err := ctx.Request().Parameters()
	if err != nil {
		return nil, err
	}
	return appendFormContentValues(values, ctx.Request()), nil
}

func requestParameterValue(ctx *arkweb.Context, name string) (string, bool, error) {
	values, ok, err := requestParameterValues(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	return values[0], true, nil
}

func requestParameterValues(ctx *arkweb.Context, name string) ([]string, bool, error) {
	values, err := requestParameters(ctx)
	if err != nil {
		return nil, false, err
	}
	list, ok := values[name]
	if !ok {
		list, ok = values[emptyArrayRequestParameterName(name)]
	}
	if !ok || len(list) == 0 {
		return nil, false, nil
	}
	return append([]string(nil), list...), true, nil
}

func emptyArrayRequestParameterName(name string) string {
	if name == "" || strings.HasSuffix(name, "[]") {
		return name
	}
	return name + "[]"
}

func appendFormContentValues(values url.Values, req *servlet.Request) url.Values {
	if !shouldAppendFormContent(req) {
		return values
	}
	form, ok := gowebfilter.FormContentValues(req)
	if !ok || len(form) == 0 {
		return values
	}
	if values == nil {
		values = url.Values{}
	}
	for name, list := range form {
		values[name] = append(values[name], list...)
	}
	return values
}

func shouldAppendFormContent(req *servlet.Request) bool {
	return req != nil && strings.EqualFold(req.Method(), http.MethodDelete)
}
