package problem

import (
	"net/http"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// Write 将 Problem Detail 写入响应。
func (d Detail) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	if ctx.Request() != nil {
		if _, ok := ctx.Request().NegotiateContentType(MediaType); !ok {
			return servlet.NewHTTPError(http.StatusNotAcceptable, http.StatusText(http.StatusNotAcceptable), nil)
		}
	}
	response := ctx.Response()
	if err := servlet.SetContentType(response, MediaType); err != nil {
		return err
	}
	response.SetStatus(normalizeStatus(d.Status))
	return arkjson.Encode(ctx.JSONCodec(), response.BodyWriter(), d.Body())
}
