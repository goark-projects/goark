package resource

import (
	"context"
	stderrors "errors"
	"io"
	"os"
	"path/filepath"

	arkerrors "goark.dev/goark/errors"
)

// FileResource 表示本地文件系统资源。
type FileResource struct {
	path string
}

// NewFileResource 创建本地文件资源。
func NewFileResource(path string) (*FileResource, error) {
	if path == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "file resource path is empty")
	}
	return &FileResource{path: filepath.Clean(path)}, nil
}

// Path 返回本地文件路径。
func (r *FileResource) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

func (r *FileResource) Location() string {
	if r == nil {
		return ""
	}
	return "file:" + r.path
}

func (r *FileResource) Name() string {
	if r == nil {
		return ""
	}
	return filepath.Base(r.path)
}

func (r *FileResource) Exists(ctx context.Context) (bool, error) {
	if err := checkContext(ctx, "file resource exists"); err != nil {
		return false, err
	}
	_, err := os.Stat(r.path)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to stat file resource %q", r.path)
}

func (r *FileResource) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := checkContext(ctx, "file resource open"); err != nil {
		return nil, err
	}
	file, err := os.Open(r.path)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to open file resource %q", r.path)
	}
	return file, nil
}

func (r *FileResource) ReadAll(ctx context.Context) ([]byte, error) {
	return readAll(ctx, r)
}

func (r *FileResource) Stat(ctx context.Context) (Info, error) {
	if err := checkContext(ctx, "file resource stat"); err != nil {
		return Info{}, err
	}
	stat, err := os.Stat(r.path)
	if err != nil {
		return Info{}, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to stat file resource %q", r.path)
	}
	return Info{
		Name:    stat.Name(),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
	}, nil
}
