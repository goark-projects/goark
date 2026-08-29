package client

import (
	"net/http"
	"strings"
)

// WithDefaultCookie 追加每次请求都携带的 Cookie。
func WithDefaultCookie(cookie *http.Cookie) Option {
	cookie, err := cleanCookie(cookie)
	return func(client *Client) error {
		if err != nil {
			return err
		}
		client.defaultCookies = append(client.defaultCookies, cookie)
		return nil
	}
}

// WithDefaultCookieValue 追加每次请求都携带的简单 Cookie。
func WithDefaultCookieValue(name, value string) Option {
	return WithDefaultCookie(&http.Cookie{Name: name, Value: value})
}

// WithCookie 追加单次请求 Cookie。
func WithCookie(cookie *http.Cookie) RequestOption {
	cookie, err := cleanCookie(cookie)
	return func(config *requestConfig) error {
		if err != nil {
			return err
		}
		config.cookies = append(config.cookies, cookie)
		return nil
	}
}

// WithCookieValue 追加单次请求简单 Cookie。
func WithCookieValue(name, value string) RequestOption {
	return WithCookie(&http.Cookie{Name: name, Value: value})
}

func cleanCookie(cookie *http.Cookie) (*http.Cookie, error) {
	if cookie == nil {
		return nil, ErrInvalidCookie
	}
	copied := cloneCookie(cookie)
	copied.Name = strings.TrimSpace(copied.Name)
	if err := copied.Valid(); err != nil {
		return nil, ErrInvalidCookie
	}
	if value := copied.String(); value == "" || strings.ContainsAny(value, "\r\n") {
		return nil, ErrInvalidCookie
	}
	return copied, nil
}

func cloneCookie(cookie *http.Cookie) *http.Cookie {
	if cookie == nil {
		return nil
	}
	copied := *cookie
	copied.Unparsed = append([]string(nil), cookie.Unparsed...)
	return &copied
}

func addCookies(request *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
}
