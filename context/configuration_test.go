package context_test

import (
	stdcontext "context"
	stderrors "errors"
	"reflect"
	"testing"

	"github.com/goark-projects/goark/container"
	appcontext "github.com/goark-projects/goark/context"
	coreenv "github.com/goark-projects/goark/core/env"
	arkerrors "github.com/goark-projects/goark/errors"
)

type testConfiguration struct {
	name  string
	order int
	log   *[]string
}

func (c *testConfiguration) Name() string {
	return c.name
}

func (c *testConfiguration) Order() int {
	return c.order
}

func (c *testConfiguration) ConfigureEnvironment(stdcontext.Context, coreenv.ConfigurableEnvironment) error {
	*c.log = append(*c.log, "configure:"+c.name)
	return nil
}

func (c *testConfiguration) Register(_ stdcontext.Context, registry *container.Registry) error {
	*c.log = append(*c.log, "register:"+c.name)
	return container.RegisterInstance[string](registry, "configuration."+c.name, c.name)
}

type priorityTestConfiguration struct {
	*testConfiguration
}

func (c *priorityTestConfiguration) PriorityOrdered() {
}

type failingConfiguration struct {
	name string
	fail bool
}

func (c *failingConfiguration) Name() string {
	return c.name
}

func (c *failingConfiguration) Order() int {
	return 0
}

func (c *failingConfiguration) Register(_ stdcontext.Context, registry *container.Registry) error {
	if err := container.RegisterInstance[string](registry, "configuration.partial", "partial"); err != nil {
		return err
	}
	if c.fail {
		return stderrors.New("configuration failed")
	}
	return nil
}

type contextAwareConfiguration struct {
	name string
	log  *[]string
}

func (c *contextAwareConfiguration) Name() string {
	return c.name
}

func (c *contextAwareConfiguration) Order() int {
	return 0
}

func (c *contextAwareConfiguration) Register(stdcontext.Context, *container.Registry) error {
	*c.log = append(*c.log, "legacy-register")
	return nil
}

func (c *contextAwareConfiguration) RegisterWithContext(_ stdcontext.Context, config appcontext.ConfigurationContext) error {
	*c.log = append(*c.log, "context-register")
	if config.Environment() == nil {
		return stderrors.New("environment missing")
	}
	return container.RegisterInstance[string](config.Registry(), "configuration."+c.name, c.name)
}

func TestApplicationContext_whenConfigurationsRegistered_shouldConfigureEnvironmentBeforeRegistration(t *testing.T) {
	log := make([]string, 0, 6)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}

	configurations := []appcontext.Configuration{
		&testConfiguration{name: "late", order: 20, log: &log},
		&priorityTestConfiguration{testConfiguration: &testConfiguration{name: "priority", order: 100, log: &log}},
		&testConfiguration{name: "early", order: 10, log: &log},
	}
	for _, configuration := range configurations {
		if err := app.RegisterConfiguration(configuration); err != nil {
			t.Fatalf("register configuration %s failed: %v", configuration.Name(), err)
		}
	}

	if err := app.Refresh(stdcontext.Background()); err != nil {
		t.Fatalf("refresh app failed: %v", err)
	}

	expected := []string{
		"configure:priority",
		"configure:early",
		"configure:late",
		"register:priority",
		"register:early",
		"register:late",
	}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected configuration flow: %#v", log)
	}
}

func TestApplicationContext_whenConfigurationIsContextAware_shouldUseRegistrationContext(t *testing.T) {
	log := make([]string, 0)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}
	if err := app.RegisterConfiguration(&contextAwareConfiguration{name: "aware", log: &log}); err != nil {
		t.Fatalf("register configuration failed: %v", err)
	}

	if err := app.Refresh(stdcontext.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if !reflect.DeepEqual(log, []string{"context-register"}) {
		t.Fatalf("unexpected registration path: %#v", log)
	}
	value := container.MustGet[string](stdcontext.Background(), app, "configuration.aware")
	if value != "aware" {
		t.Fatalf("unexpected configuration bean: %q", value)
	}
}

func TestProfileCondition_whenExpressionMatches_shouldReturnTrue(t *testing.T) {
	environment, err := coreenv.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	if err := environment.SetActiveProfiles("prod", "mysql"); err != nil {
		t.Fatalf("set active profiles failed: %v", err)
	}
	registry := container.NewRegistry()
	conditionContext := appcontext.NewConfigurationContext(environment, registry)

	matched, err := appcontext.ProfileCondition{Expression: "prod & mysql"}.Matches(conditionContext, appcontext.AnnotationMetadata{Name: "dataSource"})
	if err != nil {
		t.Fatalf("profile condition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected profile condition to match")
	}
}

func TestApplicationContext_whenConfigurationNameDuplicated_shouldReturnError(t *testing.T) {
	log := make([]string, 0)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}

	if err := app.RegisterConfiguration(&testConfiguration{name: "user", log: &log}); err != nil {
		t.Fatalf("register first configuration failed: %v", err)
	}
	if err := app.RegisterConfiguration(&testConfiguration{name: "user", log: &log}); err == nil {
		t.Fatal("expected duplicate configuration error")
	}
}

func TestApplicationContextConfigurations_whenCalled_shouldReturnReadOnlySortedDescriptors(t *testing.T) {
	log := make([]string, 0)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}
	if err := app.RegisterConfiguration(&testConfiguration{name: "late", order: 20, log: &log}); err != nil {
		t.Fatalf("register late failed: %v", err)
	}
	if err := app.RegisterConfiguration(&priorityTestConfiguration{testConfiguration: &testConfiguration{name: "priority", order: 100, log: &log}}); err != nil {
		t.Fatalf("register priority failed: %v", err)
	}

	descriptors := app.Configurations()
	expected := []appcontext.ConfigurationDescriptor{
		{Name: "priority", Order: 100, Priority: true, ConfiguresEnvironment: true},
		{Name: "late", Order: 20, Priority: false, ConfiguresEnvironment: true},
	}
	if !reflect.DeepEqual(descriptors, expected) {
		t.Fatalf("unexpected configuration descriptors: %#v", descriptors)
	}

	descriptors[0].Name = "mutated"
	if app.Configurations()[0].Name != "priority" {
		t.Fatal("configuration descriptor snapshot should not mutate application state")
	}
}

func TestApplicationContext_whenConfigurationRegisteredAfterRefresh_shouldReturnConflict(t *testing.T) {
	log := make([]string, 0)
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}
	if err := app.Refresh(stdcontext.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	err = app.RegisterConfiguration(&testConfiguration{name: "late", log: &log})
	if err == nil {
		t.Fatal("expected configuration registration conflict")
	}
	if !arkerrors.Is(err, arkerrors.CodeConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestApplicationContext_whenConfigurationRegistrationFails_shouldRollbackRegistryAndAllowRetry(t *testing.T) {
	configuration := &failingConfiguration{name: "failing", fail: true}
	app, err := appcontext.New()
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}
	if err := app.RegisterConfiguration(configuration); err != nil {
		t.Fatalf("register configuration failed: %v", err)
	}

	err = app.Refresh(stdcontext.Background())
	if err == nil {
		t.Fatal("expected refresh error")
	}
	if !arkerrors.Is(err, arkerrors.CodeCreation) {
		t.Fatalf("expected creation error, got %v", err)
	}
	_, err = app.Get(stdcontext.Background(), "configuration.partial")
	if err == nil || !arkerrors.Is(err, arkerrors.CodeConflict) {
		t.Fatalf("application should remain unrefreshed after failed configuration registration, got %v", err)
	}

	configuration.fail = false
	if err := app.Refresh(stdcontext.Background()); err != nil {
		t.Fatalf("retry refresh failed: %v", err)
	}
	value := container.MustGet[string](stdcontext.Background(), app, "configuration.partial")
	if value != "partial" {
		t.Fatalf("unexpected retried configuration value: %q", value)
	}
}
