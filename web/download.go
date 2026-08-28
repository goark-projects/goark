package web

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

const defaultDownloadContentType = "application/octet-stream"

// DownloadOption 定制文件下载结果。
type DownloadOption func(*downloadOptions)

type downloadOptions struct {
	statusCode       int
	contentType      string
	contentLength    int64
	hasContentLength bool
	filename         string
	disposition      string
	headers          http.Header
}

// DownloadResult 表示可流式写出的下载响应。
type DownloadResult struct {
	reader  io.Reader
	options downloadOptions
}

// Download 创建流式下载响应。
func Download(reader io.Reader, options ...DownloadOption) DownloadResult {
	return DownloadResult{
		reader:  reader,
		options: newDownloadOptions(options),
	}
}

// Attachment 创建 attachment 下载响应。
func Attachment(filename string, reader io.Reader, options ...DownloadOption) DownloadResult {
	all := make([]DownloadOption, 0, len(options)+2)
	all = append(all, WithDownloadDisposition("attachment"), WithDownloadFilename(filename))
	all = append(all, options...)
	return Download(reader, all...)
}

// WithDownloadStatus 设置下载 HTTP 状态码。
func WithDownloadStatus(statusCode int) DownloadOption {
	return func(options *downloadOptions) {
		options.statusCode = statusCode
	}
}

// WithDownloadContentType 设置下载 Content-Type。
func WithDownloadContentType(contentType string) DownloadOption {
	return func(options *downloadOptions) {
		if contentType = strings.TrimSpace(contentType); contentType != "" {
			options.contentType = contentType
		}
	}
}

// WithDownloadContentLength 设置下载 Content-Length。
func WithDownloadContentLength(length int64) DownloadOption {
	return func(options *downloadOptions) {
		if length >= 0 {
			options.contentLength = length
			options.hasContentLength = true
		}
	}
}

// WithDownloadFilename 设置 Content-Disposition 的文件名参数。
func WithDownloadFilename(filename string) DownloadOption {
	return func(options *downloadOptions) {
		options.filename = sanitizeDownloadFilename(filename)
	}
}

// WithDownloadDisposition 设置 Content-Disposition 类型。
func WithDownloadDisposition(disposition string) DownloadOption {
	return func(options *downloadOptions) {
		if disposition = strings.TrimSpace(disposition); disposition != "" {
			options.disposition = disposition
		}
	}
}

// WithDownloadHeader 设置下载响应头。
func WithDownloadHeader(name string, value string) DownloadOption {
	return func(options *downloadOptions) {
		name = http.CanonicalHeaderKey(strings.TrimSpace(name))
		if name == "" {
			return
		}
		if options.headers == nil {
			options.headers = make(http.Header, 1)
		}
		options.headers.Set(name, value)
	}
}

// Write 将下载响应写入 Arkarta Web 上下文。
func (r DownloadResult) Write(ctx *arkweb.Context) (err error) {
	if ctx == nil || ctx.Response() == nil {
		return arkweb.ErrNilContext
	}
	if r.reader == nil {
		return ErrNilDownloadReader
	}
	if closer, ok := r.reader.(io.Closer); ok {
		defer func() {
			err = errors.Join(err, closer.Close())
		}()
	}

	response := ctx.Response()
	statusCode := normalizeEntityStatus(r.options.statusCode, http.StatusOK)
	if err := writeDownloadHeaders(response, r.options, statusCode); err != nil {
		return err
	}
	response.SetStatus(statusCode)
	if !entityStatusAllowsBody(statusCode) || isDownloadHeadRequest(ctx) {
		return nil
	}
	_, err = io.Copy(response.BodyWriter(), r.reader)
	return err
}

func newDownloadOptions(options []DownloadOption) downloadOptions {
	out := downloadOptions{contentType: defaultDownloadContentType}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

func writeDownloadHeaders(response servlet.Response, options downloadOptions, statusCode int) error {
	if options.contentType != "" && response.Header().Get("Content-Type") == "" {
		if err := servlet.SetContentType(response, options.contentType); err != nil {
			return err
		}
	}
	if options.hasContentLength && entityStatusAllowsBody(statusCode) {
		if err := servlet.SetContentLength(response, options.contentLength); err != nil {
			return err
		}
	}
	if disposition := downloadContentDisposition(options); disposition != "" {
		response.Header().Set("Content-Disposition", disposition)
	}
	applyEntityHeaders(response.Header(), options.headers)
	return nil
}

func downloadContentDisposition(options downloadOptions) string {
	disposition := strings.TrimSpace(options.disposition)
	if disposition == "" && options.filename != "" {
		disposition = "attachment"
	}
	if disposition == "" {
		return ""
	}
	if options.filename == "" {
		return disposition
	}
	return mime.FormatMediaType(disposition, map[string]string{"filename": options.filename})
}

func sanitizeDownloadFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	filename = path.Base(filename)
	filename = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, filename)
	if filename == "" || filename == "." || filename == ".." {
		return "download"
	}
	return filename
}

func isDownloadHeadRequest(ctx *arkweb.Context) bool {
	return ctx.Request() != nil && strings.EqualFold(ctx.Request().Method(), http.MethodHead)
}
