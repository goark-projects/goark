package websocket_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servletcontainer "goark.dev/arkarta/servlet/container"
	servletnethttp "goark.dev/arkarta/servlet/nethttp"
	arkws "goark.dev/arkarta/websocket"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	gowebsocket "goark.dev/goark/web/websocket"
)

func TestConfigurerRegistersUpgradeServlet(t *testing.T) {
	t.Parallel()

	configurer, err := gowebsocket.New("/ws", arkws.EndpointFunc{}, gowebsocket.WithServletName("chatSocket"), gowebsocket.WithSubprotocols("chat"))
	if err != nil {
		t.Fatalf("websocket.New failed: %v", err)
	}
	registry := goweb.NewRegistry()
	if err := configurer.ConfigureWeb(t.Context(), registry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	deployment, err := goweb.BuildDeployment(registry, goweb.DeploymentSpec{})
	if err != nil {
		t.Fatalf("BuildDeployment failed: %v", err)
	}
	if !servletcontainer.SupportsProfile(deployment.Profiles(), servletcontainer.ProfileUpgrade) {
		t.Fatalf("profiles = %#v, want upgrade profile", deployment.Profiles())
	}
	handler, err := deployment.Handler()
	if err != nil {
		t.Fatalf("Deployment Handler failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	servletnethttp.Handler(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "missing Connection upgrade token") {
		t.Fatalf("body = %q, want websocket handshake error", recorder.Body.String())
	}
}

func TestRegisterEndpointContributesConfigurer(t *testing.T) {
	t.Parallel()

	beanRegistry := container.NewRegistry()
	if err := gowebsocket.RegisterEndpoint(beanRegistry, "chatWebSocket", "/chat", arkws.EndpointFunc{}); err != nil {
		t.Fatalf("RegisterEndpoint failed: %v", err)
	}
	resolver, err := container.New(beanRegistry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}
	registry := goweb.NewRegistry()
	if err := goweb.ApplyConfigurers(t.Context(), resolver, registry); err != nil {
		t.Fatalf("ApplyConfigurers failed: %v", err)
	}
	deployment, err := goweb.BuildDeployment(registry, goweb.DeploymentSpec{})
	if err != nil {
		t.Fatalf("BuildDeployment failed: %v", err)
	}
	if !servletcontainer.SupportsProfile(deployment.Profiles(), servletcontainer.ProfileUpgrade) {
		t.Fatalf("profiles = %#v, want upgrade profile", deployment.Profiles())
	}
}

func TestNewRejectsNilEndpoint(t *testing.T) {
	t.Parallel()

	_, err := gowebsocket.New("/ws", nil)
	if !errors.Is(err, arkws.ErrNilEndpoint) {
		t.Fatalf("err = %v, want ErrNilEndpoint", err)
	}
}
