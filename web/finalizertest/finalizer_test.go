package finalizertest

import (
	"context"
	"testing"

	"goark.dev/goark/container"
	"goark.dev/goark/web"
)

type contributor struct{ phase *[]string }

func (c contributor) ConfigureWeb(context.Context, *web.Registry) error {
	*c.phase = append(*c.phase, "configure")
	return nil
}
func (c contributor) FinalizeWeb(context.Context, *web.Registry) error {
	*c.phase = append(*c.phase, "finalize")
	return nil
}

func TestFinalizersRunAfterAllContributors(t *testing.T) {
	phases := []string{}
	beans := container.NewRegistry()
	for _, name := range []string{"first", "second"} {
		if err := web.RegisterConfigurer(beans, name, contributor{&phases}); err != nil {
			t.Fatal(err)
		}
	}
	resolver, err := container.New(beans)
	if err != nil {
		t.Fatal(err)
	}
	if err := web.ApplyConfigurers(t.Context(), resolver, web.NewRegistry()); err != nil {
		t.Fatal(err)
	}
	if len(phases) != 4 || phases[1] != "configure" || phases[2] != "finalize" {
		t.Fatal(phases)
	}
}
