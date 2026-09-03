package web

import (
	"net/http"
	"strings"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// CheckNotModified 根据请求条件头判断资源是否未变化。
func CheckNotModified(ctx *arkweb.Context, etag string, lastModified time.Time) bool {
	if ctx == nil || ctx.Request() == nil || ctx.Response() == nil || ctx.Response().Committed() {
		return false
	}
	req := ctx.Request()
	if !conditionalMethodAllowed(req.Method()) {
		return false
	}
	validator := newConditionalValidator(etag, lastModified)
	if validator.empty() {
		return false
	}
	validator.write(ctx.Response().Header())
	if !validator.notModified(req.Header()) {
		return false
	}
	ctx.Response().Header().Delete("Content-Length")
	ctx.Response().SetStatus(http.StatusNotModified)
	return true
}

type conditionalValidator struct {
	etag         string
	lastModified time.Time
}

func newConditionalValidator(etag string, lastModified time.Time) conditionalValidator {
	return conditionalValidator{
		etag:         cleanETag(etag, false),
		lastModified: truncateHTTPTime(lastModified),
	}
}

func (v conditionalValidator) empty() bool {
	return v.etag == "" && v.lastModified.IsZero()
}

func (v conditionalValidator) write(header servlet.Header) {
	if v.etag != "" {
		header.Set("ETag", v.etag)
	}
	if !v.lastModified.IsZero() {
		header.Set("Last-Modified", v.lastModified.UTC().Format(http.TimeFormat))
	}
}

func (v conditionalValidator) notModified(header servlet.Header) bool {
	if len(header.Values("If-None-Match")) > 0 {
		return conditionalETagMatch(header.Values("If-None-Match"), v.etag)
	}
	return conditionalModifiedSince(header.Get("If-Modified-Since"), v.lastModified)
}

func conditionalETagMatch(values []string, etag string) bool {
	if etag == "" {
		return false
	}
	normalized := normalizeConditionalETag(etag)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "*" || normalizeConditionalETag(item) == normalized {
				return true
			}
		}
	}
	return false
}

func normalizeConditionalETag(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	return value
}

func conditionalModifiedSince(value string, lastModified time.Time) bool {
	if value == "" || lastModified.IsZero() {
		return false
	}
	since, err := http.ParseTime(value)
	if err != nil {
		return false
	}
	return !lastModified.After(truncateHTTPTime(since))
}

func conditionalMethodAllowed(method string) bool {
	return strings.EqualFold(method, http.MethodGet) || strings.EqualFold(method, http.MethodHead)
}

func truncateHTTPTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC().Truncate(time.Second)
}
