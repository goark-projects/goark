package message

import (
	"errors"
	"mime"
	"net/http"
	"reflect"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

func ensureContext(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	return nil
}

func ensureReadableContext(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	return nil
}

func defaultMediaType(mediaType, fallback string) string {
	mediaType = strings.TrimSpace(mediaType)
	if mediaType == "" {
		return fallback
	}
	return mediaType
}

func mediaTypeMatches(left, right string) bool {
	leftType, _, leftOK := parseMediaType(left)
	rightType, _, rightOK := parseMediaType(right)
	if !leftOK || !rightOK {
		return false
	}
	return leftType == rightType
}

func structuredJSONType(mediaType string) bool {
	typ, _, ok := parseMediaType(mediaType)
	if !ok {
		return false
	}
	main, subtype, ok := strings.Cut(typ, "/")
	return ok && main == "application" && strings.HasSuffix(subtype, "+json")
}

func parseMediaType(value string) (string, map[string]string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, false
	}
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil {
		return "", nil, false
	}
	return strings.ToLower(mediaType), params, true
}

func nilTarget(target any) bool {
	if target == nil {
		return true
	}
	value := reflect.ValueOf(target)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func normalizeStatus(statusCode, fallback int) int {
	if statusCode == 0 {
		return fallback
	}
	if statusCode < 100 || statusCode > 999 {
		return http.StatusInternalServerError
	}
	return statusCode
}

func joinErrors(left, right error) error {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	return errors.Join(left, right)
}
