package filter

import (
	"context"
	"io"
	"mime"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	// DefaultCharacterEncoding 是 Web 请求和响应的默认字符集。
	DefaultCharacterEncoding = "UTF-8"
)

// CharacterEncodingOption 定制字符集过滤器。
type CharacterEncodingOption func(*characterEncodingConfig)

type characterEncodingConfig struct {
	encoding      string
	forceRequest  bool
	forceResponse bool
}

type characterEncodingFilter struct {
	encoding      string
	forceRequest  bool
	forceResponse bool
}

// CharacterEncoding 为请求和响应补齐或强制字符集。
func CharacterEncoding(options ...CharacterEncodingOption) servlet.Filter {
	config := characterEncodingConfig{
		encoding:     DefaultCharacterEncoding,
		forceRequest: false,
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return characterEncodingFilter{
		encoding:      config.encoding,
		forceRequest:  config.forceRequest,
		forceResponse: config.forceResponse,
	}
}

// WithCharacterEncoding 设置请求和响应使用的字符集。
func WithCharacterEncoding(encoding string) CharacterEncodingOption {
	return func(config *characterEncodingConfig) {
		encoding = strings.TrimSpace(encoding)
		if encoding != "" {
			config.encoding = encoding
		}
	}
}

// WithForceRequestEncoding 设置是否覆盖请求已有字符集。
func WithForceRequestEncoding(force bool) CharacterEncodingOption {
	return func(config *characterEncodingConfig) {
		config.forceRequest = force
	}
}

// WithForceResponseEncoding 设置是否覆盖响应已有字符集。
func WithForceResponseEncoding(force bool) CharacterEncodingOption {
	return func(config *characterEncodingConfig) {
		config.forceResponse = force
	}
}

// WithForceEncoding 同时设置请求和响应是否覆盖已有字符集。
func WithForceEncoding(force bool) CharacterEncodingOption {
	return func(config *characterEncodingConfig) {
		config.forceRequest = force
		config.forceResponse = force
	}
}

func (f characterEncodingFilter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if res == nil {
		return servlet.ErrNilResponse
	}
	if chain == nil {
		return ErrNilChain
	}
	applyCharset(req.Header(), f.encoding, f.forceRequest)
	if !f.forceResponse {
		return chain.Next(ctx, req, res)
	}
	response := &characterEncodingResponse{
		parent:   res,
		encoding: f.encoding,
		force:    f.forceResponse,
	}
	if err := chain.Next(ctx, req, response); err != nil {
		return err
	}
	if !response.Committed() {
		response.apply()
	}
	return nil
}

type characterEncodingResponse struct {
	parent   servlet.Response
	encoding string
	force    bool
}

func (r *characterEncodingResponse) Header() http.Header {
	return r.parent.Header()
}

func (r *characterEncodingResponse) SetStatus(code int) {
	r.parent.SetStatus(code)
}

func (r *characterEncodingResponse) Status() int {
	return r.parent.Status()
}

func (r *characterEncodingResponse) Write(data []byte) (int, error) {
	r.apply()
	return r.parent.Write(data)
}

func (r *characterEncodingResponse) WriteString(value string) (int, error) {
	r.apply()
	return r.parent.WriteString(value)
}

func (r *characterEncodingResponse) Flush() error {
	r.apply()
	return r.parent.Flush()
}

func (r *characterEncodingResponse) Committed() bool {
	return r.parent.Committed()
}

func (r *characterEncodingResponse) Reset() error {
	return r.parent.Reset()
}

func (r *characterEncodingResponse) BodyWriter() io.Writer {
	return r
}

func (r *characterEncodingResponse) apply() {
	applyCharset(r.Header(), r.encoding, r.force)
}

func applyCharset(header http.Header, encoding string, force bool) {
	if header == nil {
		return
	}
	encoding = strings.TrimSpace(encoding)
	if encoding == "" {
		return
	}
	value := strings.TrimSpace(header.Get("Content-Type"))
	if value == "" {
		return
	}
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil {
		return
	}
	if params == nil {
		params = map[string]string{}
	}
	if !force && strings.TrimSpace(params["charset"]) != "" {
		return
	}
	params["charset"] = encoding
	header.Set("Content-Type", mime.FormatMediaType(mediaType, params))
}
