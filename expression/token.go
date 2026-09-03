package expression

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind uint8

const (
	tokenEOF tokenKind = iota
	tokenIdentifier
	tokenString
	tokenNumber
	tokenTrue
	tokenFalse
	tokenNil
	tokenLeftParen
	tokenRightParen
	tokenLeftBracket
	tokenRightBracket
	tokenComma
	tokenPlus
	tokenMinus
	tokenStar
	tokenSlash
	tokenPercent
	tokenBang
	tokenEqual
	tokenNotEqual
	tokenLess
	tokenLessEqual
	tokenGreater
	tokenGreaterEqual
	tokenAnd
	tokenOr
)

type token struct {
	kind     tokenKind
	literal  string
	position int
}

type lexer struct {
	input []rune
	index int
}

func lex(input string) ([]token, error) {
	l := lexer{input: []rune(input)}
	tokens := make([]token, 0, len(l.input)/2+1)
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.kind == tokenEOF {
			return tokens, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	for l.index < len(l.input) && unicode.IsSpace(l.input[l.index]) {
		l.index++
	}
	if l.index >= len(l.input) {
		return token{kind: tokenEOF, position: l.index}, nil
	}
	start := l.index
	current := l.input[l.index]
	if unicode.IsLetter(current) || current == '_' {
		l.index++
		for l.index < len(l.input) && (unicode.IsLetter(l.input[l.index]) || unicode.IsDigit(l.input[l.index]) || l.input[l.index] == '_') {
			l.index++
		}
		literal := string(l.input[start:l.index])
		switch literal {
		case "and":
			return token{kind: tokenAnd, literal: literal, position: start}, nil
		case "or":
			return token{kind: tokenOr, literal: literal, position: start}, nil
		case "not":
			return token{kind: tokenBang, literal: literal, position: start}, nil
		case "eq":
			return token{kind: tokenEqual, literal: literal, position: start}, nil
		case "ne":
			return token{kind: tokenNotEqual, literal: literal, position: start}, nil
		case "lt":
			return token{kind: tokenLess, literal: literal, position: start}, nil
		case "le":
			return token{kind: tokenLessEqual, literal: literal, position: start}, nil
		case "gt":
			return token{kind: tokenGreater, literal: literal, position: start}, nil
		case "ge":
			return token{kind: tokenGreaterEqual, literal: literal, position: start}, nil
		case "true":
			return token{kind: tokenTrue, literal: literal, position: start}, nil
		case "false":
			return token{kind: tokenFalse, literal: literal, position: start}, nil
		case "nil", "null":
			return token{kind: tokenNil, literal: literal, position: start}, nil
		default:
			return token{kind: tokenIdentifier, literal: literal, position: start}, nil
		}
	}
	if unicode.IsDigit(current) {
		return l.number(start)
	}
	if current == '\'' || current == '"' {
		return l.stringValue(start, current)
	}
	l.index++
	single := func(kind tokenKind) (token, error) {
		return token{kind: kind, literal: string(current), position: start}, nil
	}
	switch current {
	case '(':
		return single(tokenLeftParen)
	case ')':
		return single(tokenRightParen)
	case '[':
		return single(tokenLeftBracket)
	case ']':
		return single(tokenRightBracket)
	case ',':
		return single(tokenComma)
	case '+':
		return single(tokenPlus)
	case '-':
		return single(tokenMinus)
	case '*':
		return single(tokenStar)
	case '/':
		return single(tokenSlash)
	case '%':
		return single(tokenPercent)
	case '!':
		if l.take('=') {
			return token{kind: tokenNotEqual, literal: "!=", position: start}, nil
		}
		return single(tokenBang)
	case '=':
		if l.take('=') {
			return token{kind: tokenEqual, literal: "==", position: start}, nil
		}
	case '<':
		if l.take('=') {
			return token{kind: tokenLessEqual, literal: "<=", position: start}, nil
		}
		return single(tokenLess)
	case '>':
		if l.take('=') {
			return token{kind: tokenGreaterEqual, literal: ">=", position: start}, nil
		}
		return single(tokenGreater)
	case '&':
		if l.take('&') {
			return token{kind: tokenAnd, literal: "&&", position: start}, nil
		}
	case '|':
		if l.take('|') {
			return token{kind: tokenOr, literal: "||", position: start}, nil
		}
	}
	return token{}, fmt.Errorf("unexpected character %q at position %d", current, start)
}

func (l *lexer) take(expected rune) bool {
	if l.index >= len(l.input) || l.input[l.index] != expected {
		return false
	}
	l.index++
	return true
}

func (l *lexer) number(start int) (token, error) {
	hasDot := false
	for l.index < len(l.input) {
		current := l.input[l.index]
		if unicode.IsDigit(current) {
			l.index++
			continue
		}
		if current == '.' && !hasDot {
			hasDot = true
			l.index++
			continue
		}
		break
	}
	literal := string(l.input[start:l.index])
	if strings.HasSuffix(literal, ".") {
		return token{}, fmt.Errorf("invalid number %q at position %d", literal, start)
	}
	return token{kind: tokenNumber, literal: literal, position: start}, nil
}

func (l *lexer) stringValue(start int, quote rune) (token, error) {
	l.index++
	var builder strings.Builder
	for l.index < len(l.input) {
		current := l.input[l.index]
		l.index++
		if current == quote {
			return token{kind: tokenString, literal: builder.String(), position: start}, nil
		}
		if current != '\\' {
			builder.WriteRune(current)
			continue
		}
		if l.index >= len(l.input) {
			break
		}
		escaped := l.input[l.index]
		l.index++
		decoded, err := strconv.Unquote(`"\\` + string(escaped) + `"`)
		if err != nil {
			return token{}, fmt.Errorf("invalid string escape at position %d: %w", l.index-2, err)
		}
		builder.WriteString(decoded)
	}
	return token{}, fmt.Errorf("unclosed string at position %d", start)
}
