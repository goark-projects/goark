package env

import (
	"strings"
	"unicode"

	arkerrors "goark.dev/goark/errors"
)

// MatchProfileExpression 判断 profile 表达式是否匹配当前环境。
func MatchProfileExpression(environment Environment, expression string) (bool, error) {
	if environment == nil {
		return false, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	parser := newProfileExpressionParser(expression)
	node, err := parser.parse()
	if err != nil {
		return false, err
	}
	return node.match(environment), nil
}

type profileExpressionNode interface {
	match(Environment) bool
}

type profileNameNode struct {
	name string
}

func (n profileNameNode) match(environment Environment) bool {
	return environment.AcceptsProfiles(n.name)
}

type profileNotNode struct {
	child profileExpressionNode
}

func (n profileNotNode) match(environment Environment) bool {
	return !n.child.match(environment)
}

type profileAndNode struct {
	left  profileExpressionNode
	right profileExpressionNode
}

func (n profileAndNode) match(environment Environment) bool {
	return n.left.match(environment) && n.right.match(environment)
}

type profileOrNode struct {
	left  profileExpressionNode
	right profileExpressionNode
}

func (n profileOrNode) match(environment Environment) bool {
	return n.left.match(environment) || n.right.match(environment)
}

type profileExpressionParser struct {
	input string
	pos   int
}

func newProfileExpressionParser(expression string) *profileExpressionParser {
	return &profileExpressionParser{input: strings.TrimSpace(expression)}
}

func (p *profileExpressionParser) parse() (profileExpressionNode, error) {
	if p.input == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "profile expression is empty")
	}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if !p.eof() {
		return nil, p.errorf("unexpected token %q", p.peek())
	}
	return node, nil
}

func (p *profileExpressionParser) parseOr() (profileExpressionNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if !p.consume('|') {
			return left, nil
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = profileOrNode{left: left, right: right}
	}
}

func (p *profileExpressionParser) parseAnd() (profileExpressionNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if !p.consume('&') {
			return left, nil
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = profileAndNode{left: left, right: right}
	}
}

func (p *profileExpressionParser) parseUnary() (profileExpressionNode, error) {
	p.skipSpaces()
	if p.consume('!') {
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return profileNotNode{child: child}, nil
	}
	return p.parsePrimary()
}

func (p *profileExpressionParser) parsePrimary() (profileExpressionNode, error) {
	p.skipSpaces()
	if p.consume('(') {
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if !p.consume(')') {
			return nil, p.errorf("missing closing profile expression parenthesis")
		}
		return node, nil
	}
	name := p.parseName()
	if name == "" {
		if p.eof() {
			return nil, p.errorf("unexpected end of profile expression")
		}
		return nil, p.errorf("unexpected token %q", p.peek())
	}
	return profileNameNode{name: name}, nil
}

func (p *profileExpressionParser) parseName() string {
	start := p.pos
	for !p.eof() {
		r := p.peek()
		if unicode.IsSpace(r) || r == '!' || r == '&' || r == '|' || r == '(' || r == ')' {
			break
		}
		p.pos += len(string(r))
	}
	return strings.TrimSpace(p.input[start:p.pos])
}

func (p *profileExpressionParser) skipSpaces() {
	for !p.eof() && unicode.IsSpace(p.peek()) {
		p.pos += len(string(p.peek()))
	}
}

func (p *profileExpressionParser) consume(expected rune) bool {
	if p.eof() || p.peek() != expected {
		return false
	}
	p.pos += len(string(expected))
	return true
}

func (p *profileExpressionParser) eof() bool {
	return p.pos >= len(p.input)
}

func (p *profileExpressionParser) peek() rune {
	for _, r := range p.input[p.pos:] {
		return r
	}
	return 0
}

func (p *profileExpressionParser) errorf(format string, args ...any) error {
	return arkerrors.Newf(arkerrors.CodeInvalidArgument, "invalid profile expression %q at offset %d: "+format, append([]any{p.input, p.pos}, args...)...)
}
