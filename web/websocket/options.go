package websocket

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"

	"goark.dev/arkarta/servlet"
	arkws "goark.dev/arkarta/websocket"
	servletws "goark.dev/arkarta/websocket/servlet"
)

// SessionIDGenerator 为每个 WebSocket 连接生成会话标识。
type SessionIDGenerator func(ctx context.Context, req *servlet.Request) (string, error)

type config struct {
	servletName        string
	handshakeOptions   []arkws.HandshakeOption
	frameOptions       []servletws.FrameConnectionOption
	sessionIDGenerator SessionIDGenerator
}

// Option 定制 WebSocket 端点注册。
type Option func(*config) error

// WithServletName 设置底层 Servlet 名称。
func WithServletName(name string) Option {
	return func(cfg *config) error {
		cfg.servletName = strings.TrimSpace(name)
		return nil
	}
}

// WithSubprotocols 设置服务端支持的 WebSocket 子协议。
func WithSubprotocols(protocols ...string) Option {
	return WithHandshakeOptions(arkws.WithSubprotocols(protocols...))
}

// WithHandshakeOptions 追加底层 WebSocket 握手选项。
func WithHandshakeOptions(options ...arkws.HandshakeOption) Option {
	copied := append([]arkws.HandshakeOption(nil), options...)
	return func(cfg *config) error {
		for _, option := range copied {
			if option != nil {
				cfg.handshakeOptions = append(cfg.handshakeOptions, option)
			}
		}
		return nil
	}
}

// WithFrameOptions 追加底层 WebSocket 帧连接选项。
func WithFrameOptions(options ...servletws.FrameConnectionOption) Option {
	copied := append([]servletws.FrameConnectionOption(nil), options...)
	return func(cfg *config) error {
		for _, option := range copied {
			if option != nil {
				cfg.frameOptions = append(cfg.frameOptions, option)
			}
		}
		return nil
	}
}

// WithMaxFrameBytes 设置单帧最大载荷字节数。
func WithMaxFrameBytes(maxBytes int64) Option {
	return WithFrameOptions(servletws.WithMaxFrameBytes(maxBytes))
}

// WithMaxMessageBytes 设置聚合消息最大载荷字节数。
func WithMaxMessageBytes(maxBytes int64) Option {
	return WithFrameOptions(servletws.WithMaxMessageBytes(maxBytes))
}

// WithSessionIDGenerator 设置 WebSocket 会话标识生成器。
func WithSessionIDGenerator(generator SessionIDGenerator) Option {
	return func(cfg *config) error {
		if generator != nil {
			cfg.sessionIDGenerator = generator
		}
		return nil
	}
}

func defaultSessionIDGenerator(name string) SessionIDGenerator {
	prefix := strings.TrimSpace(name)
	if prefix == "" {
		prefix = "websocket"
	}
	var sequence atomic.Uint64
	return func(ctx context.Context, _ *servlet.Request) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return prefix + "-" + strconv.FormatUint(sequence.Add(1), 36), nil
	}
}
