package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WithCookie 追加 Set-Cookie 响应头。
func (e ResponseEntity[T]) WithCookie(cookie *http.Cookie) ResponseEntity[T] {
	if cookie == nil {
		return e
	}
	value := cleanHeaderValue(cookie.String())
	if value == "" {
		return e
	}
	return e.WithAddedHeader("Set-Cookie", value)
}

// WithCacheControl 设置 Cache-Control 响应头。
func (e ResponseEntity[T]) WithCacheControl(control CacheControl) ResponseEntity[T] {
	return e.WithCacheControlValue(control.HeaderValue())
}

// WithCacheControlValue 设置原始 Cache-Control 响应头值。
func (e ResponseEntity[T]) WithCacheControlValue(value string) ResponseEntity[T] {
	return e.WithHeader("Cache-Control", value)
}

// WithETag 设置强 ETag 响应头，未加引号的值会自动规范化。
func (e ResponseEntity[T]) WithETag(value string) ResponseEntity[T] {
	etag := cleanETag(value, false)
	if etag == "" {
		return e
	}
	return e.WithHeader("ETag", etag)
}

// WithWeakETag 设置弱 ETag 响应头。
func (e ResponseEntity[T]) WithWeakETag(value string) ResponseEntity[T] {
	etag := cleanETag(value, true)
	if etag == "" {
		return e
	}
	return e.WithHeader("ETag", etag)
}

// WithLastModified 设置 Last-Modified 响应头。
func (e ResponseEntity[T]) WithLastModified(value time.Time) ResponseEntity[T] {
	if value.IsZero() {
		return e
	}
	return e.WithHeader("Last-Modified", value.UTC().Format(http.TimeFormat))
}

// WithLocation 设置 Location 响应头。
func (e ResponseEntity[T]) WithLocation(location string) ResponseEntity[T] {
	return e.WithHeader("Location", location)
}

// WithContentLength 设置 Content-Length 响应头。
func (e ResponseEntity[T]) WithContentLength(length int64) ResponseEntity[T] {
	if length < 0 {
		return e
	}
	return e.WithHeader("Content-Length", strconv.FormatInt(length, 10))
}

// WithAllow 设置 Allow 响应头。
func (e ResponseEntity[T]) WithAllow(methods ...string) ResponseEntity[T] {
	values := cleanTokens(methods, strings.ToUpper)
	if len(values) == 0 {
		return e
	}
	return e.WithHeader("Allow", strings.Join(values, ", "))
}

// WithVary 设置 Vary 响应头。
func (e ResponseEntity[T]) WithVary(names ...string) ResponseEntity[T] {
	values := cleanTokens(names, http.CanonicalHeaderKey)
	if len(values) == 0 {
		return e
	}
	return e.WithHeader("Vary", strings.Join(values, ", "))
}

func cleanHeaderName(name string) string {
	name = http.CanonicalHeaderKey(strings.TrimSpace(name))
	if name == "" || !isHTTPToken(name) {
		return ""
	}
	return name
}

func cleanHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func cleanETag(value string, weak bool) string {
	value = cleanHeaderValue(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "W/") {
		weak = true
		value = strings.TrimSpace(value[2:])
	}
	if isQuotedETag(value) {
		if weak {
			return "W/" + value
		}
		return value
	}
	if strings.ContainsAny(value, "\"\\") {
		return ""
	}
	if weak {
		return `W/"` + value + `"`
	}
	return `"` + value + `"`
}

func isQuotedETag(value string) bool {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	return !strings.ContainsAny(value[1:len(value)-1], "\"\\\r\n")
}

func cleanTokens(values []string, normalize func(string) string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalize(strings.TrimSpace(value))
		if value == "" || !isHTTPToken(value) {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isHTTPToken(value string) bool {
	for i := 0; i < len(value); i++ {
		if !isHTTPTokenByte(value[i]) {
			return false
		}
	}
	return value != ""
}

func isHTTPTokenByte(value byte) bool {
	if value >= 'a' && value <= 'z' {
		return true
	}
	if value >= 'A' && value <= 'Z' {
		return true
	}
	if value >= '0' && value <= '9' {
		return true
	}
	switch value {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}
