package resource

import (
	"context"
	stderrors "errors"
	"io"
	"io/fs"
	"path"

	arkerrors "github.com/goark-projects/goark/errors"
)

// FSResource 表示 fs.FS 中的资源，适合 embed 场景。
type FSResource struct {
	fs   fs.FS
	path string
}

// NewFSResource 创建 fs.FS 资源。
func NewFSResource(filesystem fs.FS, resourcePath string) (*FSResource, error) {
	if filesystem == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "filesystem is nil")
	}
	if resourcePath == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "filesystem resource path is empty")
	}
	return &FSResource{
		fs:   filesystem,
		path: path.Clean(resourcePath),
	}, nil
}

func (r *FSResource) Location() string {
	if r == nil {
		return ""
	}
	return "fs:" + r.path
}

func (r *FSResource) Name() string {
	if r == nil {
		return ""
	}
	return path.Base(r.path)
}

func (r *FSResource) Exists(ctx context.Context) (bool, error) {
	if err := checkContext(ctx, "filesystem resource exists"); err != nil {
		return false, err
	}
	_, err := fs.Stat(r.fs, r.path)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to stat filesystem resource %q", r.path)
}

func (r *FSResource) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := checkContext(ctx, "filesystem resource open"); err != nil {
		return nil, err
	}
	file, err := r.fs.Open(r.path)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to open filesystem resource %q", r.path)
	}
	return file, nil
}

func (r *FSResource) ReadAll(ctx context.Context) ([]byte, error) {
	return readAll(ctx, r)
}

func (r *FSResource) Stat(ctx context.Context) (Info, error) {
	if err := checkContext(ctx, "filesystem resource stat"); err != nil {
		return Info{}, err
	}
	stat, err := fs.Stat(r.fs, r.path)
	if err != nil {
		return Info{}, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to stat filesystem resource %q", r.path)
	}
	return Info{
		Name:    stat.Name(),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
	}, nil
}
