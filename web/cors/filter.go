package cors

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"goark.dev/arkarta/servlet"
)

const (
	headerOrigin                    = "Origin"
	headerVary                      = "Vary"
	headerAllowOrigin               = "Access-Control-Allow-Origin"
	headerAllowMethods              = "Access-Control-Allow-Methods"
	headerAllowHeaders              = "Access-Control-Allow-Headers"
	headerAllowCredentials          = "Access-Control-Allow-Credentials"
	headerExposeHeaders             = "Access-Control-Expose-Headers"
	headerMaxAge                    = "Access-Control-Max-Age"
	headerRequestMethod             = "Access-Control-Request-Method"
	headerRequestHeaders            = "Access-Control-Request-Headers"
	varyOrigin                      = "Origin"
	varyAccessControlRequestMethod  = "Access-Control-Request-Method"
	varyAccessControlRequestHeaders = "Access-Control-Request-Headers"
	allowCredentialsValue           = "true"
)

// Filter 按 CORS 策略处理预检请求和实际跨域请求。
type Filter struct {
	config compiledConfig
}

// New 创建 CORS 过滤器。
func New(config Config) (*Filter, error) {
	compiled, err := compileConfig(config)
	if err != nil {
		return nil, err
	}
	return &Filter{config: compiled}, nil
}

// PermitAll 创建允许全部 Origin、方法和请求头的 CORS 过滤器。
func PermitAll() (*Filter, error) {
	config := PermitDefaultValues()
	config.AllowedMethods = []string{AllMethods}
	return New(config)
}

// Filter 执行 CORS 访问控制。
func (f *Filter) Filter(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
	if req == nil {
		return servlet.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), nil)
	}
	if res == nil {
		return servlet.ErrNilResponse
	}
	origin := strings.TrimSpace(req.Header().Get(headerOrigin))
	if origin == "" {
		if chain == nil {
			return ErrNilChain
		}
		return chain.Next(ctx, req, res)
	}
	if isPreflight(req) {
		return f.handlePreflight(req, res)
	}
	if chain == nil {
		return ErrNilChain
	}
	return f.handleActual(ctx, req, res, chain, origin)
}

func (f *Filter) handlePreflight(req *servlet.Request, res servlet.Response) error {
	origin := strings.TrimSpace(req.Header().Get(headerOrigin))
	requestMethod := strings.ToUpper(strings.TrimSpace(req.Header().Get(headerRequestMethod)))
	requestHeaders := requestedHeaders(req.Header().Get(headerRequestHeaders))
	if !f.config.originAllowed(origin) ||
		!f.config.methodAllowed(requestMethod) ||
		!f.config.headersAllowed(requestHeaders) {
		return servlet.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden), nil)
	}
	addCorsVaryHeaders(res.Header())
	f.writeAllowOrigin(res.Header(), origin)
	f.writeAllowCredentials(res.Header())
	f.writeAllowMethods(res.Header(), requestMethod)
	f.writeAllowHeaders(res.Header(), requestHeaders)
	if f.config.maxAge > 0 {
		res.Header().Set(headerMaxAge, strconv.FormatInt(int64(f.config.maxAge.Seconds()), 10))
	}
	res.SetStatus(http.StatusNoContent)
	return nil
}

func (f *Filter) handleActual(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain, origin string) error {
	if !f.config.originAllowed(origin) || !f.config.methodAllowed(req.Method()) {
		return servlet.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden), nil)
	}
	addVary(res.Header(), varyOrigin)
	f.writeAllowOrigin(res.Header(), origin)
	f.writeAllowCredentials(res.Header())
	if len(f.config.exposedHeaders) > 0 {
		res.Header().Set(headerExposeHeaders, strings.Join(f.config.exposedHeaders, ", "))
	}
	return chain.Next(ctx, req, res)
}

func (f *Filter) writeAllowOrigin(header http.Header, origin string) {
	if f.config.allowAllOrigins && !f.config.allowCredentials {
		header.Set(headerAllowOrigin, AllOrigins)
		return
	}
	header.Set(headerAllowOrigin, origin)
}

func (f *Filter) writeAllowCredentials(header http.Header) {
	if f.config.allowCredentials {
		header.Set(headerAllowCredentials, allowCredentialsValue)
	}
}

func (f *Filter) writeAllowMethods(header http.Header, requestedMethod string) {
	if f.config.allowAllMethods {
		header.Set(headerAllowMethods, requestedMethod)
		return
	}
	header.Set(headerAllowMethods, strings.Join(f.config.allowedMethods, ", "))
}

func (f *Filter) writeAllowHeaders(header http.Header, requestedHeaders []string) {
	if f.config.allowAllHeaders {
		if len(requestedHeaders) == 0 {
			header.Set(headerAllowHeaders, AllHeaders)
			return
		}
		header.Set(headerAllowHeaders, strings.Join(requestedHeaders, ", "))
		return
	}
	if len(f.config.allowedHeaders) > 0 {
		header.Set(headerAllowHeaders, strings.Join(f.config.allowedHeaders, ", "))
	}
}

func isPreflight(req *servlet.Request) bool {
	return strings.EqualFold(req.Method(), http.MethodOptions) &&
		strings.TrimSpace(req.Header().Get(headerRequestMethod)) != ""
}

func addCorsVaryHeaders(header http.Header) {
	addVary(header, varyOrigin)
	addVary(header, varyAccessControlRequestMethod)
	addVary(header, varyAccessControlRequestHeaders)
}

func addVary(header http.Header, value string) {
	for _, existing := range header.Values(headerVary) {
		for _, item := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add(headerVary, value)
}
