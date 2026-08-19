package config_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/goark-projects/goark/config"
	arkerrors "github.com/goark-projects/goark/errors"
)

type bindDatabaseConfig struct {
	DSN          string
	MaxOpenConns int
}

type bindAppConfig struct {
	Name     string
	HTTPPort int
	Debug    bool
	Timeout  time.Duration
	Started  time.Time
	Hosts    []string
	Ratio    float64
	Database *bindDatabaseConfig `goark:"db"`
	Ignored  string              `goark:"-"`
}

func TestBind_whenPropertiesAreValid_shouldPopulateStruct(t *testing.T) {
	env := newTestEnvironment(t, map[string]string{
		"app.name":              "goark",
		"app.http-port":         "8080",
		"app.debug":             "true",
		"app.timeout":           "150ms",
		"app.started":           "2026-08-19T12:00:00Z",
		"app.hosts":             "127.0.0.1,localhost",
		"app.ratio":             "0.75",
		"app.db.dsn":            "memory",
		"app.db.max-open-conns": "32",
		"app.ignored":           "must-not-bind",
	})

	var target bindAppConfig
	if err := config.Bind(env, "app", &target); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	if target.Name != "goark" || target.HTTPPort != 8080 || !target.Debug {
		t.Fatalf("unexpected scalar binding: %#v", target)
	}
	if target.Timeout != 150*time.Millisecond {
		t.Fatalf("unexpected timeout: %v", target.Timeout)
	}
	if target.Started.Format(time.RFC3339) != "2026-08-19T12:00:00Z" {
		t.Fatalf("unexpected started time: %v", target.Started)
	}
	if !reflect.DeepEqual(target.Hosts, []string{"127.0.0.1", "localhost"}) {
		t.Fatalf("unexpected hosts: %#v", target.Hosts)
	}
	if target.Ratio != 0.75 {
		t.Fatalf("unexpected ratio: %v", target.Ratio)
	}
	if target.Database == nil || target.Database.DSN != "memory" || target.Database.MaxOpenConns != 32 {
		t.Fatalf("unexpected database binding: %#v", target.Database)
	}
	if target.Ignored != "" {
		t.Fatalf("ignored field should not be bound: %q", target.Ignored)
	}
}

func TestBind_whenPropertyTypeIsInvalid_shouldReturnTypeMismatch(t *testing.T) {
	env := newTestEnvironment(t, map[string]string{
		"app.http-port": "not-a-number",
	})

	var target bindAppConfig
	err := config.Bind(env, "app", &target)
	if err == nil {
		t.Fatal("expected bind error")
	}
	if !arkerrors.Is(err, arkerrors.CodeTypeMismatch) {
		t.Fatalf("expected type mismatch, got %v", err)
	}
}

func newTestEnvironment(t *testing.T, values map[string]string) *config.Environment {
	t.Helper()
	source, err := config.NewMapSource("test", values)
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	env, err := config.NewEnvironment(source)
	if err != nil {
		t.Fatalf("create environment failed: %v", err)
	}
	return env
}
