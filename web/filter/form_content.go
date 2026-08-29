package filter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	// AttributeFormContent 保存 FormContent 过滤器解析出的 URL 编码表单参数。
	AttributeFormContent = "goark.web.filter.form_content.values"
	// DefaultFormContentMaxBodyBytes 是 FormContent 默认允许缓存的最大请求体。
	DefaultFormContentMaxBodyBytes int64 = 1 << 20
)

// FormContentOption 定制 FormContent 过滤器。
type FormContentOption func(*formContentConfig)

type formContentConfig struct {
	methods      map[string]struct{}
	maxBodyBytes int64
}

type formContentFilter struct {
	methods      map[string]struct{}
	maxBodyBytes int64
}

// FormContent 解析非 POST 的 URL 编码表单体并暴露给 MVC 参数绑定。
func FormContent(options ...FormContentOption) servlet.Filter {
	config := formContentConfig{
		methods:      defaultFormContentMethods(),
		maxBodyBytes: DefaultFormContentMaxBodyBytes,
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return formContentFilter{
		methods:      cloneHTTPMethodSet(config.methods),
		maxBodyBytes: config.maxBodyBytes,
	}
}

// WithFormContentMethods 设置允许解析 URL 编码表单体的 HTTP 方法集合。
func WithFormContentMethods(methods ...string) FormContentOption {
	copied := append([]string(nil), methods...)
	return func(config *formContentConfig) {
		allowed := make(map[string]struct{}, len(copied))
		for _, method := range copied {
			method = normalizeHTTPMethod(method)
			if method != "" {
				allowed[method] = struct{}{}
			}
		}
		config.methods = allowed
	}
}

// WithFormContentMaxBodyBytes 设置 FormContent 最大缓存请求体字节数。
func WithFormContentMaxBodyBytes(size int64) FormContentOption {
	return func(config *formContentConfig) {
		if size >= 0 {
			config.maxBodyBytes = size
		}
	}
}

func (f formContentFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if chain == nil {
		return ErrNilChain
	}
	if !f.shouldParse(req) {
		return chain.Next(ctx, req, res)
	}
	values, ok, err := parseFormContent(req.HTTPRequest(), f.maxBodyBytes)
	if err != nil {
		return err
	}
	if ok {
		req.SetAttribute(AttributeFormContent, values)
	}
	return chain.Next(ctx, req, res)
}

func (f formContentFilter) shouldParse(req *servlet.Request) bool {
	if _, ok := f.methods[normalizeHTTPMethod(req.Method())]; !ok {
		return false
	}
	return strings.EqualFold(req.ContentType(), "application/x-www-form-urlencoded")
}

// FormContentValue 返回 FormContent 过滤器解析出的第一个表单参数。
func FormContentValue(req *servlet.Request, name string) (string, bool) {
	values, ok := FormContentValues(req)
	if !ok {
		return "", false
	}
	list, ok := values[name]
	if !ok || len(list) == 0 {
		return "", false
	}
	return list[0], true
}

// FormContentValues 返回 FormContent 过滤器解析出的表单参数副本。
func FormContentValues(req *servlet.Request) (url.Values, bool) {
	if req == nil {
		return nil, false
	}
	value, ok := req.Attribute(AttributeFormContent)
	if !ok {
		return nil, false
	}
	values, ok := value.(url.Values)
	if !ok {
		return nil, false
	}
	return cloneFormContentValues(values), true
}

func parseFormContent(request *http.Request, maxBodyBytes int64) (url.Values, bool, error) {
	if request == nil || request.Body == nil || request.Body == http.NoBody {
		return nil, false, nil
	}
	if maxBodyBytes >= 0 && request.ContentLength > maxBodyBytes {
		return nil, false, formContentTooLarge()
	}
	body, err := readAndRestoreBody(request, maxBodyBytes)
	if err != nil {
		return nil, false, err
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, false, servlet.NewHTTPError(http.StatusBadRequest, "表单请求体解析失败", err)
	}
	return values, true, nil
}

func readAndRestoreBody(request *http.Request, maxBodyBytes int64) ([]byte, error) {
	reader := io.Reader(request.Body)
	if maxBodyBytes >= 0 {
		reader = io.LimitReader(request.Body, maxBodyBytes+1)
	}
	body, err := io.ReadAll(reader)
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, servlet.NewHTTPError(http.StatusBadRequest, "表单请求体读取失败", err)
	}
	if maxBodyBytes >= 0 && int64(len(body)) > maxBodyBytes {
		return nil, formContentTooLarge()
	}
	return body, nil
}

func formContentTooLarge() error {
	return servlet.NewHTTPError(http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge), nil)
}

func defaultFormContentMethods() map[string]struct{} {
	return map[string]struct{}{
		http.MethodPut:    {},
		http.MethodPatch:  {},
		http.MethodDelete: {},
	}
}

func cloneHTTPMethodSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return map[string]struct{}{}
	}
	dst := make(map[string]struct{}, len(src))
	for method := range src {
		if method = normalizeHTTPMethod(method); method != "" {
			dst[method] = struct{}{}
		}
	}
	return dst
}

func cloneFormContentValues(src url.Values) url.Values {
	if len(src) == 0 {
		return url.Values{}
	}
	dst := make(url.Values, len(src))
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
	return dst
}

func normalizeHTTPMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}
