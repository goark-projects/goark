package mvc

import (
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

const (
	// ForwardViewNamePrefix 表示 Spring 风格服务端转发视图名前缀。
	ForwardViewNamePrefix = "forward:"
)

type forwardResult struct {
	target string
}

func forwardResultFromViewName(viewName string) (arkweb.Result, bool) {
	target, ok := forwardLocationFromViewName(viewName)
	if !ok {
		return nil, false
	}
	return forwardResult{target: target}, true
}

func forwardLocationFromViewName(viewName string) (string, bool) {
	viewName = strings.TrimSpace(viewName)
	if !strings.HasPrefix(viewName, ForwardViewNamePrefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(viewName, ForwardViewNamePrefix)), true
}

func (r forwardResult) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	target := cleanForwardLocation(r.target)
	if target == "" {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), ErrInvalidForwardLocation)
	}
	app, ok := goweb.CurrentWebApp(ctx)
	if !ok {
		return servlet.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), ErrForwardDispatcherUnavailable)
	}
	dispatcher, err := app.RequestDispatcher(target)
	if err != nil {
		return err
	}
	return dispatcher.Forward(ctx.Context(), ctx.Request(), ctx.Response())
}

func cleanForwardLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" || strings.ContainsAny(location, "\r\n") || !strings.HasPrefix(location, "/") {
		return ""
	}
	return location
}
