package web

import "net/http"

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
