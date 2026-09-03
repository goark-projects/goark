package env_test

import (
	"strings"
	"testing"

	"goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

func TestValidateConfigurationPropertyNames_whenPropertiesKnown_shouldPass(t *testing.T) {
	environment := environmentWithProperties(t, map[string]any{
		"server.port":               "8080",
		"server.hertz.idle-timeout": "30s",
		"logging.level.root":        "INFO",
	})
	err := env.ValidateConfigurationPropertyNames(environment, "server", []string{
		"server.port",
		"server.hertz.idle-timeout",
	})
	if err != nil {
		t.Fatalf("ValidateConfigurationPropertyNames() error = %v", err)
	}
}

func TestValidateConfigurationPropertyNames_whenPropertyUnknown_shouldReturnSortedError(t *testing.T) {
	environment := environmentWithProperties(t, map[string]any{
		"server.unknown-z": "1",
		"server.port":      "8080",
		"server.unknown-a": "2",
	})
	err := env.ValidateConfigurationPropertyNames(environment, "server", []string{"server.port"})
	if !arkerrors.Is(err, arkerrors.CodeInvalidArgument) {
		t.Fatalf("ValidateConfigurationPropertyNames() error = %v, want invalid argument", err)
	}
	want := "unknown configuration properties: server.unknown-a, server.unknown-z"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("ValidateConfigurationPropertyNames() error = %v, want %q", err, want)
	}
}

func environmentWithProperties(t *testing.T, properties map[string]any) *env.StandardEnvironment {
	t.Helper()
	environment := env.MustNewStandardEnvironment()
	source, err := env.NewMapPropertySource("test", properties)
	if err != nil {
		t.Fatalf("NewMapPropertySource() error = %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("AddFirst() error = %v", err)
	}
	return environment
}
