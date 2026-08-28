package stream

import (
	"net/http"
	"strings"
)

// Option 定制流式响应。
type Option func(*Result)

// WithStatus 设置流式响应状态码。
func WithStatus(status int) Option {
	return func(result *Result) {
		if status >= 100 && status <= 999 {
			result.status = status
		}
	}
}

// WithHeader 追加流式响应头。
func WithHeader(name string, values ...string) Option {
	cleanName := http.CanonicalHeaderKey(strings.TrimSpace(name))
	cleanValues := cleanHeaderValues(values)
	return func(result *Result) {
		if cleanName == "" || len(cleanValues) == 0 {
			return
		}
		for _, value := range cleanValues {
			result.headers.Add(cleanName, value)
		}
	}
}

func cleanHeaderValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.ContainsAny(value, "\r\n") {
			continue
		}
		out = append(out, value)
	}
	return out
}
