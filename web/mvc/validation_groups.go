package mvc

import (
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
)

// BindJSONGroups 绑定 JSON 请求体，并按显式校验分组写出 JSON 响应。
func BindJSONGroups[In any, Out any](statusCode int, fn BindFunc[In, Out], groups ...string) arkweb.Handler {
	return bindJSON(statusCode, fn, groups)
}

// BindEntityGroups 绑定 JSON 请求体，并按显式校验分组写出响应实体。
func BindEntityGroups[In any, Out any](fn BindEntityFunc[In, Out], groups ...string) arkweb.Handler {
	return bindEntity(fn, groups)
}

// BindMultipartGroups 绑定 multipart/form-data 请求体，并按显式校验分组写出 JSON 响应。
func BindMultipartGroups[In any, Out any](statusCode int, fn BindFunc[In, Out], groups []string, options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipart(statusCode, fn, groups, options...)
}

// BindMultipartEntityGroups 绑定 multipart/form-data 请求体，并按显式校验分组写出响应实体。
func BindMultipartEntityGroups[In any, Out any](fn BindEntityFunc[In, Out], groups []string, options ...servletmultipart.Option) arkweb.Handler {
	return bindMultipartEntity(fn, groups, options...)
}
