package static

import (
	"context"
	"io/fs"
	"path"
	"strings"

	servletresource "goark.dev/arkarta/servlet/resource"
)

// ResourceURLProvider 生成静态资源对外访问 URL。
type ResourceURLProvider struct {
	root           fs.FS
	pathPrefix     string
	contentVersion bool
	fixedVersion   string
}

// URLProviderOption 定制静态资源 URL 提供器。
type URLProviderOption func(*ResourceURLProvider) error

// NewURLProvider 创建静态资源 URL 提供器。
func NewURLProvider(root fs.FS, options ...URLProviderOption) (ResourceURLProvider, error) {
	provider := ResourceURLProvider{root: root}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&provider); err != nil {
			return ResourceURLProvider{}, err
		}
	}
	if provider.contentVersion && provider.root == nil {
		return ResourceURLProvider{}, servletresource.ErrNilFileSystem
	}
	return provider, nil
}

// WithURLPathPrefix 设置静态资源对外路径前缀。
func WithURLPathPrefix(prefix string) URLProviderOption {
	return func(provider *ResourceURLProvider) error {
		clean, err := cleanURLPathPrefix(prefix)
		if err != nil {
			return err
		}
		provider.pathPrefix = clean
		return nil
	}
}

// WithURLContentVersioning 启用内容哈希版本 URL。
func WithURLContentVersioning() URLProviderOption {
	return func(provider *ResourceURLProvider) error {
		provider.contentVersion = true
		return nil
	}
}

// WithURLFixedVersion 启用固定版本 URL 前缀。
func WithURLFixedVersion(version string) URLProviderOption {
	return func(provider *ResourceURLProvider) error {
		clean, err := cleanFixedVersion(version)
		if err != nil {
			return err
		}
		provider.fixedVersion = clean
		return nil
	}
}

// URL 返回资源的对外访问 URL。
func (p ResourceURLProvider) URL(ctx context.Context, name string) (string, error) {
	leadingSlash := strings.HasPrefix(strings.TrimSpace(name), "/") || p.pathPrefix != ""
	clean, err := cleanVersionPath(name)
	if err != nil {
		return "", err
	}
	versioned := clean
	if p.contentVersion {
		versioned, err = ContentVersionPath(ctx, p.root, versioned)
		if err != nil {
			return "", err
		}
	}
	if p.fixedVersion != "" {
		versioned, err = FixedVersionPath(p.fixedVersion, versioned)
		if err != nil {
			return "", err
		}
	}
	if p.pathPrefix != "" {
		return p.pathPrefix + "/" + strings.TrimPrefix(versioned, "/"), nil
	}
	if leadingSlash {
		return "/" + strings.TrimPrefix(versioned, "/"), nil
	}
	return versioned, nil
}

func cleanURLPathPrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(strings.ReplaceAll(prefix, "\\", "/"))
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" || prefix == "." {
		return "", nil
	}
	if strings.ContainsAny(prefix, "\r\n?#") {
		return "", ErrInvalidResourcePath
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	clean := path.Clean(prefix)
	if clean == "/" || clean == "." || clean == ".." || strings.HasPrefix(clean, "/..") {
		return "", ErrInvalidResourcePath
	}
	return clean, nil
}
