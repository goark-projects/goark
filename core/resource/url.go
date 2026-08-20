package resource

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	arkerrors "github.com/goark-projects/goark/errors"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// URLResource 表示 HTTP/HTTPS 资源。
type URLResource struct {
	url    *url.URL
	client *http.Client
}

// NewURLResource 创建 URL 资源。
func NewURLResource(rawURL string, client *http.Client) (*URLResource, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid resource url %q", rawURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "unsupported resource url scheme %q", parsed.Scheme)
	}
	if client == nil {
		client = defaultHTTPClient
	}
	return &URLResource{url: parsed, client: client}, nil
}

func (r *URLResource) Location() string {
	if r == nil || r.url == nil {
		return ""
	}
	return r.url.String()
}

func (r *URLResource) Name() string {
	if r == nil || r.url == nil {
		return ""
	}
	name := path.Base(r.url.Path)
	if name == "." || name == "/" {
		return r.url.Host
	}
	return name
}

func (r *URLResource) Exists(ctx context.Context) (bool, error) {
	if err := checkContext(ctx, "url resource exists"); err != nil {
		return false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, r.url.String(), nil)
	if err != nil {
		return false, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to create HEAD request for %q", r.url)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return false, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to check url resource %q", r.url)
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 400, nil
}

func (r *URLResource) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := checkContext(ctx, "url resource open"); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url.String(), nil)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to create GET request for %q", r.url)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to open url resource %q", r.url)
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		response.Body.Close()
		return nil, arkerrors.Newf(arkerrors.CodeResource, "url resource %q returned status %d", r.url, response.StatusCode)
	}
	return response.Body, nil
}

func (r *URLResource) ReadAll(ctx context.Context) ([]byte, error) {
	return readAll(ctx, r)
}

func (r *URLResource) Stat(ctx context.Context) (Info, error) {
	if err := checkContext(ctx, "url resource stat"); err != nil {
		return Info{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, r.url.String(), nil)
	if err != nil {
		return Info{}, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to create HEAD request for %q", r.url)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return Info{}, arkerrors.Wrapf(arkerrors.CodeResource, err, "failed to stat url resource %q", r.url)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return Info{}, arkerrors.Newf(arkerrors.CodeResource, "url resource %q returned status %d", r.url, response.StatusCode)
	}
	info := Info{Name: r.Name(), Size: response.ContentLength}
	if lastModified := response.Header.Get("Last-Modified"); lastModified != "" {
		if parsed, err := http.ParseTime(lastModified); err == nil {
			info.ModTime = parsed
		}
	}
	if length := response.Header.Get("Content-Length"); info.Size < 0 && length != "" {
		if parsed, err := strconv.ParseInt(length, 10, 64); err == nil {
			info.Size = parsed
		}
	}
	return info, nil
}
