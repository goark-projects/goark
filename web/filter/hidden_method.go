package filter

import (
	"context"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
)

const (
	// DefaultHiddenMethodParameter 是 HTML 表单覆盖 HTTP 方法的默认参数名。
	DefaultHiddenMethodParameter = "_method"
	// AttributeOriginalMethod 保存 HiddenHTTPMethod 改写前的 HTTP 方法。
	AttributeOriginalMethod = "goark.web.filter.hidden_method.original_method"
)

// HiddenMethodOption 定制隐藏方法过滤器。
type HiddenMethodOption func(*hiddenMethodConfig)

type hiddenMethodConfig struct {
	parameter      string
	allowedMethods map[string]struct{}
}

type hiddenMethodFilter struct {
	parameter      string
	allowedMethods map[string]struct{}
}

// HiddenHTTPMethod 将 POST 表单参数中的方法覆盖值转换为真实 HTTP 方法。
func HiddenHTTPMethod(options ...HiddenMethodOption) servlet.Filter {
	config := hiddenMethodConfig{
		parameter:      DefaultHiddenMethodParameter,
		allowedMethods: defaultHiddenMethods(),
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return hiddenMethodFilter{
		parameter:      config.parameter,
		allowedMethods: cloneMethodSet(config.allowedMethods),
	}
}

// WithHiddenMethodParameter 设置隐藏方法参数名。
func WithHiddenMethodParameter(name string) HiddenMethodOption {
	return func(config *hiddenMethodConfig) {
		name = strings.TrimSpace(name)
		if name != "" {
			config.parameter = name
		}
	}
}

// WithHiddenMethodAllowedMethods 设置允许覆盖的 HTTP 方法集合。
func WithHiddenMethodAllowedMethods(methods ...string) HiddenMethodOption {
	copied := append([]string(nil), methods...)
	return func(config *hiddenMethodConfig) {
		allowed := make(map[string]struct{}, len(copied))
		for _, method := range copied {
			method = normalizeHiddenMethod(method)
			if method != "" {
				allowed[method] = struct{}{}
			}
		}
		config.allowedMethods = allowed
	}
}

func (f hiddenMethodFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if chain == nil {
		return ErrNilChain
	}
	method, ok, err := f.overrideMethod(req)
	if err != nil {
		return err
	}
	if !ok {
		return chain.Next(ctx, req, res)
	}
	original := req.Method()
	req.SetAttribute(AttributeOriginalMethod, original)
	req.SetMethod(method)
	defer func() {
		req.SetMethod(original)
		req.SetAttribute(AttributeOriginalMethod, nil)
	}()
	return chain.Next(ctx, req, res)
}

func (f hiddenMethodFilter) overrideMethod(req *servlet.Request) (string, bool, error) {
	if !strings.EqualFold(req.Method(), http.MethodPost) {
		return "", false, nil
	}
	method, ok, err := req.Parameter(f.parameter)
	if err != nil {
		return "", false, err
	}
	if !ok {
		method, ok = hiddenMultipartMethod(req, f.parameter)
	}
	method = normalizeHiddenMethod(method)
	if !ok || method == "" || !f.methodAllowed(method) {
		return "", false, nil
	}
	return method, true, nil
}

func (f hiddenMethodFilter) methodAllowed(method string) bool {
	_, ok := f.allowedMethods[method]
	return ok
}

func hiddenMultipartMethod(req *servlet.Request, parameter string) (string, bool) {
	form, ok := servletmultipart.Current(req)
	if !ok || form == nil {
		return "", false
	}
	value := form.Value(parameter)
	return value, value != ""
}

func defaultHiddenMethods() map[string]struct{} {
	return map[string]struct{}{
		http.MethodPut:    {},
		http.MethodPatch:  {},
		http.MethodDelete: {},
	}
}

func cloneMethodSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return map[string]struct{}{}
	}
	dst := make(map[string]struct{}, len(src))
	for method := range src {
		dst[method] = struct{}{}
	}
	return dst
}

func normalizeHiddenMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return ""
	}
	for i := 0; i < len(method); i++ {
		if method[i] < 'A' || method[i] > 'Z' {
			return ""
		}
	}
	return method
}
