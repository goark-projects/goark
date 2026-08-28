package webtest

import (
	"context"

	arkjson "goark.dev/arkarta/json"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
)

// Option 定制 WebTest 客户端。
type Option func(*clientConfig)

type clientConfig struct {
	codec          arkjson.Codec
	netHTTPOptions []servletnethttp.Option
	close          func(context.Context) error
}

func newClientConfig(options []Option) clientConfig {
	config := clientConfig{
		codec: arkjson.DefaultCodec(),
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.codec == nil {
		config.codec = arkjson.DefaultCodec()
	}
	return config
}

// WithJSONCodec 设置测试请求体编码和响应体解码使用的 JSON 编解码器。
func WithJSONCodec(codec arkjson.Codec) Option {
	return func(config *clientConfig) {
		if codec != nil {
			config.codec = codec
		}
	}
}

// WithNetHTTPOptions 设置 Arkarta net/http 适配器选项。
func WithNetHTTPOptions(options ...servletnethttp.Option) Option {
	return func(config *clientConfig) {
		for _, option := range options {
			if option != nil {
				config.netHTTPOptions = append(config.netHTTPOptions, option)
			}
		}
	}
}
