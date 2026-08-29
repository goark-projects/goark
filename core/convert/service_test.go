package convert_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"goark.dev/goark/core/convert"
)

func TestService_whenUsingBuiltinConverters_shouldConvertCommonTypes(t *testing.T) {
	service := convert.DefaultService()

	port, err := convert.Convert[int](service, "8080")
	if err != nil {
		t.Fatalf("convert int failed: %v", err)
	}
	if port != 8080 {
		t.Fatalf("unexpected int: %d", port)
	}

	timeout, err := convert.Convert[time.Duration](service, "150ms")
	if err != nil {
		t.Fatalf("convert duration failed: %v", err)
	}
	if timeout != 150*time.Millisecond {
		t.Fatalf("unexpected duration: %v", timeout)
	}

	flags, err := convert.Convert[[]bool](service, "true,false,true")
	if err != nil {
		t.Fatalf("convert bool slice failed: %v", err)
	}
	expected := []bool{true, false, true}
	if !reflect.DeepEqual(flags, expected) {
		t.Fatalf("unexpected bool slice: %#v", flags)
	}
}

func TestService_whenCustomConverterRegistered_shouldUseConverter(t *testing.T) {
	type source struct {
		value string
	}
	type target struct {
		value string
	}

	service, err := convert.NewService(convert.ConverterFunc[source, target](func(value source) (target, error) {
		return target{value: "converted:" + value.value}, nil
	}))
	if err != nil {
		t.Fatalf("create service failed: %v", err)
	}

	converted, err := convert.Convert[target](service, source{value: "goark"})
	if err != nil {
		t.Fatalf("convert custom value failed: %v", err)
	}
	if converted.value != "converted:goark" {
		t.Fatalf("unexpected custom conversion: %#v", converted)
	}
}

func TestService_whenCloned_shouldPreserveConvertersWithoutMutatingOriginal(t *testing.T) {
	type targetA struct {
		value string
	}
	type targetB struct {
		value string
	}

	service, err := convert.NewService(convert.ConverterFunc[string, targetA](func(value string) (targetA, error) {
		return targetA{value: "a:" + value}, nil
	}))
	if err != nil {
		t.Fatalf("create service failed: %v", err)
	}
	cloned, err := service.Clone()
	if err != nil {
		t.Fatalf("clone service failed: %v", err)
	}
	if err := cloned.Register(convert.ConverterFunc[string, targetB](func(value string) (targetB, error) {
		return targetB{value: "b:" + value}, nil
	})); err != nil {
		t.Fatalf("register cloned converter failed: %v", err)
	}

	a, err := convert.Convert[targetA](cloned, "goark")
	if err != nil {
		t.Fatalf("clone should preserve original converter: %v", err)
	}
	if a.value != "a:goark" {
		t.Fatalf("targetA = %#v", a)
	}
	b, err := convert.Convert[targetB](cloned, "goark")
	if err != nil {
		t.Fatalf("clone should use new converter: %v", err)
	}
	if b.value != "b:goark" {
		t.Fatalf("targetB = %#v", b)
	}
	if _, err := convert.Convert[targetB](service, "goark"); err == nil {
		t.Fatal("original service should not see cloned converter")
	}
}

func TestService_whenTargetCannotAcceptNil_shouldReturnError(t *testing.T) {
	service := convert.DefaultService()

	_, err := convert.Convert[int](service, nil)
	if err == nil {
		t.Fatal("expected nil conversion error")
	}
}

func TestService_whenConvertingSliceToInterfaceSliceWithNilElement_shouldPreserveNil(t *testing.T) {
	service := convert.DefaultService()

	var nilBuffer *bytes.Buffer
	converted, err := service.Convert([]*bytes.Buffer{nilBuffer, bytes.NewBufferString("goark")}, reflect.TypeOf([]io.Reader{}))
	if err != nil {
		t.Fatalf("convert slice with nil interface element failed: %v", err)
	}
	readers := converted.([]io.Reader)
	if len(readers) != 2 {
		t.Fatalf("expected two readers, got %d", len(readers))
	}
	if readers[0] != nil {
		t.Fatalf("expected nil reader at index 0, got %#v", readers[0])
	}
	if readers[1] == nil {
		t.Fatal("expected non-nil reader at index 1")
	}
}
