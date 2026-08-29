package mvc

import (
	"net/http"
	"strings"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

const (
	// RedirectViewNamePrefix 表示 Spring 风格重定向视图名前缀。
	RedirectViewNamePrefix = "redirect:"
)

func redirectResultFromViewName(ctx *arkweb.Context, statusCode int, viewName string) (arkweb.Result, bool) {
	location, ok := redirectLocationFromViewName(viewName)
	if !ok {
		return nil, false
	}
	if status, ok := redirectStatus(ctx, statusCode); ok {
		return goweb.Redirect(location, goweb.WithRedirectStatus(status)), true
	}
	return goweb.Redirect(location), true
}

func redirectLocationFromViewName(viewName string) (string, bool) {
	viewName = strings.TrimSpace(viewName)
	if !strings.HasPrefix(viewName, RedirectViewNamePrefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(viewName, RedirectViewNamePrefix)), true
}

func redirectStatus(ctx *arkweb.Context, statusCode int) (int, bool) {
	if isRedirectStatus(statusCode) {
		return statusCode, true
	}
	if statusCode != 0 {
		return 0, false
	}
	status, ok := responseStatusFromContext(ctx)
	if !ok || !isRedirectStatus(status) {
		return 0, false
	}
	return status, true
}

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}
