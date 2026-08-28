package stream

import "errors"

var (
	// ErrNilWriterFunc 表示流式响应缺少写入函数。
	ErrNilWriterFunc = errors.New("goark/web/stream: writer func is nil")
	// ErrNilWriter 表示流式写入器不可用。
	ErrNilWriter = errors.New("goark/web/stream: writer is nil")
)
