package mvc_test

import (
	"net/http"
	"testing"
	"testing/fstest"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/mvc/view"
)

type modelAccountSummary struct {
	Name string
}

func TestModelInfersAttributeNames(t *testing.T) {
	t.Parallel()

	item := modelAccountSummary{Name: "Goark"}
	model := mvc.NewModel().
		AddAttributeValue(item).
		AddAttributeValue(&item).
		AddAttributeValue([]modelAccountSummary{{Name: "A"}, {Name: "B"}})

	value, ok := model.Attribute("modelAccountSummary")
	if !ok || value != &item {
		t.Fatalf("modelAccountSummary = %#v/%v, want latest pointer value", value, ok)
	}
	list, ok := model.Attribute("modelAccountSummaryList")
	if !ok || len(list.([]modelAccountSummary)) != 2 {
		t.Fatalf("modelAccountSummaryList = %#v/%v, want inferred slice value", list, ok)
	}
}

func TestModelAndViewInfersModelObjectName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	registry.Use(view.Interceptor(modelViewResolver(t, fstest.MapFS{
		"accounts/detail.html": {Data: []byte("<h1>{{.modelAccountSummary.Name}}</h1>")},
	})))
	if err := mvc.NewController("accounts",
		mvc.GET("/accounts/detail", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView("accounts/detail", modelAccountSummary{Name: "Goark"}), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts/detail", "text/html")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.String() != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want inferred model object", recorder.Body.String())
	}
}

func TestRedirectAttributesInfersAttributeName(t *testing.T) {
	t.Parallel()

	registry := web.NewRegistry()
	if err := mvc.NewController("accounts",
		mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			attributes := mvc.NewRedirectAttributes().
				AddAttribute("id", "42").
				AddAttributeValue(modelAccountSummary{Name: "Goark"})
			return mvc.Redirect("/accounts/{id}", attributes), nil
		})),
	).Register(registry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	recorder := serveMVC(t, registry, http.MethodGet, "/accounts", "")
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/accounts/42?modelAccountSummary=%7BGoark%7D" {
		t.Fatalf("Location = %q, want inferred redirect attribute", got)
	}
}
