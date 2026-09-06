package web

import (
	"net/http"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/uri"
)

// ResponseEntity 表示携带状态码、响应头和可选 JSON 响应体的 Web 结果。
type ResponseEntity[T any] struct {
	statusCode int
	headers    http.Header
	mediaTypes []string
	body       T
	hasBody    bool
}

// OK 创建 200 JSON 实体响应。
func OK[T any](body T) ResponseEntity[T] {
	return Status(http.StatusOK, body)
}

// Status 创建指定状态码的 JSON 实体响应。
func Status[T any](statusCode int, body T) ResponseEntity[T] {
	return ResponseEntity[T]{
		statusCode: statusCode,
		body:       body,
		hasBody:    true,
	}
}

// NoBody 创建无响应体实体响应。
func NoBody(statusCode int) ResponseEntity[struct{}] {
	return ResponseEntity[struct{}]{
		statusCode: statusCode,
	}
}

// StatusCode 返回归一化后的 HTTP 状态码。
func (e ResponseEntity[T]) StatusCode() int {
	return normalizeEntityStatus(e.statusCode, http.StatusOK)
}

// Body 返回响应体和是否显式设置响应体。
func (e ResponseEntity[T]) Body() (T, bool) {
	return e.body, e.hasBody
}

// Headers 返回响应头副本。
func (e ResponseEntity[T]) Headers() http.Header {
	return cloneEntityHeaders(e.headers)
}

// WithHeader 设置单值响应头。
func (e ResponseEntity[T]) WithHeader(name, value string) ResponseEntity[T] {
	name = cleanHeaderName(name)
	value = cleanHeaderValue(value)
	if name == "" || value == "" {
		return e
	}
	headers := cloneEntityHeadersForWrite(e.headers)
	headers.Set(name, value)
	e.headers = headers
	return e
}

// WithAddedHeader 追加响应头值。
func (e ResponseEntity[T]) WithAddedHeader(name, value string) ResponseEntity[T] {
	name = cleanHeaderName(name)
	value = cleanHeaderValue(value)
	if name == "" || value == "" {
		return e
	}
	headers := cloneEntityHeadersForWrite(e.headers)
	headers.Add(name, value)
	e.headers = headers
	return e
}

// WithHeaders 合并响应头，同名响应头以传入值为准。
func (e ResponseEntity[T]) WithHeaders(headers http.Header) ResponseEntity[T] {
	if len(headers) == 0 {
		return e
	}
	dst := cloneEntityHeadersForWrite(e.headers)
	for name, values := range headers {
		name = cleanHeaderName(name)
		values = cleanEntityHeaderValues(values)
		if name == "" || len(values) == 0 {
			continue
		}
		dst[name] = values
	}
	e.headers = dst
	return e
}

// WithContentType 设置实体响应体的首选媒体类型。
func (e ResponseEntity[T]) WithContentType(mediaType string) ResponseEntity[T] {
	e.mediaTypes = []string{mediaType}
	return e
}

// WithMediaTypes 设置实体响应体可写出的媒体类型集合。
func (e ResponseEntity[T]) WithMediaTypes(mediaTypes ...string) ResponseEntity[T] {
	e.mediaTypes = append([]string(nil), mediaTypes...)
	return e
}

// HasMediaTypes 返回实体是否显式设置了响应媒体类型。
func (e ResponseEntity[T]) HasMediaTypes() bool {
	return len(e.mediaTypes) > 0
}

// Write 将实体响应写入 Arkarta Web 上下文。
func (e ResponseEntity[T]) Write(ctx *arkweb.Context) error {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	statusCode := e.StatusCode()
	writeBody := e.hasBody && entityStatusAllowsBody(statusCode)
	if writeBody && len(e.mediaTypes) == 0 {
		if err := ensureEntityAccepted(ctx); err != nil {
			return err
		}
	}
	response := ctx.Response()
	applyEntityHeaders(response.Header(), e.headers)
	response.SetStatus(statusCode)
	if !writeBody {
		return nil
	}
	if len(e.mediaTypes) > 0 {
		return message.WriterFromContext(ctx).Write(ctx, statusCode, e.body, e.mediaTypes...)
	}
	if response.Header().Get("Content-Type") == "" {
		if err := servlet.SetContentType(response, arkjson.ContentType); err != nil {
			return err
		}
	}
	return arkjson.Encode(ctx.JSONCodec(), response.BodyWriter(), e.body)
}

func ensureEntityAccepted(ctx *arkweb.Context) error {
	request := ctx.Request()
	if request == nil {
		return nil
	}
	if _, ok := request.NegotiateContentType(arkjson.ContentType); ok {
		return nil
	}
	return servlet.NewHTTPError(http.StatusNotAcceptable, http.StatusText(http.StatusNotAcceptable), nil)
}

func normalizeEntityStatus(statusCode int, fallback int) int {
	if statusCode == 0 {
		return fallback
	}
	if statusCode < 100 || statusCode > 999 {
		return http.StatusInternalServerError
	}
	return statusCode
}

func entityStatusAllowsBody(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode != http.StatusNoContent && statusCode != http.StatusNotModified
}

func applyEntityHeaders(dst servlet.Header, src http.Header) {
	if len(src) == 0 {
		return
	}
	for name, values := range src {
		if len(values) == 0 {
			continue
		}
		dst.Set(name, values[0])
		for _, value := range values[1:] {
			dst.Add(name, value)
		}
	}
}

func cloneEntityHeadersForWrite(src http.Header) http.Header {
	dst := cloneEntityHeaders(src)
	if dst == nil {
		return make(http.Header, 1)
	}
	return dst
}

func cloneEntityHeaders(src http.Header) http.Header {
	if len(src) == 0 {
		return nil
	}
	dst := make(http.Header, len(src))
	for name, values := range src {
		name = cleanHeaderName(name)
		values = cleanEntityHeaderValues(values)
		if name == "" || len(values) == 0 {
			continue
		}
		dst[name] = values
	}
	return dst
}

func cleanEntityHeaderValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = cleanHeaderValue(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// Created 创建 201 Created JSON 实体响应，并设置 Location。
func Created[T any](location string, body T) ResponseEntity[T] {
	return Status(http.StatusCreated, body).WithLocation(location)
}

// CreatedNoBody 创建 201 Created 无响应体实体响应，并设置 Location。
func CreatedNoBody(location string) ResponseEntity[struct{}] {
	return NoBody(http.StatusCreated).WithLocation(location)
}

// Accepted 创建 202 Accepted JSON 实体响应。
func Accepted[T any](body T) ResponseEntity[T] {
	return Status(http.StatusAccepted, body)
}

// AcceptedNoBody 创建 202 Accepted 无响应体实体响应。
func AcceptedNoBody() ResponseEntity[struct{}] {
	return NoBody(http.StatusAccepted)
}

// NoContent 创建 204 No Content 实体响应。
func NoContent() ResponseEntity[struct{}] {
	return NoBody(http.StatusNoContent)
}

// BadRequest 创建 400 Bad Request JSON 实体响应。
func BadRequest[T any](body T) ResponseEntity[T] {
	return Status(http.StatusBadRequest, body)
}

// NotFound 创建 404 Not Found 无响应体实体响应。
func NotFound() ResponseEntity[struct{}] {
	return NoBody(http.StatusNotFound)
}

// CreatedFromCurrentRequest 基于当前请求 URI 追加路径模板并创建 201 Created 响应。
func CreatedFromCurrentRequest[T any](ctx *arkweb.Context, path string, variables map[string]string, body T) (ResponseEntity[T], error) {
	location, err := currentRequestLocation(ctx, path, variables)
	if err != nil {
		return ResponseEntity[T]{}, err
	}
	return Created(location, body), nil
}

// CreatedNoBodyFromCurrentRequest 基于当前请求 URI 追加路径模板并创建无响应体 201 Created 响应。
func CreatedNoBodyFromCurrentRequest(ctx *arkweb.Context, path string, variables map[string]string) (ResponseEntity[struct{}], error) {
	location, err := currentRequestLocation(ctx, path, variables)
	if err != nil {
		return ResponseEntity[struct{}]{}, err
	}
	return CreatedNoBody(location), nil
}

func currentRequestLocation(ctx *arkweb.Context, path string, variables map[string]string) (string, error) {
	if ctx == nil || ctx.Request() == nil {
		return "", arkweb.ErrNilContext
	}
	if variables == nil {
		variables = map[string]string{}
	}
	return uri.FromCurrentRequestURI(ctx).Path(path).BuildAndExpand(variables)
}
