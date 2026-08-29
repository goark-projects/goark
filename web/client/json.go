package client

import (
	"context"
	"net/http"
)

// JSONResponse 保存已解码的强类型 JSON 响应。
type JSONResponse[T any] struct {
	Response *Response
	Body     T
}

// RetrieveJSON 执行请求并将响应体解码为强类型 JSON。
func RetrieveJSON[T any](client *Client, ctx context.Context, method string, target string, options ...RequestOption) (JSONResponse[T], error) {
	var zero JSONResponse[T]
	if client == nil {
		return zero, ErrNilHTTPClient
	}
	response, err := client.Retrieve(ctx, method, target, options...)
	if err != nil {
		zero.Response = response
		return zero, err
	}
	var body T
	if err := response.DecodeJSON(&body); err != nil {
		return JSONResponse[T]{Response: response}, err
	}
	return JSONResponse[T]{
		Response: response,
		Body:     body,
	}, nil
}

// GetJSON 执行 GET 请求并将响应体解码为强类型 JSON。
func GetJSON[T any](client *Client, ctx context.Context, target string, options ...RequestOption) (JSONResponse[T], error) {
	return RetrieveJSON[T](client, ctx, http.MethodGet, target, options...)
}

// PostJSON 执行 POST 请求并将响应体解码为强类型 JSON。
func PostJSON[T any](client *Client, ctx context.Context, target string, options ...RequestOption) (JSONResponse[T], error) {
	return RetrieveJSON[T](client, ctx, http.MethodPost, target, options...)
}

// PutJSON 执行 PUT 请求并将响应体解码为强类型 JSON。
func PutJSON[T any](client *Client, ctx context.Context, target string, options ...RequestOption) (JSONResponse[T], error) {
	return RetrieveJSON[T](client, ctx, http.MethodPut, target, options...)
}

// PatchJSON 执行 PATCH 请求并将响应体解码为强类型 JSON。
func PatchJSON[T any](client *Client, ctx context.Context, target string, options ...RequestOption) (JSONResponse[T], error) {
	return RetrieveJSON[T](client, ctx, http.MethodPatch, target, options...)
}

// DeleteJSON 执行 DELETE 请求并将响应体解码为强类型 JSON。
func DeleteJSON[T any](client *Client, ctx context.Context, target string, options ...RequestOption) (JSONResponse[T], error) {
	return RetrieveJSON[T](client, ctx, http.MethodDelete, target, options...)
}
