package problem

import (
	"net/http"
	"strings"
)

const (
	// MediaType 是 Problem Details 的标准 JSON 媒体类型。
	MediaType = "application/problem+json"
	// TypeAboutBlank 是未指定错误类型时的标准默认类型。
	TypeAboutBlank = "about:blank"
)

// Detail 表示一次 HTTP API 错误的机器可读描述。
type Detail struct {
	Type       string
	Title      string
	Status     int
	Detail     string
	Instance   string
	Extensions map[string]any
}

// Option 定制 Problem Detail。
type Option func(*Detail)

// New 创建 Problem Detail。
func New(statusCode int, options ...Option) Detail {
	detail := Detail{
		Type:   TypeAboutBlank,
		Status: normalizeStatus(statusCode),
	}
	detail.Title = http.StatusText(detail.Status)
	if detail.Title == "" {
		detail.Title = "HTTP Error"
	}
	for _, option := range options {
		if option != nil {
			option(&detail)
		}
	}
	if strings.TrimSpace(detail.Type) == "" {
		detail.Type = TypeAboutBlank
	}
	if strings.TrimSpace(detail.Title) == "" {
		detail.Title = http.StatusText(detail.Status)
	}
	return detail
}

// WithType 设置 Problem 类型 URI。
func WithType(value string) Option {
	return func(detail *Detail) {
		if value = strings.TrimSpace(value); value != "" {
			detail.Type = value
		}
	}
}

// WithTitle 设置简短错误标题。
func WithTitle(value string) Option {
	return func(detail *Detail) {
		if value = strings.TrimSpace(value); value != "" {
			detail.Title = value
		}
	}
}

// WithDetail 设置面向客户端的错误说明。
func WithDetail(value string) Option {
	return func(detail *Detail) {
		detail.Detail = strings.TrimSpace(value)
	}
}

// WithInstance 设置本次错误实例 URI。
func WithInstance(value string) Option {
	return func(detail *Detail) {
		if value = strings.TrimSpace(value); value != "" {
			detail.Instance = value
		}
	}
}

// WithExtension 设置扩展字段。
func WithExtension(name string, value any) Option {
	return func(detail *Detail) {
		name = strings.TrimSpace(name)
		if name == "" || reservedExtensionName(name) {
			return
		}
		if detail.Extensions == nil {
			detail.Extensions = make(map[string]any, 1)
		}
		detail.Extensions[name] = value
	}
}

// WithExtensions 合并扩展字段。
func WithExtensions(values map[string]any) Option {
	return func(detail *Detail) {
		for name, value := range values {
			WithExtension(name, value)(detail)
		}
	}
}

// Body 返回可直接 JSON 编码的顶层 Problem 对象。
func (d Detail) Body() map[string]any {
	body := make(map[string]any, 5+len(d.Extensions))
	if d.Type != "" {
		body["type"] = d.Type
	}
	if d.Title != "" {
		body["title"] = d.Title
	}
	status := normalizeStatus(d.Status)
	body["status"] = status
	if d.Detail != "" {
		body["detail"] = d.Detail
	}
	if d.Instance != "" {
		body["instance"] = d.Instance
	}
	for name, value := range d.Extensions {
		if !reservedExtensionName(name) {
			body[name] = value
		}
	}
	return body
}

func reservedExtensionName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "type", "title", "status", "detail", "instance":
		return true
	default:
		return false
	}
}

func normalizeStatus(statusCode int) int {
	if statusCode < 100 || statusCode > 999 {
		return http.StatusInternalServerError
	}
	return statusCode
}
