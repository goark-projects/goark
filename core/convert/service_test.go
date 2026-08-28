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
