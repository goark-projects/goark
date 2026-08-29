package webtest

import (
	"strconv"
	"strings"

	arkjson "goark.dev/arkarta/json"
)

const (
	jsonPathKey jsonPathTokenKind = iota
	jsonPathIndex
)

type jsonPathTokenKind uint8

type jsonPathToken struct {
	kind  jsonPathTokenKind
	key   string
	index int
}

func resolveJSONPath(codec arkjson.Codec, body []byte, expression string) (any, bool, error) {
	expression = strings.TrimSpace(expression)
	tokens, err := parseJSONPath(expression)
	if err != nil {
		return nil, false, err
	}
	var value any
	if err := arkjson.Unmarshal(codec, body, &value); err != nil {
		return nil, false, err
	}
	for _, token := range tokens {
		next, ok := resolveJSONPathToken(value, token)
		if !ok {
			return nil, false, nil
		}
		value = next
	}
	return value, true, nil
}

func parseJSONPath(expression string) ([]jsonPathToken, error) {
	if expression == "" || expression[0] != '$' {
		return nil, ErrInvalidJSONPath
	}
	tokens := make([]jsonPathToken, 0, 4)
	for offset := 1; offset < len(expression); {
		switch expression[offset] {
		case '.':
			token, next, err := parseJSONPathField(expression, offset+1)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			offset = next
		case '[':
			token, next, err := parseJSONPathBracket(expression, offset+1)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			offset = next
		default:
			return nil, ErrInvalidJSONPath
		}
	}
	return tokens, nil
}

func parseJSONPathField(expression string, offset int) (jsonPathToken, int, error) {
	start := offset
	for offset < len(expression) && expression[offset] != '.' && expression[offset] != '[' {
		offset++
	}
	if start == offset {
		return jsonPathToken{}, 0, ErrInvalidJSONPath
	}
	return jsonPathToken{kind: jsonPathKey, key: expression[start:offset]}, offset, nil
}

func parseJSONPathBracket(expression string, offset int) (jsonPathToken, int, error) {
	if offset >= len(expression) {
		return jsonPathToken{}, 0, ErrInvalidJSONPath
	}
	switch expression[offset] {
	case '\'', '"':
		return parseJSONPathQuotedKey(expression, offset)
	default:
		return parseJSONPathIndex(expression, offset)
	}
}

func parseJSONPathQuotedKey(expression string, offset int) (jsonPathToken, int, error) {
	quote := expression[offset]
	var builder strings.Builder
	for offset++; offset < len(expression); offset++ {
		c := expression[offset]
		if c == quote {
			if offset+1 >= len(expression) || expression[offset+1] != ']' {
				return jsonPathToken{}, 0, ErrInvalidJSONPath
			}
			return jsonPathToken{kind: jsonPathKey, key: builder.String()}, offset + 2, nil
		}
		if c == '\\' {
			offset++
			if offset >= len(expression) {
				return jsonPathToken{}, 0, ErrInvalidJSONPath
			}
			c = expression[offset]
		}
		builder.WriteByte(c)
	}
	return jsonPathToken{}, 0, ErrInvalidJSONPath
}

func parseJSONPathIndex(expression string, offset int) (jsonPathToken, int, error) {
	start := offset
	for offset < len(expression) && expression[offset] != ']' {
		offset++
	}
	if start == offset || offset >= len(expression) {
		return jsonPathToken{}, 0, ErrInvalidJSONPath
	}
	index, err := strconv.Atoi(expression[start:offset])
	if err != nil || index < 0 {
		return jsonPathToken{}, 0, ErrInvalidJSONPath
	}
	return jsonPathToken{kind: jsonPathIndex, index: index}, offset + 1, nil
}

func resolveJSONPathToken(value any, token jsonPathToken) (any, bool) {
	switch token.kind {
	case jsonPathKey:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := object[token.key]
		return next, ok
	case jsonPathIndex:
		items, ok := value.([]any)
		if !ok || token.index >= len(items) {
			return nil, false
		}
		return items[token.index], true
	default:
		return nil, false
	}
}

func normalizeJSONValue(codec arkjson.Codec, value any) (any, error) {
	data, err := arkjson.Marshal(codec, value)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := arkjson.Unmarshal(codec, data, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}
