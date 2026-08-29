package mvc

import (
	"net/http"
	"strings"

	"goark.dev/goark/web/cors"
)

func cloneCrossOriginConfig(config cors.Config) *cors.Config {
	copied := cors.Config{
		AllowedOrigins:        append([]string(nil), config.AllowedOrigins...),
		AllowedOriginPatterns: append([]string(nil), config.AllowedOriginPatterns...),
		AllowedMethods:        append([]string(nil), config.AllowedMethods...),
		AllowedHeaders:        append([]string(nil), config.AllowedHeaders...),
		ExposedHeaders:        append([]string(nil), config.ExposedHeaders...),
		AllowCredentials:      config.AllowCredentials,
		MaxAge:                config.MaxAge,
	}
	return &copied
}

func crossOriginMethods(method string) []string {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case http.MethodGet:
		return []string{http.MethodGet, http.MethodHead}
	default:
		if method == "" {
			return nil
		}
		return []string{method}
	}
}
