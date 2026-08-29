package webtest

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
)

// ExpectNoError 断言请求执行没有前置错误。
func (r *Response) ExpectNoError(t testing.TB) *Response {
	t.Helper()
	if err := r.Err(); err != nil {
		t.Fatalf("webtest response error: %v", err)
	}
	return r
}

// ExpectStatus 断言 HTTP 状态码。
func (r *Response) ExpectStatus(t testing.TB, statusCode int) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.StatusCode(); got != statusCode {
		t.Fatalf("status = %d, want %d, body = %q", got, statusCode, r.BodyString())
	}
	return r
}

// ExpectStatus1xx 断言 HTTP 状态码属于 1xx 信息响应。
func (r *Response) ExpectStatus1xx(t testing.TB) *Response {
	return r.expectStatusRange(t, 100, 200, "1xx")
}

// ExpectStatus2xx 断言 HTTP 状态码属于 2xx 成功响应。
func (r *Response) ExpectStatus2xx(t testing.TB) *Response {
	return r.expectStatusRange(t, 200, 300, "2xx")
}

// ExpectStatus3xx 断言 HTTP 状态码属于 3xx 重定向响应。
func (r *Response) ExpectStatus3xx(t testing.TB) *Response {
	return r.expectStatusRange(t, 300, 400, "3xx")
}

// ExpectStatus4xx 断言 HTTP 状态码属于 4xx 客户端错误响应。
func (r *Response) ExpectStatus4xx(t testing.TB) *Response {
	return r.expectStatusRange(t, 400, 500, "4xx")
}

// ExpectStatus5xx 断言 HTTP 状态码属于 5xx 服务端错误响应。
func (r *Response) ExpectStatus5xx(t testing.TB) *Response {
	return r.expectStatusRange(t, 500, 600, "5xx")
}

// ExpectHeader 断言响应头第一个值。
func (r *Response) ExpectHeader(t testing.TB, name string, value string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.Header().Get(name); got != value {
		t.Fatalf("%s header = %q, want %q", name, got, value)
	}
	return r
}

// ExpectHeaderContains 断言响应头包含指定片段。
func (r *Response) ExpectHeaderContains(t testing.TB, name string, fragment string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.Header().Get(name); !strings.Contains(got, fragment) {
		t.Fatalf("%s header = %q, want fragment %q", name, got, fragment)
	}
	return r
}

// ExpectHeaderValues 断言响应头完整值列表。
func (r *Response) ExpectHeaderValues(t testing.TB, name string, values ...string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	got := r.Header().Values(name)
	if !reflect.DeepEqual(got, values) {
		t.Fatalf("%s header values = %#v, want %#v", name, got, values)
	}
	return r
}

// ExpectHeaderExists 断言响应头存在。
func (r *Response) ExpectHeaderExists(t testing.TB, name string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.Header().Values(name); len(got) == 0 {
		t.Fatalf("%s header missing", name)
	}
	return r
}

// ExpectHeaderAbsent 断言响应头不存在。
func (r *Response) ExpectHeaderAbsent(t testing.TB, name string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.Header().Values(name); len(got) > 0 {
		t.Fatalf("%s header exists: %#v", name, got)
	}
	return r
}

// ExpectCookie 断言响应 Cookie 存在且值匹配。
func (r *Response) ExpectCookie(t testing.TB, name string, value string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	cookie, ok := r.Cookie(name)
	if !ok {
		t.Fatalf("cookie %q missing", name)
	}
	if cookie.Value != value {
		t.Fatalf("cookie %q = %q, want %q", name, cookie.Value, value)
	}
	return r
}

// ExpectNoCookie 断言响应 Cookie 不存在。
func (r *Response) ExpectNoCookie(t testing.TB, name string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if cookie, ok := r.Cookie(name); ok {
		t.Fatalf("cookie %q exists: %#v", name, cookie)
	}
	return r
}

// ExpectCookieHTTPOnly 断言响应 Cookie 的 HttpOnly 标记。
func (r *Response) ExpectCookieHTTPOnly(t testing.TB, name string, httpOnly bool) *Response {
	t.Helper()
	cookie := r.expectCookieNamed(t, name)
	if cookie.HttpOnly != httpOnly {
		t.Fatalf("cookie %q HttpOnly = %v, want %v", name, cookie.HttpOnly, httpOnly)
	}
	return r
}

// ExpectCookieSecure 断言响应 Cookie 的 Secure 标记。
func (r *Response) ExpectCookieSecure(t testing.TB, name string, secure bool) *Response {
	t.Helper()
	cookie := r.expectCookieNamed(t, name)
	if cookie.Secure != secure {
		t.Fatalf("cookie %q Secure = %v, want %v", name, cookie.Secure, secure)
	}
	return r
}

// ExpectBody 断言响应体完全匹配。
func (r *Response) ExpectBody(t testing.TB, value string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.BodyString(); got != value {
		t.Fatalf("body = %q, want %q", got, value)
	}
	return r
}

// ExpectBodyContains 断言响应体包含指定片段。
func (r *Response) ExpectBodyContains(t testing.TB, fragment string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.BodyString(); !strings.Contains(got, fragment) {
		t.Fatalf("body = %q, want fragment %q", got, fragment)
	}
	return r
}

// ExpectJSON 断言响应体与期望值 JSON 语义等价。
func (r *Response) ExpectJSON(t testing.TB, expected any) *Response {
	t.Helper()
	r.ExpectNoError(t)
	var actualValue any
	if err := arkjson.Unmarshal(r.codec, r.BodyBytes(), &actualValue); err != nil {
		t.Fatalf("response json invalid: %v, body = %q", err, r.BodyString())
	}
	expectedBytes, err := arkjson.Marshal(r.codec, expected)
	if err != nil {
		t.Fatalf("expected json encode failed: %v", err)
	}
	var expectedValue any
	if err := arkjson.Unmarshal(r.codec, expectedBytes, &expectedValue); err != nil {
		t.Fatalf("expected json invalid: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("json = %#v, want %#v", actualValue, expectedValue)
	}
	return r
}

// ExpectJSONPath 断言 JSON 路径对应值与期望值语义等价。
func (r *Response) ExpectJSONPath(t testing.TB, expression string, expected any) *Response {
	t.Helper()
	r.ExpectNoError(t)
	actual, ok, err := resolveJSONPath(r.codec, r.BodyBytes(), expression)
	if err != nil {
		t.Fatalf("json path %q invalid: %v", expression, err)
	}
	if !ok {
		t.Fatalf("json path %q missing", expression)
	}
	normalized, err := normalizeJSONValue(r.codec, expected)
	if err != nil {
		t.Fatalf("expected json path value encode failed: %v", err)
	}
	if !reflect.DeepEqual(actual, normalized) {
		t.Fatalf("json path %q = %#v, want %#v", expression, actual, normalized)
	}
	return r
}

// ExpectJSONPathExists 断言 JSON 路径存在。
func (r *Response) ExpectJSONPathExists(t testing.TB, expression string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	_, ok, err := resolveJSONPath(r.codec, r.BodyBytes(), expression)
	if err != nil {
		t.Fatalf("json path %q invalid: %v", expression, err)
	}
	if !ok {
		t.Fatalf("json path %q missing", expression)
	}
	return r
}

// ExpectJSONPathAbsent 断言 JSON 路径不存在。
func (r *Response) ExpectJSONPathAbsent(t testing.TB, expression string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	_, ok, err := resolveJSONPath(r.codec, r.BodyBytes(), expression)
	if err != nil {
		t.Fatalf("json path %q invalid: %v", expression, err)
	}
	if ok {
		t.Fatalf("json path %q exists", expression)
	}
	return r
}

// DecodeJSON 在测试中解码响应 JSON。
func DecodeJSON[T any](t testing.TB, response *Response) T {
	t.Helper()
	var target T
	if response == nil {
		t.Fatal(ErrNilResponse)
	}
	if err := response.DecodeJSON(&target); err != nil {
		t.Fatalf("decode json failed: %v, body = %q", err, response.BodyString())
	}
	return target
}

func (r *Response) expectStatusRange(t testing.TB, lower int, upper int, label string) *Response {
	t.Helper()
	r.ExpectNoError(t)
	if got := r.StatusCode(); got < lower || got >= upper {
		t.Fatalf("status = %d, want %s, body = %q", got, label, r.BodyString())
	}
	return r
}

func (r *Response) expectCookieNamed(t testing.TB, name string) *http.Cookie {
	t.Helper()
	r.ExpectNoError(t)
	cookie, ok := r.Cookie(name)
	if !ok {
		t.Fatalf("cookie %q missing", name)
	}
	return cookie
}
