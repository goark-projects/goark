package websocket

import (
	"context"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	"goark.dev/arkarta/servlet/upgrade"
	arkws "goark.dev/arkarta/websocket"
	servletws "goark.dev/arkarta/websocket/servlet"
	"goark.dev/goark/container"
	"goark.dev/goark/core/util"
	goweb "goark.dev/goark/web"
)

// Configurer 将 WebSocket Endpoint 挂载到 Web 部署。
type Configurer struct {
	pattern            string
	servletName        string
	endpoint           arkws.Endpoint
	handshakeOptions   []arkws.HandshakeOption
	frameOptions       []servletws.FrameConnectionOption
	sessionIDGenerator SessionIDGenerator
}

// New 创建 WebSocket 端点配置器。
func New(pattern string, endpoint arkws.Endpoint, options ...Option) (Configurer, error) {
	if util.IsNil(endpoint) {
		return Configurer{}, arkws.ErrNilEndpoint
	}
	cfg := config{}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&cfg); err != nil {
			return Configurer{}, err
		}
	}
	servletName := cfg.servletName
	if servletName == "" {
		servletName = pattern
	}
	if cfg.sessionIDGenerator == nil {
		cfg.sessionIDGenerator = defaultSessionIDGenerator(servletName)
	}
	return Configurer{
		pattern:            pattern,
		servletName:        servletName,
		endpoint:           endpoint,
		handshakeOptions:   append([]arkws.HandshakeOption(nil), cfg.handshakeOptions...),
		frameOptions:       append([]servletws.FrameConnectionOption(nil), cfg.frameOptions...),
		sessionIDGenerator: cfg.sessionIDGenerator,
	}, nil
}

// RegisterEndpoint 注册 WebSocket 端点配置器 Bean。
func RegisterEndpoint(registry *container.Registry, name string, pattern string, endpoint arkws.Endpoint, options ...Option) error {
	configurer, err := New(pattern, endpoint, options...)
	if err != nil {
		return err
	}
	return goweb.RegisterConfigurer(registry, name, configurer)
}

// ConfigureWeb 注册 Servlet Upgrade 映射并声明容器能力。
func (c Configurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	registry.RequireProfile(servletcontainer.ProfileUpgrade)
	return registry.AddServlet(c.pattern, c.servletName, c.newServlet())
}

func (c Configurer) newServlet() servlet.Servlet {
	handshakeOptions := append([]arkws.HandshakeOption(nil), c.handshakeOptions...)
	frameOptions := append([]servletws.FrameConnectionOption(nil), c.frameOptions...)
	return endpointServlet{
		handler: servlet.HandlerFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response) error {
			_, err := servletws.Upgrade(ctx, req, res, servletws.HandlerFunc(func(ctx context.Context, handshake arkws.Handshake, conn upgrade.Connection) error {
				sessionID, err := c.sessionIDGenerator(ctx, req)
				if err != nil {
					return err
				}
				return servletws.ServeEndpoint(ctx, sessionID, handshake, conn, c.endpoint, frameOptions...)
			}), handshakeOptions...)
			return err
		}),
	}
}

type endpointServlet struct {
	handler servlet.Handler
}

func (s endpointServlet) Init(ctx context.Context, _ servlet.ServletConfig) error {
	return ctx.Err()
}

func (s endpointServlet) Destroy(ctx context.Context) error {
	return ctx.Err()
}

func (s endpointServlet) Serve(ctx context.Context, req *servlet.Request, res servlet.Response) error {
	if s.handler == nil {
		return servlet.ErrNilHandler
	}
	return s.handler.Serve(ctx, req, res)
}
