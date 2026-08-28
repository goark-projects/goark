package webtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

// Client 使用内存 HTTP 处理器执行 Goark Web 请求。
type Client struct {
	handler http.Handler
	config  clientConfig
}

// New 创建基于标准库 http.Handler 的测试客户端。
func New(handler http.Handler, options ...Option) (*Client, error) {
	if isNilValue(handler) {
		return nil, ErrNilHTTPHandler
	}
	return &Client{
		handler: handler,
		config:  newClientConfig(options),
	}, nil
}

// NewServlet 创建基于 Arkarta Servlet Handler 的测试客户端。
func NewServlet(handler servlet.Handler, options ...Option) (*Client, error) {
	if isNilValue(handler) {
		return nil, ErrNilServletHandler
	}
	config := newClientConfig(options)
	return &Client{
		handler: servletnethttp.HandlerWithOptions(handler, config.netHTTPOptions...),
		config:  config,
	}, nil
}

// NewRouter 创建基于 Arkarta Web Router 的测试客户端。
func NewRouter(router *arkweb.Router, options ...Option) (*Client, error) {
	if router == nil {
		return nil, ErrNilRouter
	}
	return NewServlet(router, options...)
}

// NewRegistry 创建带 Servlet 生命周期初始化的 Goark Web 注册表测试客户端。
func NewRegistry(ctx context.Context, registry *goweb.Registry, spec goweb.DeploymentSpec, options ...Option) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config := newClientConfig(options)
	deployment, err := goweb.BuildDeployment(registry, spec)
	if err != nil {
		return nil, err
	}
	app, err := servletcontainer.NewApplication(ctx, deployment)
	if err != nil {
		return nil, err
	}
	netHTTPOptions := make([]servletnethttp.Option, 0, len(config.netHTTPOptions)+1)
	if webApp := app.WebApp(); webApp != nil {
		netHTTPOptions = append(netHTTPOptions, servletnethttp.WithRequestContextPath(webApp.ContextPath()))
	}
	netHTTPOptions = append(netHTTPOptions, config.netHTTPOptions...)
	config.close = app.Stop
	return &Client{
		handler: servletnethttp.HandlerWithOptions(app.Handler(), netHTTPOptions...),
		config:  config,
	}, nil
}

// Must 在测试中快速处理客户端构造错误。
func Must(t testing.TB, client *Client, err error) *Client {
	t.Helper()
	if err != nil {
		t.Fatalf("webtest client failed: %v", err)
	}
	return client
}

// Do 执行已经构造好的 HTTP 请求。
func (c *Client) Do(request *http.Request) *Response {
	recorder := httptest.NewRecorder()
	if c == nil || c.handler == nil {
		return newResponse(recorder, nil, ErrNilHTTPHandler)
	}
	if request == nil {
		return newResponse(recorder, c.config.codec, ErrNilRequest)
	}
	c.handler.ServeHTTP(recorder, request)
	return newResponse(recorder, c.config.codec, nil)
}

// Execute 构造并执行一次 HTTP 请求。
func (c *Client) Execute(method string, target string, options ...RequestOption) (*Response, error) {
	codec := arkCodec(c)
	request, err := newRequest(method, target, codec, options...)
	if err != nil {
		return nil, err
	}
	return c.Do(request), nil
}

// Perform 在测试中构造并执行一次 HTTP 请求。
func (c *Client) Perform(t testing.TB, method string, target string, options ...RequestOption) *Response {
	t.Helper()
	response, err := c.Execute(method, target, options...)
	if err != nil {
		t.Fatalf("webtest request failed: %v", err)
	}
	return response
}

// Close 关闭 Registry 客户端持有的 Servlet 应用生命周期。
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.config.close == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.config.close(ctx)
}

func arkCodec(c *Client) arkjson.Codec {
	if c == nil || c.config.codec == nil {
		return nil
	}
	return c.config.codec
}

func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflectValue.IsNil()
	default:
		return false
	}
}
