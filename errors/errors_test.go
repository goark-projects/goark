package errors_test

import (
	stderrors "errors"
	"io"
	"strings"
	"testing"

	arkerrors "goark.dev/goark/errors"
)

func TestError_whenWrapped_shouldExposeCodeAndCause(t *testing.T) {
	err := arkerrors.Wrap(arkerrors.CodeCreation, io.EOF, "create bean failed")

	if !arkerrors.Is(err, arkerrors.CodeCreation) {
		t.Fatalf("expected error code %s, got %v", arkerrors.CodeCreation, err)
	}
	if !stderrors.Is(err, io.EOF) {
		t.Fatalf("expected wrapped cause %v, got %v", io.EOF, err)
	}
	code, ok := arkerrors.CodeOf(err)
	if !ok || code != arkerrors.CodeCreation {
		t.Fatalf("expected code %s, got %s, %v", arkerrors.CodeCreation, code, ok)
	}
	if !strings.Contains(err.Error(), "create bean failed") {
		t.Fatalf("expected message to contain context, got %q", err.Error())
	}
}

func TestCodeOf_whenErrorIsNotGoarkError_shouldReturnFalse(t *testing.T) {
	code, ok := arkerrors.CodeOf(io.EOF)
	if ok {
		t.Fatalf("expected no code, got %s", code)
	}
}
