package resource

import (
	"context"
	"io"
	"time"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Info 描述资源元数据。
type Info struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// Resource 表示可定位、可读取的资源。
type Resource interface {
	Location() string
	Name() string
	Exists(ctx context.Context) (bool, error)
	Open(ctx context.Context) (io.ReadCloser, error)
	ReadAll(ctx context.Context) ([]byte, error)
	Stat(ctx context.Context) (Info, error)
}

// Loader 根据位置字符串创建资源。
type Loader interface {
	Load(location string) (Resource, error)
}

func checkContext(ctx context.Context, operation string) error {
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if err := ctx.Err(); err != nil {
		return arkerrors.Wrap(arkerrors.CodeResource, err, operation+" canceled")
	}
	return nil
}

func readAll(ctx context.Context, resource Resource) ([]byte, error) {
	if err := checkContext(ctx, "resource read"); err != nil {
		return nil, err
	}
	reader, err := resource.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to read resource %q", resource.Location())
	}
	return data, nil
}
