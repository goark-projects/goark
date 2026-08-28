package filter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strconv"
	"strings"

	"goark.dev/arkarta/servlet"
)

const defaultMaxETagBodyBytes int64 = 1 << 20

// ETagOption 定制浅 ETag 过滤器。
type ETagOption func(*etagOptions)

type etagOptions struct {
	weak         bool
	maxBodyBytes int64
}

type shallowETagFilter struct {
	options etagOptions
}

// WithStrongETag 使用强 ETag，适合小响应且需要强校验的场景。
func WithStrongETag() ETagOption {
	return func(options *etagOptions) {
		options.weak = false
	}
}

// WithMaxBodyBytes 设置参与浅 ETag 计算的最大响应体字节数。
func WithMaxBodyBytes(size int64) ETagOption {
	return func(options *etagOptions) {
		if size >= 0 {
			options.maxBodyBytes = size
		}
	}
}

// ShallowETag 返回基于响应体快照的 ETag 过滤器。
func ShallowETag(options ...ETagOption) servlet.Filter {
	cfg := etagOptions{
		weak:         true,
		maxBodyBytes: defaultMaxETagBodyBytes,
	}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	return shallowETagFilter{options: cfg}
}

func (f shallowETagFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if res == nil {
		return servlet.ErrNilResponse
	}
	if chain == nil {
		return ErrNilChain
	}
	if !etagMethodAllowed(req.Method()) || res.Header().Get("ETag") != "" {
		return chain.Next(ctx, req, res)
	}
	capture := newCaptureResponse(res, f.options.maxBodyBytes)
	if err := chain.Next(ctx, req, capture); err != nil {
		return err
	}
	return capture.finish(req, f.options.weak)
}

type captureResponse struct {
	parent       servlet.Response
	body         bytes.Buffer
	status       int
	maxBodyBytes int64
	passthrough  bool
}

func newCaptureResponse(parent servlet.Response, maxBodyBytes int64) *captureResponse {
	return &captureResponse{
		parent:       parent,
		status:       parent.Status(),
		maxBodyBytes: maxBodyBytes,
	}
}

func (r *captureResponse) Header() http.Header {
	return r.parent.Header()
}

func (r *captureResponse) SetStatus(code int) {
	if r.passthrough {
		r.parent.SetStatus(code)
		return
	}
	r.status = normalizeStatus(code)
}

func (r *captureResponse) Status() int {
	if r.passthrough {
		return r.parent.Status()
	}
	return r.status
}

func (r *captureResponse) Write(data []byte) (int, error) {
	if r.passthrough {
		return r.parent.Write(data)
	}
	if r.maxBodyBytes >= 0 && int64(r.body.Len()+len(data)) > r.maxBodyBytes {
		if err := r.switchToPassthrough(); err != nil {
			return 0, err
		}
		return r.parent.Write(data)
	}
	return r.body.Write(data)
}

func (r *captureResponse) WriteString(value string) (int, error) {
	return r.Write([]byte(value))
}

func (r *captureResponse) Flush() error {
	if !r.passthrough {
		if err := r.switchToPassthrough(); err != nil {
			return err
		}
	}
	return r.parent.Flush()
}

func (r *captureResponse) Committed() bool {
	return r.parent.Committed()
}

func (r *captureResponse) Reset() error {
	if err := r.parent.Reset(); err != nil {
		return err
	}
	r.body.Reset()
	r.status = http.StatusOK
	r.passthrough = false
	return nil
}

func (r *captureResponse) BodyWriter() io.Writer {
	return r
}

func (r *captureResponse) finish(req *servlet.Request, weak bool) error {
	if r.passthrough || r.parent.Committed() {
		return nil
	}
	status := normalizeStatus(r.status)
	body := r.body.Bytes()
	if !etagStatusAllowed(status) || cacheControlNoStore(r.Header().Get("Cache-Control")) || r.Header().Get("ETag") != "" {
		return r.writeCaptured(req, status, body)
	}
	etag := makeETag(body, weak)
	r.Header().Set("ETag", etag)
	if ifNoneMatch(req.Header().Values("If-None-Match"), etag) {
		r.Header().Del("Content-Length")
		r.parent.SetStatus(http.StatusNotModified)
		return nil
	}
	if entityStatusAllowsBody(status) {
		r.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	return r.writeCaptured(req, status, body)
}

func (r *captureResponse) writeCaptured(req *servlet.Request, status int, body []byte) error {
	r.parent.SetStatus(status)
	if !entityStatusAllowsBody(status) || strings.EqualFold(req.Method(), http.MethodHead) {
		return nil
	}
	_, err := r.parent.Write(body)
	return err
}

func (r *captureResponse) switchToPassthrough() error {
	if r.passthrough {
		return nil
	}
	r.passthrough = true
	r.parent.SetStatus(r.status)
	if r.body.Len() == 0 {
		return nil
	}
	_, err := r.parent.Write(r.body.Bytes())
	r.body.Reset()
	return err
}

func makeETag(body []byte, weak bool) string {
	if !weak {
		sum := sha256.Sum256(body)
		return `"` + hex.EncodeToString(sum[:]) + `"`
	}
	hash := fnv.New64a()
	_, _ = hash.Write(body)
	return fmt.Sprintf(`W/"%x-%x"`, len(body), hash.Sum64())
}

func ifNoneMatch(values []string, etag string) bool {
	if etag == "" {
		return false
	}
	normalized := normalizeETag(etag)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "*" || normalizeETag(item) == normalized {
				return true
			}
		}
	}
	return false
}

func normalizeETag(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	return value
}

func etagMethodAllowed(method string) bool {
	return strings.EqualFold(method, http.MethodGet) || strings.EqualFold(method, http.MethodHead)
}

func etagStatusAllowed(status int) bool {
	return status >= http.StatusOK && status < http.StatusMultipleChoices && entityStatusAllowsBody(status)
}

func entityStatusAllowsBody(status int) bool {
	return status != http.StatusNoContent && status != http.StatusResetContent && status != http.StatusNotModified
}

func cacheControlNoStore(value string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(item), "no-store") {
			return true
		}
	}
	return false
}

func normalizeStatus(code int) int {
	if code < 100 || code > 999 {
		return http.StatusInternalServerError
	}
	return code
}
