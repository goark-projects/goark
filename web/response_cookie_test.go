package web_test

import (
	"net/http"
	"testing"
	"time"

	"goark.dev/goark/web"
)

func TestResponseCookieBuildsSetCookieValue(t *testing.T) {
	t.Parallel()

	cookie := web.NewResponseCookie("sid", "abc").
		WithPath("/").
		WithDomain("example.com").
		WithMaxAge(time.Minute).
		WithSecure(true).
		WithHTTPOnly(true).
		WithSameSite(http.SameSiteLaxMode)

	if got := cookie.String(); got != "sid=abc; Path=/; Domain=example.com; Max-Age=60; HttpOnly; Secure; SameSite=Lax" {
		t.Fatalf("cookie = %q", got)
	}
}

func TestResponseCookieRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []web.ResponseCookie{
		web.NewResponseCookie("", "abc"),
		web.NewResponseCookie("bad name", "abc"),
		web.NewResponseCookie("sid", "a\nb"),
		web.NewResponseCookie("sid", "abc").WithPath("/bad\npath"),
		web.NewResponseCookie("sid", "abc").WithDomain("bad domain"),
	}
	for _, cookie := range tests {
		if got := cookie.String(); got != "" {
			t.Fatalf("invalid cookie string = %q, want empty", got)
		}
	}
}

func TestResponseEntityAddsResponseCookie(t *testing.T) {
	t.Parallel()

	entity := web.NoBody(http.StatusNoContent).
		WithResponseCookie(web.NewResponseCookie("sid", "abc").WithHTTPOnly(true)).
		WithResponseCookie(web.ResponseCookie{})

	if got := entity.Headers().Values("Set-Cookie"); len(got) != 1 || got[0] != "sid=abc; HttpOnly" {
		t.Fatalf("Set-Cookie = %#v", got)
	}
}

func TestResponseCookieReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	cookie := web.NewResponseCookie("sid", "abc").WithPath("/")
	raw := cookie.Cookie()
	raw.Path = "/mutated"

	if got := cookie.String(); got != "sid=abc; Path=/" {
		t.Fatalf("cookie mutated through copy: %q", got)
	}
}
