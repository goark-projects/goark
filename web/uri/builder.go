package uri

import (
	"net/url"
	"strings"
)

// Builder 以不可变方式累积 URI 组件。
type Builder struct {
	scheme   string
	host     string
	path     string
	query    url.Values
	fragment string
}

// New 创建空 URI 构建器。
func New() Builder {
	return Builder{}
}

// From 基于已有 URI 创建构建器。
func From(rawURI string) (Builder, error) {
	rawURI = strings.TrimSpace(rawURI)
	if rawURI == "" {
		return Builder{}, ErrInvalidURI
	}
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return Builder{}, ErrInvalidURI
	}
	return Builder{
		scheme:   parsed.Scheme,
		host:     parsed.Host,
		path:     parsed.Path,
		query:    parsed.Query(),
		fragment: parsed.Fragment,
	}, nil
}

// Scheme 设置 URI 协议。
func (b Builder) Scheme(value string) Builder {
	b.scheme = cleanComponent(value)
	return b
}

// Host 设置 URI 主机。
func (b Builder) Host(value string) Builder {
	b.host = cleanComponent(value)
	return b
}

// Path 追加 URI 路径。
func (b Builder) Path(value string) Builder {
	b.path = joinPath(b.path, strings.TrimSpace(value))
	return b
}

// ReplacePath 替换 URI 路径。
func (b Builder) ReplacePath(value string) Builder {
	b.path = normalizePath(strings.TrimSpace(value))
	return b
}

// QueryParam 追加查询参数值。
func (b Builder) QueryParam(name string, values ...string) Builder {
	name = strings.TrimSpace(name)
	if name == "" {
		return b
	}
	query := cloneQuery(b.query)
	for _, value := range values {
		query.Add(name, value)
	}
	b.query = query
	return b
}

// ReplaceQueryParam 替换查询参数值；不传值时删除该参数。
func (b Builder) ReplaceQueryParam(name string, values ...string) Builder {
	name = strings.TrimSpace(name)
	if name == "" {
		return b
	}
	query := cloneQuery(b.query)
	query.Del(name)
	for _, value := range values {
		query.Add(name, value)
	}
	b.query = query
	return b
}

// ClearQuery 删除所有查询参数。
func (b Builder) ClearQuery() Builder {
	b.query = nil
	return b
}

// Fragment 设置 URI fragment。
func (b Builder) Fragment(value string) Builder {
	b.fragment = strings.TrimPrefix(cleanComponent(value), "#")
	return b
}

// Build 构建 URI 字符串。
func (b Builder) Build() string {
	value, _ := b.build(nil)
	return value
}

// BuildAndExpand 构建 URI，并使用变量展开路径模板。
func (b Builder) BuildAndExpand(variables map[string]string) (string, error) {
	return b.build(variables)
}

func (b Builder) build(variables map[string]string) (string, error) {
	path, rawPath, err := expandPath(b.path, variables)
	if err != nil {
		return "", err
	}
	out := url.URL{
		Scheme:   b.scheme,
		Host:     b.host,
		Path:     path,
		RawPath:  rawPath,
		RawQuery: cloneQuery(b.query).Encode(),
		Fragment: b.fragment,
	}
	return out.String(), nil
}

func cleanComponent(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func cloneQuery(src url.Values) url.Values {
	if len(src) == 0 {
		return make(url.Values)
	}
	dst := make(url.Values, len(src))
	for name, values := range src {
		dst[name] = append([]string(nil), values...)
	}
	return dst
}
