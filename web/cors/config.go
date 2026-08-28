package cors

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	// AllOrigins 表示允许任意 Origin。
	AllOrigins = "*"
	// AllMethods 表示允许任意 HTTP 方法。
	AllMethods = "*"
	// AllHeaders 表示允许任意请求头。
	AllHeaders = "*"
)

const defaultMaxAge = 30 * time.Minute

var (
	// ErrInvalidOrigin 表示 Origin 规则非法。
	ErrInvalidOrigin = errors.New("goark/web/cors: invalid origin")
	// ErrInvalidMethod 表示 HTTP 方法规则非法。
	ErrInvalidMethod = errors.New("goark/web/cors: invalid method")
	// ErrNilChain 表示过滤器链为空。
	ErrNilChain = errors.New("goark/web/cors: chain is nil")
)

// Config 描述 CORS 访问控制策略。
type Config struct {
	AllowedOrigins        []string
	AllowedOriginPatterns []string
	AllowedMethods        []string
	AllowedHeaders        []string
	ExposedHeaders        []string
	AllowCredentials      bool
	MaxAge                time.Duration
}

// PermitDefaultValues 返回贴近 Spring CorsConfiguration 默认许可值的策略。
func PermitDefaultValues() Config {
	return Config{
		AllowedOrigins: []string{AllOrigins},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
		},
		AllowedHeaders: []string{AllHeaders},
		MaxAge:         defaultMaxAge,
	}
}

type compiledConfig struct {
	allowAllOrigins       bool
	allowAllMethods       bool
	allowAllHeaders       bool
	allowedOrigins        map[string]struct{}
	allowedOriginPatterns []string
	allowedMethods        []string
	allowedMethodSet      map[string]struct{}
	allowedHeaders        []string
	allowedHeaderSet      map[string]struct{}
	exposedHeaders        []string
	allowCredentials      bool
	maxAge                time.Duration
}

func compileConfig(config Config) (compiledConfig, error) {
	origins, allowAllOrigins, err := cleanOrigins(config.AllowedOrigins)
	if err != nil {
		return compiledConfig{}, err
	}
	methods, allowAllMethods, err := cleanMethods(config.AllowedMethods)
	if err != nil {
		return compiledConfig{}, err
	}
	headers, allowAllHeaders := cleanHeaders(config.AllowedHeaders)
	patterns, _, err := cleanOrigins(config.AllowedOriginPatterns)
	if err != nil {
		return compiledConfig{}, err
	}
	return compiledConfig{
		allowAllOrigins:       allowAllOrigins,
		allowAllMethods:       allowAllMethods,
		allowAllHeaders:       allowAllHeaders,
		allowedOrigins:        stringSet(origins),
		allowedOriginPatterns: patterns,
		allowedMethods:        methods,
		allowedMethodSet:      stringSet(methods),
		allowedHeaders:        headers,
		allowedHeaderSet:      caseFoldSet(headers),
		exposedHeaders:        cleanHeaderNames(config.ExposedHeaders),
		allowCredentials:      config.AllowCredentials,
		maxAge:                config.MaxAge,
	}, nil
}

func cleanOrigins(values []string) ([]string, bool, error) {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	allowAll := false
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if hasControlByte(value) {
			return nil, false, ErrInvalidOrigin
		}
		if value == AllOrigins {
			allowAll = true
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned, allowAll, nil
}

func cleanMethods(values []string) ([]string, bool, error) {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	allowAll := false
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if value == AllMethods {
			allowAll = true
		} else if !validHTTPToken(value) {
			return nil, false, ErrInvalidMethod
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned, allowAll, nil
}

func cleanHeaders(values []string) ([]string, bool) {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	allowAll := false
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if value == AllHeaders {
			allowAll = true
		} else if !validHTTPToken(value) {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned, allowAll
}

func cleanHeaderNames(values []string) []string {
	cleaned, _ := cleanHeaders(values)
	return cleaned
}

func stringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func caseFoldSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[strings.ToLower(value)] = struct{}{}
	}
	return out
}

func hasControlByte(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f {
			return true
		}
	}
	return false
}

func validHTTPToken(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return value != ""
}
