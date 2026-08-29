package web

import (
	"strconv"
	"strings"
	"time"
)

// CacheControl 表示可写入 Cache-Control 响应头的缓存指令集合。
type CacheControl struct {
	directives []string
}

// CacheControlValue 使用原始响应头值创建缓存指令。
func CacheControlValue(value string) CacheControl {
	value = cleanHeaderValue(value)
	if value == "" {
		return CacheControl{}
	}
	return CacheControl{directives: []string{value}}
}

// NoCache 创建 no-cache 缓存指令。
func NoCache() CacheControl {
	return CacheControl{}.NoCache()
}

// NoStore 创建 no-store 缓存指令。
func NoStore() CacheControl {
	return CacheControl{}.NoStore()
}

// MaxAge 创建 max-age 缓存指令。
func MaxAge(age time.Duration) CacheControl {
	return CacheControl{}.MaxAge(age)
}

// NoCache 追加 no-cache 指令。
func (c CacheControl) NoCache() CacheControl {
	return c.appendDirective("no-cache")
}

// NoStore 追加 no-store 指令。
func (c CacheControl) NoStore() CacheControl {
	return c.appendDirective("no-store")
}

// Public 追加 public 指令。
func (c CacheControl) Public() CacheControl {
	return c.appendDirective("public")
}

// Private 追加 private 指令。
func (c CacheControl) Private() CacheControl {
	return c.appendDirective("private")
}

// NoTransform 追加 no-transform 指令。
func (c CacheControl) NoTransform() CacheControl {
	return c.appendDirective("no-transform")
}

// MustRevalidate 追加 must-revalidate 指令。
func (c CacheControl) MustRevalidate() CacheControl {
	return c.appendDirective("must-revalidate")
}

// ProxyRevalidate 追加 proxy-revalidate 指令。
func (c CacheControl) ProxyRevalidate() CacheControl {
	return c.appendDirective("proxy-revalidate")
}

// Immutable 追加 immutable 指令。
func (c CacheControl) Immutable() CacheControl {
	return c.appendDirective("immutable")
}

// MaxAge 追加 max-age 指令。
func (c CacheControl) MaxAge(age time.Duration) CacheControl {
	seconds, ok := cacheSeconds(age)
	if !ok {
		return c
	}
	return c.appendDirective("max-age=" + strconv.FormatInt(seconds, 10))
}

// SMaxAge 追加 s-maxage 指令。
func (c CacheControl) SMaxAge(age time.Duration) CacheControl {
	seconds, ok := cacheSeconds(age)
	if !ok {
		return c
	}
	return c.appendDirective("s-maxage=" + strconv.FormatInt(seconds, 10))
}

// HeaderValue 返回可写入响应头的缓存指令值。
func (c CacheControl) HeaderValue() string {
	if len(c.directives) == 0 {
		return ""
	}
	return strings.Join(c.directives, ", ")
}

func (c CacheControl) appendDirective(value string) CacheControl {
	value = cleanHeaderValue(value)
	if value == "" {
		return c
	}
	for _, item := range c.directives {
		if strings.EqualFold(item, value) {
			return c
		}
	}
	next := make([]string, 0, len(c.directives)+1)
	next = append(next, c.directives...)
	next = append(next, value)
	c.directives = next
	return c
}

func cacheSeconds(age time.Duration) (int64, bool) {
	if age < 0 {
		return 0, false
	}
	return int64(age / time.Second), true
}
