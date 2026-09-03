package expression

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Expression 是已解析且可重复并发求值的表达式。
type Expression interface {
	Evaluate(context.Context, EvaluationContext) (any, error)
}

// Parser 定义 GaEL 解析契约。
type Parser interface {
	Parse(text string) (Expression, error)
}

// StandardParser 是无状态的默认 GaEL 解析器。
type StandardParser struct{}

// NewParser 创建默认解析器。
func NewParser() Parser {
	return StandardParser{}
}

// Parse 将表达式文本解析为不可变语法树。
func (StandardParser) Parse(text string) (Expression, error) {
	tokens, err := lex(strings.TrimSpace(text))
	if err != nil {
		return nil, err
	}
	state := parserState{tokens: tokens}
	expression, err := state.parseOr()
	if err != nil {
		return nil, err
	}
	if current := state.current(); current.kind != tokenEOF {
		return nil, fmt.Errorf("unexpected token %q at position %d", current.literal, current.position)
	}
	return expression, nil
}

type parserState struct {
	tokens []token
	index  int
}

func (p *parserState) parseOr() (node, error) {
	return p.parseBinary(p.parseAnd, tokenOr)
}

func (p *parserState) parseAnd() (node, error) {
	return p.parseBinary(p.parseEquality, tokenAnd)
}

func (p *parserState) parseEquality() (node, error) {
	return p.parseBinary(p.parseComparison, tokenEqual, tokenNotEqual)
}

func (p *parserState) parseComparison() (node, error) {
	return p.parseBinary(p.parseTerm, tokenLess, tokenLessEqual, tokenGreater, tokenGreaterEqual)
}

func (p *parserState) parseTerm() (node, error) {
	return p.parseBinary(p.parseFactor, tokenPlus, tokenMinus)
}

func (p *parserState) parseFactor() (node, error) {
	return p.parseBinary(p.parseUnary, tokenStar, tokenSlash, tokenPercent)
}

func (p *parserState) parseBinary(next func() (node, error), kinds ...tokenKind) (node, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for containsKind(kinds, p.current().kind) {
		operator := p.advance()
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = binaryNode{operator: operator.kind, left: left, right: right}
	}
	return left, nil
}

func (p *parserState) parseUnary() (node, error) {
	if current := p.current(); current.kind == tokenBang || current.kind == tokenMinus || current.kind == tokenPlus {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{operator: current.kind, operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *parserState) parsePostfix() (node, error) {
	value, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenLeftBracket {
		p.advance()
		index, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRightBracket); err != nil {
			return nil, err
		}
		value = indexNode{target: value, index: index}
	}
	return value, nil
}

func (p *parserState) parsePrimary() (node, error) {
	current := p.advance()
	switch current.kind {
	case tokenString:
		return literalNode{value: current.literal}, nil
	case tokenNumber:
		if strings.Contains(current.literal, ".") {
			value, err := strconv.ParseFloat(current.literal, 64)
			return literalNode{value: value}, err
		}
		value, err := strconv.ParseInt(current.literal, 10, 64)
		return literalNode{value: value}, err
	case tokenTrue:
		return literalNode{value: true}, nil
	case tokenFalse:
		return literalNode{value: false}, nil
	case tokenNil:
		return literalNode{}, nil
	case tokenIdentifier:
		if p.current().kind != tokenLeftParen {
			return identifierNode{name: current.literal}, nil
		}
		p.advance()
		arguments := make([]node, 0, 2)
		if p.current().kind != tokenRightParen {
			for {
				argument, err := p.parseOr()
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, argument)
				if p.current().kind != tokenComma {
					break
				}
				p.advance()
			}
		}
		if _, err := p.expect(tokenRightParen); err != nil {
			return nil, err
		}
		return callNode{name: current.literal, arguments: arguments}, nil
	case tokenLeftParen:
		expression, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRightParen); err != nil {
			return nil, err
		}
		return expression, nil
	default:
		return nil, fmt.Errorf("expected expression at position %d", current.position)
	}
}

func (p *parserState) current() token {
	if p.index >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.index]
}

func (p *parserState) advance() token {
	current := p.current()
	if p.index < len(p.tokens) {
		p.index++
	}
	return current
}

func (p *parserState) expect(kind tokenKind) (token, error) {
	current := p.current()
	if current.kind != kind {
		return token{}, fmt.Errorf("unexpected token %q at position %d", current.literal, current.position)
	}
	p.index++
	return current, nil
}

func containsKind(kinds []tokenKind, target tokenKind) bool {
	for _, kind := range kinds {
		if kind == target {
			return true
		}
	}
	return false
}
