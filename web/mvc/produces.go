package mvc

import (
	"strings"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

func jsonResult(ctx *arkweb.Context, statusCode int, value any) arkweb.Result {
	if mediaType, ok := selectedProducesMediaType(ctx); ok {
		return goweb.Message(statusCode, value, mediaType)
	}
	return arkweb.JSON(statusCode, value)
}

func entityResult[T any](ctx *arkweb.Context, entity goweb.ResponseEntity[T]) arkweb.Result {
	if mediaType, ok := selectedProducesMediaType(ctx); ok && !entity.HasMediaTypes() {
		return entity.WithContentType(mediaType)
	}
	return entity
}

func selectedProducesMediaType(ctx *arkweb.Context) (string, bool) {
	if ctx == nil || ctx.Request() == nil {
		return "", false
	}
	value, ok := ctx.Request().Attribute(AttributeProducesMediaType)
	if !ok {
		return "", false
	}
	mediaType, ok := value.(string)
	if !ok {
		return "", false
	}
	mediaType = strings.TrimSpace(mediaType)
	return mediaType, mediaType != ""
}
