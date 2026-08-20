package convert

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goark-projects/goark/core/lang"
	"github.com/goark-projects/goark/core/util"
	arkerrors "github.com/goark-projects/goark/errors"
)

var (
	durationType        = reflect.TypeOf(time.Duration(0))
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

type typePair struct {
	source reflect.Type
	target reflect.Type
}

// Service 提供并发安全的类型转换能力。
type Service struct {
	mu         sync.RWMutex
	converters map[typePair]Converter
}

// NewService 创建转换服务。
func NewService(converters ...Converter) (*Service, error) {
	service := &Service{
		converters: make(map[typePair]Converter),
	}
	for _, converter := range converters {
		if err := service.Register(converter); err != nil {
			return nil, err
		}
	}
	return service, nil
}

// MustNewService 创建转换服务，失败时 panic。
func MustNewService(converters ...Converter) *Service {
	service, err := NewService(converters...)
	if err != nil {
		panic(err)
	}
	return service
}

// DefaultService 返回带默认规则的转换服务。
func DefaultService() *Service {
	return MustNewService()
}

// Register 注册转换器，已有相同源/目标类型时拒绝覆盖。
func (s *Service) Register(converter Converter) error {
	if s == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "conversion service is nil")
	}
	if converter == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "converter is nil")
	}
	pair := typePair{source: converter.SourceType(), target: converter.TargetType()}
	if pair.source == nil || pair.target == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "converter type is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.converters[pair]; exists {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "converter %s -> %s already exists", pair.source, pair.target)
	}
	s.converters[pair] = converter
	return nil
}

// CanConvert 判断当前服务是否支持源类型到目标类型的转换。
func (s *Service) CanConvert(sourceType reflect.Type, targetType reflect.Type) bool {
	if s == nil || sourceType == nil || targetType == nil {
		return false
	}
	if canAssign(sourceType, targetType) {
		return true
	}
	if s.lookup(sourceType, targetType) != nil {
		return true
	}
	return canUseBuiltin(sourceType, targetType)
}

// Convert 将输入值转换为指定目标类型。
func (s *Service) Convert(value any, targetType reflect.Type) (any, error) {
	if s == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "conversion service is nil")
	}
	if targetType == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "conversion target type is nil")
	}
	if util.IsNil(value) {
		return convertNil(targetType)
	}

	sourceType := reflect.TypeOf(value)
	if converted, ok := assignValue(value, targetType); ok {
		return converted, nil
	}
	if converter := s.lookup(sourceType, targetType); converter != nil {
		converted, err := converter.Convert(value)
		if err != nil {
			return nil, arkerrors.Wrapf(arkerrors.CodeConversion, err, "failed to convert %s to %s", sourceType, targetType)
		}
		return normalizeConverted(converted, targetType)
	}
	converted, err := builtinConvert(value, targetType, s)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeConversion, err, "failed to convert %s to %s", sourceType, targetType)
	}
	return normalizeConverted(converted, targetType)
}

// Convert 将输入值转换为泛型目标类型。
func Convert[T any](service *Service, value any) (T, error) {
	var zero T
	if service == nil {
		return zero, arkerrors.New(arkerrors.CodeInvalidArgument, "conversion service is nil")
	}
	converted, err := service.Convert(value, lang.TypeOf[T]())
	if err != nil {
		return zero, err
	}
	typed, ok := converted.(T)
	if !ok {
		return zero, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted value is %T, expected %s", converted, lang.TypeOf[T]())
	}
	return typed, nil
}

func (s *Service) lookup(sourceType reflect.Type, targetType reflect.Type) Converter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if converter := s.converters[typePair{source: sourceType, target: targetType}]; converter != nil {
		return converter
	}
	for pair, converter := range s.converters {
		if !sourceMatches(sourceType, pair.source) {
			continue
		}
		if pair.target == targetType || pair.target.AssignableTo(targetType) {
			return converter
		}
	}
	return nil
}

func sourceMatches(actual reflect.Type, expected reflect.Type) bool {
	if actual == expected || actual.AssignableTo(expected) {
		return true
	}
	return expected.Kind() == reflect.Interface && actual.Implements(expected)
}

func convertNil(targetType reflect.Type) (any, error) {
	switch targetType.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflect.Zero(targetType).Interface(), nil
	default:
		return nil, arkerrors.Newf(arkerrors.CodeConversion, "nil cannot be converted to %s", targetType)
	}
}

func assignValue(value any, targetType reflect.Type) (any, bool) {
	sourceValue := reflect.ValueOf(value)
	if !sourceValue.IsValid() {
		return nil, false
	}
	if canAssign(sourceValue.Type(), targetType) {
		if sourceValue.Type().AssignableTo(targetType) {
			return value, true
		}
		return sourceValue.Convert(targetType).Interface(), true
	}
	return nil, false
}

func canAssign(sourceType reflect.Type, targetType reflect.Type) bool {
	if sourceType.AssignableTo(targetType) {
		return true
	}
	return sourceType.ConvertibleTo(targetType) && isSafePrimitiveConversion(sourceType, targetType)
}

func isSafePrimitiveConversion(sourceType reflect.Type, targetType reflect.Type) bool {
	if sourceType.Kind() == targetType.Kind() {
		switch sourceType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64:
			return true
		}
	}
	return false
}

func canUseBuiltin(sourceType reflect.Type, targetType reflect.Type) bool {
	if targetType.Kind() == reflect.Pointer {
		return canUseBuiltin(sourceType, targetType.Elem())
	}
	if targetType.Kind() == reflect.Slice {
		return sourceType.Kind() == reflect.String || sourceType.Kind() == reflect.Slice
	}
	if textUnmarshalerTarget(targetType) {
		return sourceType.Kind() == reflect.String
	}
	switch sourceType.Kind() {
	case reflect.String:
		return isStringTarget(targetType)
	default:
		return targetType.Kind() == reflect.String
	}
}

func builtinConvert(value any, targetType reflect.Type, service *Service) (any, error) {
	if targetType.Kind() == reflect.Pointer {
		converted, err := builtinConvert(value, targetType.Elem(), service)
		if err != nil {
			return nil, err
		}
		pointer := reflect.New(targetType.Elem())
		pointer.Elem().Set(reflect.ValueOf(converted))
		return pointer.Interface(), nil
	}

	if targetType.Kind() == reflect.Slice {
		return convertSlice(value, targetType, service)
	}

	if text, ok := value.(string); ok {
		return convertString(text, targetType)
	}
	if targetType.Kind() == reflect.String {
		return fmt.Sprint(value), nil
	}
	return nil, arkerrors.Newf(arkerrors.CodeConversion, "unsupported conversion %T -> %s", value, targetType)
}

func convertString(text string, targetType reflect.Type) (any, error) {
	if textUnmarshalerTarget(targetType) {
		return unmarshalText(text, targetType)
	}
	if targetType == durationType {
		return time.ParseDuration(text)
	}
	switch targetType.Kind() {
	case reflect.String:
		return text, nil
	case reflect.Bool:
		return strconv.ParseBool(text)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(text, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetInt(parsed)
		return out.Interface(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(text, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetUint(parsed)
		return out.Interface(), nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(text, targetType.Bits())
		if err != nil {
			return nil, err
		}
		out := reflect.New(targetType).Elem()
		out.SetFloat(parsed)
		return out.Interface(), nil
	default:
		return nil, arkerrors.Newf(arkerrors.CodeConversion, "unsupported string target %s", targetType)
	}
}

func convertSlice(value any, targetType reflect.Type, service *Service) (any, error) {
	var parts []any
	if text, ok := value.(string); ok {
		rawParts := strings.Split(text, ",")
		parts = make([]any, 0, len(rawParts))
		for _, part := range rawParts {
			parts = append(parts, strings.TrimSpace(part))
		}
	} else {
		sourceValue := reflect.ValueOf(value)
		if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
			return nil, arkerrors.Newf(arkerrors.CodeConversion, "source %T is not slice-compatible", value)
		}
		parts = make([]any, 0, sourceValue.Len())
		for i := 0; i < sourceValue.Len(); i++ {
			parts = append(parts, sourceValue.Index(i).Interface())
		}
	}

	slice := reflect.MakeSlice(targetType, 0, len(parts))
	for _, part := range parts {
		converted, err := service.Convert(part, targetType.Elem())
		if err != nil {
			return nil, err
		}
		convertedValue, err := sliceElementValue(converted, targetType.Elem())
		if err != nil {
			return nil, err
		}
		slice = reflect.Append(slice, convertedValue)
	}
	return slice.Interface(), nil
}

func sliceElementValue(value any, elemType reflect.Type) (reflect.Value, error) {
	if util.IsNil(value) {
		switch elemType.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return reflect.Zero(elemType), nil
		default:
			return reflect.Value{}, arkerrors.Newf(arkerrors.CodeConversion, "nil cannot be appended to %s slice", elemType)
		}
	}
	elemValue := reflect.ValueOf(value)
	if elemValue.Type().AssignableTo(elemType) {
		return elemValue, nil
	}
	if elemValue.Type().ConvertibleTo(elemType) {
		return elemValue.Convert(elemType), nil
	}
	return reflect.Value{}, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted slice element is %T, expected %s", value, elemType)
}

func textUnmarshalerTarget(targetType reflect.Type) bool {
	if targetType.Implements(textUnmarshalerType) {
		return true
	}
	return reflect.PointerTo(targetType).Implements(textUnmarshalerType)
}

func unmarshalText(text string, targetType reflect.Type) (any, error) {
	value := reflect.New(targetType)
	unmarshaler, ok := value.Interface().(encoding.TextUnmarshaler)
	if !ok {
		if value.Elem().CanAddr() {
			unmarshaler, ok = value.Elem().Addr().Interface().(encoding.TextUnmarshaler)
		}
		if !ok {
			return nil, arkerrors.Newf(arkerrors.CodeConversion, "%s does not implement encoding.TextUnmarshaler", targetType)
		}
	}
	if err := unmarshaler.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return value.Elem().Interface(), nil
}

func isStringTarget(targetType reflect.Type) bool {
	if textUnmarshalerTarget(targetType) {
		return true
	}
	if targetType == durationType {
		return true
	}
	switch targetType.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func normalizeConverted(value any, targetType reflect.Type) (any, error) {
	if util.IsNil(value) {
		switch targetType.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return value, nil
		default:
			return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted value is nil, expected %s", targetType)
		}
	}
	converted, ok := assignValue(value, targetType)
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "converted value is %T, expected %s", value, targetType)
	}
	return converted, nil
}
