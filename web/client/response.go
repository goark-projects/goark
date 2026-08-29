package client

import (
	"io"
	"net/http"

	arkjson "goark.dev/arkarta/json"
)

// Response 是已读取响应体快照的 HTTP 响应。
type Response struct {
	statusCode int
	status     string
	header     http.Header
	body       []byte
	codec      arkjson.Codec
}

func newResponse(raw *http.Response, body []byte, codec arkjson.Codec) *Response {
	if codec == nil {
		codec = arkjson.DefaultCodec()
	}
	return &Response{
		statusCode: raw.StatusCode,
		status:     raw.Status,
		header:     raw.Header.Clone(),
		body:       append([]byte(nil), body...),
		codec:      codec,
	}
}

// StatusCode 返回 HTTP 状态码。
func (r *Response) StatusCode() int {
	if r == nil {
		return 0
	}
	return r.statusCode
}

// Status 返回 HTTP 状态文本。
func (r *Response) Status() string {
	if r == nil {
		return ""
	}
	return r.status
}

// Header 返回响应头副本。
func (r *Response) Header() http.Header {
	if r == nil {
		return make(http.Header)
	}
	return r.header.Clone()
}

// HeaderValue 返回响应头第一个值。
func (r *Response) HeaderValue(name string) string {
	if r == nil {
		return ""
	}
	return r.header.Get(name)
}

// HeaderValues 返回响应头全部值副本。
func (r *Response) HeaderValues(name string) []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.header.Values(name)...)
}

// Cookies 返回响应 Set-Cookie 头解析结果。
func (r *Response) Cookies() []*http.Cookie {
	if r == nil {
		return nil
	}
	response := http.Response{Header: r.header.Clone()}
	return response.Cookies()
}

// Cookie 返回指定名称的响应 Cookie。
func (r *Response) Cookie(name string) (*http.Cookie, bool) {
	for _, cookie := range r.Cookies() {
		if cookie != nil && cookie.Name == name {
			return cookie, true
		}
	}
	return nil, false
}

// BodyBytes 返回响应体副本。
func (r *Response) BodyBytes() []byte {
	if r == nil {
		return nil
	}
	return append([]byte(nil), r.body...)
}

// BodyString 返回响应体字符串。
func (r *Response) BodyString() string {
	if r == nil {
		return ""
	}
	return string(r.body)
}

// DecodeJSON 将响应体按当前 JSON 编解码器解码。
func (r *Response) DecodeJSON(target any) error {
	if target == nil {
		return ErrNilJSONTarget
	}
	return arkjson.Unmarshal(r.codec, r.body, target)
}

// IsSuccess 判断响应状态码是否为 2xx。
func (r *Response) IsSuccess() bool {
	if r == nil {
		return false
	}
	return r.statusCode >= http.StatusOK && r.statusCode < http.StatusMultipleChoices
}

// EnsureSuccess 将非 2xx 响应转换为 StatusError。
func (r *Response) EnsureSuccess() error {
	if r == nil || r.IsSuccess() {
		return nil
	}
	return &StatusError{
		StatusCode: r.statusCode,
		Status:     r.status,
		Body:       r.BodyBytes(),
	}
}

func readResponseBody(body io.Reader, maxBytes int64) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if maxBytes < 0 {
		return io.ReadAll(body)
	}
	reader := &io.LimitedReader{R: body, N: maxBytes + 1}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}
