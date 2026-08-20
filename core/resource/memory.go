package resource

import (
	"bytes"
	"context"
	"io"
	"time"

	arkerrors "github.com/goark-projects/goark/errors"
)

// MemoryResource 表示内存中的不可变资源。
type MemoryResource struct {
	name    string
	data    []byte
	modTime time.Time
}

// NewMemoryResource 创建内存资源，并复制输入数据。
func NewMemoryResource(name string, data []byte) (*MemoryResource, error) {
	if name == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "memory resource name is empty")
	}
	copied := append([]byte(nil), data...)
	return &MemoryResource{
		name:    name,
		data:    copied,
		modTime: time.Now(),
	}, nil
}

func (r *MemoryResource) Location() string {
	if r == nil {
		return ""
	}
	return "memory:" + r.name
}

func (r *MemoryResource) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}

func (r *MemoryResource) Exists(ctx context.Context) (bool, error) {
	if err := checkContext(ctx, "memory resource exists"); err != nil {
		return false, err
	}
	return true, nil
}

func (r *MemoryResource) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := checkContext(ctx, "memory resource open"); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(r.data)), nil
}

func (r *MemoryResource) ReadAll(ctx context.Context) ([]byte, error) {
	if err := checkContext(ctx, "memory resource read"); err != nil {
		return nil, err
	}
	return append([]byte(nil), r.data...), nil
}

func (r *MemoryResource) Stat(ctx context.Context) (Info, error) {
	if err := checkContext(ctx, "memory resource stat"); err != nil {
		return Info{}, err
	}
	return Info{
		Name:    r.name,
		Size:    int64(len(r.data)),
		ModTime: r.modTime,
	}, nil
}
