package static

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"

	servletresource "goark.dev/arkarta/servlet/resource"
)

const minContentVersionLength = 8

// ErrInvalidResourcePath 表示静态资源路径非法。
var ErrInvalidResourcePath = errors.New("goark/web/static: invalid resource path")

// ErrInvalidResourceVersion 表示静态资源版本号非法。
var ErrInvalidResourceVersion = errors.New("goark/web/static: invalid resource version")

// ContentVersionPath 生成插入内容哈希的静态资源路径。
func ContentVersionPath(ctx context.Context, root fs.FS, name string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	leadingSlash := strings.HasPrefix(strings.TrimSpace(name), "/")
	clean, err := cleanVersionPath(name)
	if err != nil {
		return "", err
	}
	version, err := contentVersion(ctx, root, clean)
	if err != nil {
		return "", err
	}
	versioned := insertContentVersion(clean, version)
	if leadingSlash {
		return "/" + versioned, nil
	}
	return versioned, nil
}

// FixedVersionPath 生成带固定版本前缀的静态资源路径。
func FixedVersionPath(version string, name string) (string, error) {
	cleanVersion, err := cleanFixedVersion(version)
	if err != nil {
		return "", err
	}
	leadingSlash := strings.HasPrefix(strings.TrimSpace(name), "/")
	clean, err := cleanVersionPath(name)
	if err != nil {
		return "", err
	}
	versioned := cleanVersion + "/" + clean
	if leadingSlash {
		return "/" + versioned, nil
	}
	return versioned, nil
}

type contentVersionFS struct {
	root  fs.FS
	cache sync.Map
}

type fixedVersionFS struct {
	root    fs.FS
	version string
}

type versionCacheEntry struct {
	signature fileSignature
	version   string
}

type fileSignature struct {
	size        int64
	modUnixNano int64
}

func newContentVersionFS(root fs.FS) fs.FS {
	return &contentVersionFS{root: root}
}

func newFixedVersionFS(root fs.FS, version string) fs.FS {
	return fixedVersionFS{root: root, version: version}
}

func (f fixedVersionFS) Open(name string) (fs.File, error) {
	file, err := f.root.Open(name)
	if err == nil {
		return file, nil
	}
	base, ok := f.stripVersion(name)
	if !ok {
		return nil, err
	}
	return f.root.Open(base)
}

func (f fixedVersionFS) stripVersion(name string) (string, bool) {
	clean, err := cleanVersionPath(name)
	if err != nil {
		return "", false
	}
	prefix := f.version + "/"
	if !strings.HasPrefix(clean, prefix) {
		return "", false
	}
	base := strings.TrimPrefix(clean, prefix)
	return base, base != ""
}

func (f *contentVersionFS) Open(name string) (fs.File, error) {
	file, err := f.root.Open(name)
	if err == nil {
		return file, nil
	}
	base, requested, ok := splitContentVersion(name)
	if !ok {
		return nil, err
	}
	version, versionErr := f.version(context.Background(), base)
	if versionErr != nil || !contentVersionMatches(requested, version) {
		return nil, err
	}
	return f.root.Open(base)
}

func (f *contentVersionFS) version(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	file, err := f.root.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	signature := fileSignature{
		size:        info.Size(),
		modUnixNano: info.ModTime().UnixNano(),
	}
	if cached, ok := f.cache.Load(name); ok {
		entry, ok := cached.(versionCacheEntry)
		if ok && entry.signature == signature {
			return entry.version, nil
		}
	}
	version, err := hashFile(ctx, file)
	if err != nil {
		return "", err
	}
	f.cache.Store(name, versionCacheEntry{signature: signature, version: version})
	return version, nil
}

func contentVersion(ctx context.Context, root fs.FS, name string) (string, error) {
	if root == nil {
		return "", servletresource.ErrNilFileSystem
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashFile(ctx, file)
}

func hashFile(ctx context.Context, reader io.Reader) (string, error) {
	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			if _, writeErr := hash.Write(buffer[:n]); writeErr != nil {
				return "", writeErr
			}
		}
		if err == io.EOF {
			return hex.EncodeToString(hash.Sum(nil)), nil
		}
		if err != nil {
			return "", err
		}
	}
}

func insertContentVersion(name string, version string) string {
	dir, file := path.Split(name)
	ext := path.Ext(file)
	stem := strings.TrimSuffix(file, ext)
	return dir + stem + "-" + version + ext
}

func splitContentVersion(name string) (string, string, bool) {
	clean, err := cleanVersionPath(name)
	if err != nil {
		return "", "", false
	}
	dir, file := path.Split(clean)
	ext := path.Ext(file)
	stem := strings.TrimSuffix(file, ext)
	index := strings.LastIndexByte(stem, '-')
	if index <= 0 || index == len(stem)-1 {
		return "", "", false
	}
	version := stem[index+1:]
	if len(version) < minContentVersionLength || !isHex(version) {
		return "", "", false
	}
	return dir + stem[:index] + ext, strings.ToLower(version), true
}

func contentVersionMatches(requested string, actual string) bool {
	if len(requested) < minContentVersionLength || len(requested) > len(actual) {
		return false
	}
	return strings.HasPrefix(actual, strings.ToLower(requested))
}

func cleanFixedVersion(version string) (string, error) {
	version = strings.TrimSpace(strings.ReplaceAll(version, "\\", "/"))
	version = strings.Trim(version, "/")
	if version == "" || version == "." || version == ".." || strings.Contains(version, "/") || !fs.ValidPath(version) {
		return "", ErrInvalidResourceVersion
	}
	return version, nil
}

func cleanVersionPath(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimPrefix(name, "/")
	clean := path.Clean(name)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." || !fs.ValidPath(clean) {
		return "", ErrInvalidResourcePath
	}
	return clean, nil
}

func isHex(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return value != ""
}
