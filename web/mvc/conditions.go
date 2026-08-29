package mvc

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const (
	// AttributeProducesMediaType 保存 MVC produces 条件协商后的媒体类型。
	AttributeProducesMediaType = "goark.web.mvc.produces.media_type"
)

// Conditions 描述 MVC 路由额外匹配条件。
type Conditions struct {
	Consumes []string
	Produces []string
	Params   []string
	Headers  []string
}

func (c Conditions) wrap(handler arkweb.Handler) arkweb.Handler {
	if handler == nil {
		return nil
	}
	if c.empty() {
		return handler
	}
	return arkweb.HandlerFunc(func(ctx *arkweb.Context) (arkweb.Result, error) {
		if err := c.match(ctx); err != nil {
			return nil, err
		}
		return handler.Handle(ctx)
	})
}

func (c Conditions) empty() bool {
	return len(c.Consumes) == 0 && len(c.Produces) == 0 && len(c.Params) == 0 && len(c.Headers) == 0
}

func (c Conditions) specificity() int {
	return len(c.Params)*8 + len(c.Headers)*8 + len(c.Consumes)*4 + len(c.Produces)*4
}

func (c Conditions) match(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Request() == nil {
		return arkweb.ErrNilContext
	}
	req := ctx.Request()
	if err := matchConsumes(req, c.Consumes); err != nil {
		return err
	}
	if err := matchProduces(req, c.Produces); err != nil {
		return err
	}
	if err := matchParameterExpressions(req, c.Params); err != nil {
		return err
	}
	return matchHeaderExpressions(req, c.Headers)
}

func matchConsumes(req *servlet.Request, consumes []string) error {
	if len(consumes) == 0 {
		return nil
	}
	contentType, _, err := mime.ParseMediaType(req.Header().Get("Content-Type"))
	if err != nil || contentType == "" {
		return servlet.NewHTTPError(http.StatusUnsupportedMediaType, http.StatusText(http.StatusUnsupportedMediaType), err)
	}
	for _, mediaType := range consumes {
		if mediaTypeMatches(mediaType, contentType) {
			return nil
		}
	}
	return servlet.NewHTTPError(http.StatusUnsupportedMediaType, http.StatusText(http.StatusUnsupportedMediaType), nil)
}

func matchProduces(req *servlet.Request, produces []string) error {
	if len(produces) == 0 {
		return nil
	}
	selected, ok := req.NegotiateContentType(produces...)
	if !ok {
		return servlet.NewHTTPError(http.StatusNotAcceptable, http.StatusText(http.StatusNotAcceptable), nil)
	}
	req.SetAttribute(AttributeProducesMediaType, selected)
	return nil
}

func matchParameterExpressions(req *servlet.Request, expressions []string) error {
	for _, text := range expressions {
		expr, ok := parseConditionExpression(text)
		if !ok {
			continue
		}
		value, exists, err := req.Parameter(expr.name)
		if err != nil {
			return err
		}
		if !expr.matches(value, exists) {
			return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("请求参数条件不匹配 %q", text), nil)
		}
	}
	return nil
}

func matchHeaderExpressions(req *servlet.Request, expressions []string) error {
	for _, text := range expressions {
		expr, ok := parseConditionExpression(text)
		if !ok {
			continue
		}
		value, exists := req.HeaderValue(expr.name)
		if !expr.matches(value, exists) {
			return servlet.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("请求头条件不匹配 %q", text), nil)
		}
	}
	return nil
}

type conditionExpression struct {
	name        string
	value       string
	hasValue    bool
	negated     bool
	valueNegate bool
}

func parseConditionExpression(text string) (conditionExpression, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return conditionExpression{}, false
	}
	if name, value, ok := strings.Cut(text, "!="); ok {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		return conditionExpression{name: name, value: value, hasValue: true, valueNegate: true}, name != ""
	}
	if name, value, ok := strings.Cut(text, "="); ok {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		return conditionExpression{name: name, value: value, hasValue: true}, name != ""
	}
	if strings.HasPrefix(text, "!") {
		name := strings.TrimSpace(strings.TrimPrefix(text, "!"))
		return conditionExpression{name: name, negated: true}, name != ""
	}
	return conditionExpression{name: text}, true
}

func (e conditionExpression) matches(actual string, exists bool) bool {
	if e.negated {
		return !exists
	}
	if !exists {
		return false
	}
	if !e.hasValue {
		return true
	}
	matched := actual == e.value
	if e.valueNegate {
		return !matched
	}
	return matched
}

func mediaTypeMatches(pattern string, actual string) bool {
	mediaRange, ok := servlet.NewMediaType(pattern)
	if !ok {
		return false
	}
	return mediaRange.Matches(actual)
}
