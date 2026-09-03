package expression_test

import (
	"context"
	"testing"

	"goark.dev/goark/core/env"
	"goark.dev/goark/expression"
)

func TestExpression_whenUsingPropertiesVariablesAndOperators_shouldEvaluate(t *testing.T) {
	environment := env.MustNewStandardEnvironment()
	source, err := env.NewMapPropertySource("test", map[string]any{"server.port": "8080"})
	if err != nil {
		t.Fatalf("NewMapPropertySource() error = %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("AddFirst() error = %v", err)
	}
	evaluationContext, err := expression.NewEvaluationContext(environment, expression.WithVariable("minimum", int64(8000)))
	if err != nil {
		t.Fatalf("NewEvaluationContext() error = %v", err)
	}
	parsed, err := expression.NewParser().Parse(`environment['server.port'] == '8080' && minimum + 80 == 8080`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	value, err := parsed.Evaluate(context.Background(), evaluationContext)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if value != true {
		t.Fatalf("Evaluate() = %v, want true", value)
	}
}

func TestExpression_whenFunctionIsNotRegistered_shouldRejectInvocation(t *testing.T) {
	evaluationContext, err := expression.NewEvaluationContext(env.MustNewStandardEnvironment())
	if err != nil {
		t.Fatalf("NewEvaluationContext() error = %v", err)
	}
	parsed, err := expression.NewParser().Parse(`execute('command')`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := parsed.Evaluate(context.Background(), evaluationContext); err == nil {
		t.Fatal("Evaluate() error = nil, want unregistered function error")
	}
}

func TestExpression_whenLogicalLeftDeterminesResult_shouldShortCircuit(t *testing.T) {
	evaluationContext, err := expression.NewEvaluationContext(env.MustNewStandardEnvironment())
	if err != nil {
		t.Fatalf("NewEvaluationContext() error = %v", err)
	}
	parsed, err := expression.NewParser().Parse(`true || missing()`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	value, err := parsed.Evaluate(context.Background(), evaluationContext)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if value != true {
		t.Fatalf("Evaluate() = %v, want true", value)
	}
}

func TestExpression_whenFunctionIsRegistered_shouldInvokeIt(t *testing.T) {
	evaluationContext, err := expression.NewEvaluationContext(
		env.MustNewStandardEnvironment(),
		expression.WithFunction("double", func(_ context.Context, arguments ...any) (any, error) {
			return arguments[0].(int64) * 2, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewEvaluationContext() error = %v", err)
	}
	parsed, err := expression.NewParser().Parse(`double(21) == 42`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	value, err := parsed.Evaluate(context.Background(), evaluationContext)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if value != true {
		t.Fatalf("Evaluate() = %v, want true", value)
	}
}
