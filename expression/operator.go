package expression

import (
	"fmt"
	"math"
	"reflect"

	arkerrors "goark.dev/goark/errors"
)

func evaluateBinary(operator tokenKind, left any, right any) (any, error) {
	switch operator {
	case tokenEqual:
		return equalValues(left, right), nil
	case tokenNotEqual:
		return !equalValues(left, right), nil
	case tokenAnd, tokenOr:
		rightBoolean, ok := right.(bool)
		if !ok {
			return nil, typeError("logical operator", right)
		}
		return rightBoolean, nil
	case tokenPlus:
		if leftString, ok := left.(string); ok {
			return leftString + fmt.Sprint(right), nil
		}
		if rightString, ok := right.(string); ok {
			return fmt.Sprint(left) + rightString, nil
		}
	}
	leftNumber, leftErr := numeric(left)
	rightNumber, rightErr := numeric(right)
	if leftErr != nil || rightErr != nil {
		return nil, arkerrors.Newf(arkerrors.CodeTypeMismatch, "GaEL operator requires compatible operands, got %T and %T", left, right)
	}
	return evaluateNumeric(operator, leftNumber, rightNumber)
}

func evaluateNumeric(operator tokenKind, left any, right any) (any, error) {
	leftFloat, leftInteger := numberAsFloat(left)
	rightFloat, rightInteger := numberAsFloat(right)
	switch operator {
	case tokenLess:
		return leftFloat < rightFloat, nil
	case tokenLessEqual:
		return leftFloat <= rightFloat, nil
	case tokenGreater:
		return leftFloat > rightFloat, nil
	case tokenGreaterEqual:
		return leftFloat >= rightFloat, nil
	case tokenPlus:
		if leftInteger && rightInteger {
			return left.(int64) + right.(int64), nil
		}
		return leftFloat + rightFloat, nil
	case tokenMinus:
		if leftInteger && rightInteger {
			return left.(int64) - right.(int64), nil
		}
		return leftFloat - rightFloat, nil
	case tokenStar:
		if leftInteger && rightInteger {
			return left.(int64) * right.(int64), nil
		}
		return leftFloat * rightFloat, nil
	case tokenSlash:
		if rightFloat == 0 {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL division by zero")
		}
		return leftFloat / rightFloat, nil
	case tokenPercent:
		if !leftInteger || !rightInteger || right.(int64) == 0 {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL remainder requires non-zero integers")
		}
		return left.(int64) % right.(int64), nil
	default:
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "unsupported GaEL binary operator")
	}
}

func numeric(value any) (any, error) {
	reflection := reflect.ValueOf(value)
	if !reflection.IsValid() {
		return nil, typeError("numeric operator", value)
	}
	switch reflection.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflection.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		unsigned := reflection.Uint()
		if unsigned > math.MaxInt64 {
			return nil, arkerrors.New(arkerrors.CodeConversion, "GaEL unsigned integer overflows int64")
		}
		return int64(unsigned), nil
	case reflect.Float32, reflect.Float64:
		return reflection.Float(), nil
	default:
		return nil, typeError("numeric operator", value)
	}
}

func numberAsFloat(value any) (float64, bool) {
	if integer, ok := value.(int64); ok {
		return float64(integer), true
	}
	return value.(float64), false
}

func equalValues(left any, right any) bool {
	leftNumber, leftErr := numeric(left)
	rightNumber, rightErr := numeric(right)
	if leftErr == nil && rightErr == nil {
		leftFloat, _ := numberAsFloat(leftNumber)
		rightFloat, _ := numberAsFloat(rightNumber)
		return leftFloat == rightFloat
	}
	return reflect.DeepEqual(left, right)
}

func typeError(operation string, value any) error {
	return arkerrors.Newf(arkerrors.CodeTypeMismatch, "GaEL %s does not support %T", operation, value)
}
