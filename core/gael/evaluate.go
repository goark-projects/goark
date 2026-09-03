package gael

import (
	"context"

	arkerrors "goark.dev/goark/errors"
)

type node interface {
	Expression
}

type literalNode struct{ value any }
type identifierNode struct{ name string }
type unaryNode struct {
	operator tokenKind
	operand  node
}
type binaryNode struct {
	operator tokenKind
	left     node
	right    node
}
type callNode struct {
	name      string
	arguments []node
}
type indexNode struct {
	target node
	index  node
}

func (n literalNode) Evaluate(context.Context, EvaluationContext) (any, error) { return n.value, nil }

func (n identifierNode) Evaluate(_ context.Context, evaluationContext EvaluationContext) (any, error) {
	if evaluationContext == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL evaluation context is nil")
	}
	if n.name == "environment" || n.name == "properties" {
		return evaluationContext.Properties(), nil
	}
	value, ok := evaluationContext.Variable(n.name)
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "GaEL variable %q not found", n.name)
	}
	return value, nil
}

func (n unaryNode) Evaluate(ctx context.Context, evaluationContext EvaluationContext) (any, error) {
	value, err := n.operand.Evaluate(ctx, evaluationContext)
	if err != nil {
		return nil, err
	}
	switch n.operator {
	case tokenBang:
		boolean, ok := value.(bool)
		if !ok {
			return nil, typeError("!", value)
		}
		return !boolean, nil
	case tokenPlus:
		return numeric(value)
	case tokenMinus:
		number, err := numeric(value)
		if err != nil {
			return nil, err
		}
		if integer, ok := number.(int64); ok {
			return -integer, nil
		}
		return -number.(float64), nil
	default:
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "unsupported GaEL unary operator")
	}
}

func (n binaryNode) Evaluate(ctx context.Context, evaluationContext EvaluationContext) (any, error) {
	left, err := n.left.Evaluate(ctx, evaluationContext)
	if err != nil {
		return nil, err
	}
	if n.operator == tokenAnd {
		boolean, ok := left.(bool)
		if !ok {
			return nil, typeError("&&", left)
		}
		if !boolean {
			return false, nil
		}
	}
	if n.operator == tokenOr {
		boolean, ok := left.(bool)
		if !ok {
			return nil, typeError("||", left)
		}
		if boolean {
			return true, nil
		}
	}
	right, err := n.right.Evaluate(ctx, evaluationContext)
	if err != nil {
		return nil, err
	}
	return evaluateBinary(n.operator, left, right)
}

func (n callNode) Evaluate(ctx context.Context, evaluationContext EvaluationContext) (any, error) {
	if evaluationContext == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL evaluation context is nil")
	}
	arguments := make([]any, len(n.arguments))
	for index, argument := range n.arguments {
		value, err := argument.Evaluate(ctx, evaluationContext)
		if err != nil {
			return nil, err
		}
		arguments[index] = value
	}
	if n.name == "property" || n.name == "hasProperty" {
		return evaluatePropertyCall(evaluationContext, n.name, arguments)
	}
	function, ok := evaluationContext.Function(n.name)
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "GaEL function %q not found", n.name)
	}
	return function(ctx, arguments...)
}

func (n indexNode) Evaluate(ctx context.Context, evaluationContext EvaluationContext) (any, error) {
	target, err := n.target.Evaluate(ctx, evaluationContext)
	if err != nil {
		return nil, err
	}
	index, err := n.index.Evaluate(ctx, evaluationContext)
	if err != nil {
		return nil, err
	}
	if properties, ok := target.(PropertyResolver); ok {
		key, ok := index.(string)
		if !ok {
			return nil, typeError("property index", index)
		}
		value, found := properties.GetProperty(key)
		if !found {
			return nil, arkerrors.Newf(arkerrors.CodeNotFound, "GaEL property %q not found", key)
		}
		return value, nil
	}
	return indexedValue(target, index)
}

func evaluatePropertyCall(evaluationContext EvaluationContext, name string, arguments []any) (any, error) {
	if len(arguments) != 1 {
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "GaEL %s requires one argument", name)
	}
	key, ok := arguments[0].(string)
	if !ok {
		return nil, typeError(name, arguments[0])
	}
	properties := evaluationContext.Properties()
	if properties == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL property resolver is nil")
	}
	if name == "hasProperty" {
		return properties.ContainsProperty(key), nil
	}
	value, ok := properties.GetProperty(key)
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "GaEL property %q not found", key)
	}
	return value, nil
}
