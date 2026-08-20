package resource

import (
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	arkerrors "github.com/goark-projects/goark/errors"
)

// ProtocolResolver 支持自定义资源协议。
type ProtocolResolver interface {
	Resolve(location string) (Resource, bool, error)
}

// DefaultLoader 是 Goark 默认资源加载器。
type DefaultLoader struct {
	fileBase  string
	files     map[string]fs.FS
	memory    map[string]*MemoryResource
	resolvers []ProtocolResolver
	client    *http.Client
}

// LoaderOption 调整默认资源加载器。
type LoaderOption func(*DefaultLoader) error

// NewLoader 创建默认资源加载器。
func NewLoader(options ...LoaderOption) (*DefaultLoader, error) {
	loader := &DefaultLoader{
		files:  make(map[string]fs.FS),
		memory: make(map[string]*MemoryResource),
		client: defaultHTTPClient,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(loader); err != nil {
			return nil, err
		}
	}
	return loader, nil
}

// WithFileBase 设置无协议文件路径的基础目录。
func WithFileBase(base string) LoaderOption {
	return func(loader *DefaultLoader) error {
		loader.fileBase = base
		return nil
	}
}

// WithFS 注册命名 fs.FS。
func WithFS(name string, filesystem fs.FS) LoaderOption {
	return func(loader *DefaultLoader) error {
		if strings.TrimSpace(name) == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "filesystem name is empty")
		}
		if filesystem == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "filesystem is nil")
		}
		loader.files[name] = filesystem
		return nil
	}
}

// WithMemory 注册内存资源。
func WithMemory(name string, data []byte) LoaderOption {
	return func(loader *DefaultLoader) error {
		resource, err := NewMemoryResource(name, data)
		if err != nil {
			return err
		}
		loader.memory[name] = resource
		return nil
	}
}

// WithProtocolResolver 注册自定义协议解析器。
func WithProtocolResolver(resolver ProtocolResolver) LoaderOption {
	return func(loader *DefaultLoader) error {
		if resolver == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "protocol resolver is nil")
		}
		loader.resolvers = append(loader.resolvers, resolver)
		return nil
	}
}

// WithHTTPClient 设置 URLResource 使用的 HTTP 客户端。
func WithHTTPClient(client *http.Client) LoaderOption {
	return func(loader *DefaultLoader) error {
		if client == nil {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "http client is nil")
		}
		loader.client = client
		return nil
	}
}

// Load 根据 location 创建资源。支持 file:, fs:, memory:, http:, https: 与无协议文件路径。
func (l *DefaultLoader) Load(location string) (Resource, error) {
	if l == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "resource loader is nil")
	}
	location = strings.TrimSpace(location)
	if location == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "resource location is empty")
	}
	for _, resolver := range l.resolvers {
		resource, ok, err := resolver.Resolve(location)
		if err != nil || ok {
			return resource, err
		}
	}

	switch {
	case strings.HasPrefix(location, "file:"):
		return NewFileResource(filePathFromLocation(location))
	case strings.HasPrefix(location, "fs:"):
		return l.loadFS(strings.TrimPrefix(location, "fs:"))
	case strings.HasPrefix(location, "memory:"):
		return l.loadMemory(strings.TrimPrefix(location, "memory:"))
	case strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://"):
		return NewURLResource(location, l.client)
	default:
		if scheme := locationScheme(location); scheme != "" {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "unsupported resource scheme %q", scheme)
		}
		return NewFileResource(l.resolveFilePath(location))
	}
}

func (l *DefaultLoader) loadFS(location string) (Resource, error) {
	name, resourcePath := splitFirstPath(location)
	filesystem := l.files[name]
	if filesystem == nil {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "filesystem %q not found", name)
	}
	return NewFSResource(filesystem, resourcePath)
}

func (l *DefaultLoader) loadMemory(name string) (Resource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "memory resource name is empty")
	}
	resource := l.memory[name]
	if resource == nil {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "memory resource %q not found", name)
	}
	return resource, nil
}

func (l *DefaultLoader) resolveFilePath(location string) string {
	if l.fileBase == "" || filepath.IsAbs(location) {
		return location
	}
	return filepath.Join(l.fileBase, location)
}

func splitFirstPath(location string) (string, string) {
	location = strings.Trim(strings.TrimSpace(location), "/")
	index := strings.Index(location, "/")
	if index < 0 {
		return location, "."
	}
	return location[:index], path.Clean(location[index+1:])
}

func filePathFromLocation(location string) string {
	raw := strings.TrimPrefix(location, "file:")
	if strings.HasPrefix(raw, "//") {
		if parsed, err := url.Parse(location); err == nil {
			raw = parsed.Path
		}
	}
	raw = filepath.FromSlash(raw)
	if len(raw) >= 3 && raw[0] == filepath.Separator && raw[2] == ':' {
		raw = raw[1:]
	}
	return raw
}

func locationScheme(location string) string {
	index := strings.Index(location, ":")
	if index <= 1 {
		return ""
	}
	scheme := location[:index]
	for _, r := range scheme {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.') {
			return ""
		}
	}
	return strings.ToLower(scheme)
}
