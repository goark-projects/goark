package web

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

// CacheControl 表示可写入 Cache-Control 响应头的缓存指令集合。
type CacheControl struct {
	directives []string
}

// CacheControlValue 使用原始响应头值创建缓存指令。
func CacheControlValue(value string) CacheControl {
	value = cleanHeaderValue(value)
	if value == "" {
		return CacheControl{}
	}
	return CacheControl{directives: []string{value}}
}

// NoCache 创建 no-cache 缓存指令。
func NoCache() CacheControl {
	return CacheControl{}.NoCache()
}

// NoStore 创建 no-store 缓存指令。
func NoStore() CacheControl {
	return CacheControl{}.NoStore()
}

// MaxAge 创建 max-age 缓存指令。
func MaxAge(age time.Duration) CacheControl {
	return CacheControl{}.MaxAge(age)
}

// NoCache 追加 no-cache 指令。
func (c CacheControl) NoCache() CacheControl {
	return c.appendDirective("no-cache")
}

// NoStore 追加 no-store 指令。
func (c CacheControl) NoStore() CacheControl {
	return c.appendDirective("no-store")
}

// Public 追加 public 指令。
func (c CacheControl) Public() CacheControl {
	return c.appendDirective("public")
}

// Private 追加 private 指令。
func (c CacheControl) Private() CacheControl {
	return c.appendDirective("private")
}

// NoTransform 追加 no-transform 指令。
func (c CacheControl) NoTransform() CacheControl {
	return c.appendDirective("no-transform")
}

// MustRevalidate 追加 must-revalidate 指令。
func (c CacheControl) MustRevalidate() CacheControl {
	return c.appendDirective("must-revalidate")
}

// ProxyRevalidate 追加 proxy-revalidate 指令。
func (c CacheControl) ProxyRevalidate() CacheControl {
	return c.appendDirective("proxy-revalidate")
}

// Immutable 追加 immutable 指令。
func (c CacheControl) Immutable() CacheControl {
	return c.appendDirective("immutable")
}

// MaxAge 追加 max-age 指令。
func (c CacheControl) MaxAge(age time.Duration) CacheControl {
	seconds, ok := cacheSeconds(age)
	if !ok {
		return c
	}
	return c.appendDirective("max-age=" + strconv.FormatInt(seconds, 10))
}

// SMaxAge 追加 s-maxage 指令。
func (c CacheControl) SMaxAge(age time.Duration) CacheControl {
	seconds, ok := cacheSeconds(age)
	if !ok {
		return c
	}
	return c.appendDirective("s-maxage=" + strconv.FormatInt(seconds, 10))
}

// HeaderValue 返回可写入响应头的缓存指令值。
func (c CacheControl) HeaderValue() string {
	if len(c.directives) == 0 {
		return ""
	}
	return strings.Join(c.directives, ", ")
}

func (c CacheControl) appendDirective(value string) CacheControl {
	value = cleanHeaderValue(value)
	if value == "" {
		return c
	}
	for _, item := range c.directives {
		if strings.EqualFold(item, value) {
			return c
		}
	}
	next := make([]string, 0, len(c.directives)+1)
	next = append(next, c.directives...)
	next = append(next, value)
	c.directives = next
	return c
}

func cacheSeconds(age time.Duration) (int64, bool) {
	if age < 0 {
		return 0, false
	}
	return int64(age / time.Second), true
}

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
