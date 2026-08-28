package webtest

import (
	"net/http"
	"net/http/httptest"

	arkjson "goark.dev/arkarta/json"
)

// Response 封装一次 WebTest 响应。
type Response struct {
	recorder *httptest.ResponseRecorder
	codec    arkjson.Codec
	err      error
}

func newResponse(recorder *httptest.ResponseRecorder, codec arkjson.Codec, err error) *Response {
	if codec == nil {
		codec = arkjson.DefaultCodec()
	}
	return &Response{
		recorder: recorder,
		codec:    codec,
		err:      err,
	}
}

// Err 返回请求执行前的构造错误。
func (r *Response) Err() error {
	if r == nil {
		return ErrNilResponse
	}
	return r.err
}

// Recorder 返回底层 httptest 响应记录器。
func (r *Response) Recorder() *httptest.ResponseRecorder {
	if r == nil {
		return nil
	}
	return r.recorder
}

// Result 返回标准库响应视图。
func (r *Response) Result() *http.Response {
	if r == nil || r.recorder == nil {
		return nil
	}
	return r.recorder.Result()
}

// StatusCode 返回 HTTP 状态码。
func (r *Response) StatusCode() int {
	if r == nil || r.recorder == nil {
		return 0
	}
	return r.recorder.Code
}

// Header 返回响应头。
func (r *Response) Header() http.Header {
	if r == nil || r.recorder == nil {
		return make(http.Header)
	}
	return r.recorder.Header()
}

// BodyBytes 返回响应体副本。
func (r *Response) BodyBytes() []byte {
	if r == nil || r.recorder == nil || r.recorder.Body == nil {
		return nil
	}
	return append([]byte(nil), r.recorder.Body.Bytes()...)
}

// BodyString 返回响应体字符串。
func (r *Response) BodyString() string {
	if r == nil {
		return ""
	}
	return string(r.BodyBytes())
}

// DecodeJSON 将响应体按当前 WebTest JSON 编解码器解码。
func (r *Response) DecodeJSON(target any) error {
	if r == nil {
		return ErrNilResponse
	}
	if target == nil {
		return ErrNilJSONTarget
	}
	if r.err != nil {
		return r.err
	}
	return arkjson.Unmarshal(r.codec, r.BodyBytes(), target)
}
