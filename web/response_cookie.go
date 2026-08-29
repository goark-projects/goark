package web

import (
	"math"
	"net/http"
	"strings"
	"time"
)

// ResponseCookie 表示不可变的响应 Cookie 值对象。
type ResponseCookie struct {
	cookie http.Cookie
}

// NewResponseCookie 创建响应 Cookie。
func NewResponseCookie(name, value string) ResponseCookie {
	return ResponseCookie{
		cookie: http.Cookie{
			Name:  strings.TrimSpace(name),
			Value: value,
		},
	}
}

// WithPath 设置 Cookie Path。
func (c ResponseCookie) WithPath(path string) ResponseCookie {
	c.cookie.Path = strings.TrimSpace(path)
	return c
}

// WithDomain 设置 Cookie Domain。
func (c ResponseCookie) WithDomain(domain string) ResponseCookie {
	c.cookie.Domain = strings.TrimSpace(domain)
	return c
}

// WithMaxAge 设置 Cookie Max-Age；负数表示立即删除。
func (c ResponseCookie) WithMaxAge(maxAge time.Duration) ResponseCookie {
	c.cookie.MaxAge = responseCookieMaxAgeSeconds(maxAge)
	return c
}

// WithExpires 设置 Cookie Expires。
func (c ResponseCookie) WithExpires(expires time.Time) ResponseCookie {
	c.cookie.Expires = expires
	return c
}

// WithSecure 设置 Cookie Secure 标记。
func (c ResponseCookie) WithSecure(secure bool) ResponseCookie {
	c.cookie.Secure = secure
	return c
}

// WithHTTPOnly 设置 Cookie HttpOnly 标记。
func (c ResponseCookie) WithHTTPOnly(httpOnly bool) ResponseCookie {
	c.cookie.HttpOnly = httpOnly
	return c
}

// WithSameSite 设置 Cookie SameSite 策略。
func (c ResponseCookie) WithSameSite(sameSite http.SameSite) ResponseCookie {
	switch sameSite {
	case http.SameSiteDefaultMode,
		http.SameSiteLaxMode,
		http.SameSiteStrictMode,
		http.SameSiteNoneMode:
		c.cookie.SameSite = sameSite
	default:
		c.cookie.SameSite = 0
	}
	return c
}

// Cookie 返回标准库 Cookie 副本；非法 Cookie 返回 nil。
func (c ResponseCookie) Cookie() *http.Cookie {
	if c.cookie.Valid() != nil {
		return nil
	}
	copied := c.cookie
	return &copied
}

// String 返回 Set-Cookie 头值；非法 Cookie 返回空字符串。
func (c ResponseCookie) String() string {
	cookie := c.Cookie()
	if cookie == nil {
		return ""
	}
	return cleanHeaderValue(cookie.String())
}

func responseCookieMaxAgeSeconds(maxAge time.Duration) int {
	if maxAge < 0 {
		return -1
	}
	if maxAge == 0 {
		return 0
	}
	seconds := int64(maxAge / time.Second)
	if maxAge%time.Second != 0 {
		seconds++
	}
	if seconds > int64(math.MaxInt) {
		return math.MaxInt
	}
	return int(seconds)
}
