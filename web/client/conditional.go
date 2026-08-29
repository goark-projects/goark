package client

import (
	"net/http"
	"strings"
	"time"
)

// WithIfNoneMatch 设置 If-None-Match 条件请求头。
func WithIfNoneMatch(etags ...string) RequestOption {
	values, err := cleanConditionalETags(etags)
	return func(config *requestConfig) error {
		if err != nil {
			return err
		}
		config.headers.Set("If-None-Match", strings.Join(values, ", "))
		return nil
	}
}

// WithIfModifiedSince 设置 If-Modified-Since 条件请求头。
func WithIfModifiedSince(value time.Time) RequestOption {
	value = truncateHTTPTime(value)
	return func(config *requestConfig) error {
		if value.IsZero() {
			return ErrInvalidHeader
		}
		config.headers.Set("If-Modified-Since", value.Format(http.TimeFormat))
		return nil
	}
}

func cleanConditionalETags(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrInvalidHeader
	}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.ContainsAny(value, "\r\n") {
			return nil, ErrInvalidHeader
		}
		if value == "*" {
			if len(values) != 1 {
				return nil, ErrInvalidHeader
			}
			return []string{"*"}, nil
		}
		if strings.HasPrefix(value, "W/") {
			etag, err := cleanConditionalETag(value[2:])
			if err != nil {
				return nil, err
			}
			cleaned = append(cleaned, "W/"+etag)
			continue
		}
		etag, err := cleanConditionalETag(value)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, etag)
	}
	if len(cleaned) == 0 {
		return nil, ErrInvalidHeader
	}
	return cleaned, nil
}

func cleanConditionalETag(value string) (string, error) {
	value = strings.TrimSpace(value)
	if isQuotedConditionalETag(value) {
		return value, nil
	}
	if value == "" || strings.ContainsAny(value, "\"\\") {
		return "", ErrInvalidHeader
	}
	return `"` + value + `"`, nil
}

func isQuotedConditionalETag(value string) bool {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	return !strings.ContainsAny(value[1:len(value)-1], "\"\\\r\n")
}

func truncateHTTPTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC().Truncate(time.Second)
}
