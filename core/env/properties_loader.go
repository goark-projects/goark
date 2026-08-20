package env

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	arkerrors "github.com/goark-projects/goark/errors"
)

// ParseProperties 解析 Java .properties 风格键值文本。
func ParseProperties(data []byte) (map[string]any, error) {
	lines := logicalPropertyLines(string(data))
	values := make(map[string]any, len(lines))
	for _, line := range lines {
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value := splitPropertyLine(line)
		key, err := unescapePropertyText(strings.TrimSpace(key))
		if err != nil {
			return nil, err
		}
		if key == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property key is empty")
		}
		value, err = unescapePropertyText(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, nil
}

func logicalPropertyLines(data string) []string {
	rawLines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	var builder strings.Builder
	continued := false
	for _, rawLine := range rawLines {
		line := strings.TrimSuffix(rawLine, "\r")
		if continued {
			line = strings.TrimLeftFunc(line, unicode.IsSpace)
		}
		if hasLineContinuation(line) {
			builder.WriteString(line[:len(line)-1])
			continued = true
			continue
		}
		builder.WriteString(line)
		lines = append(lines, builder.String())
		builder.Reset()
		continued = false
	}
	if builder.Len() > 0 || continued {
		lines = append(lines, builder.String())
	}
	return lines
}

func hasLineContinuation(line string) bool {
	backslashes := 0
	for i := len(line) - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func splitPropertyLine(line string) (string, string) {
	escaped := false
	for index, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '=' || r == ':' || unicode.IsSpace(r) {
			key := line[:index]
			value := strings.TrimLeftFunc(line[index+len(string(r)):], unicode.IsSpace)
			if len(value) > 0 && (value[0] == '=' || value[0] == ':') {
				value = strings.TrimLeftFunc(value[1:], unicode.IsSpace)
			}
			return key, value
		}
	}
	return line, ""
}

func unescapePropertyText(text string) (string, error) {
	var builder strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] != '\\' {
			builder.WriteByte(text[i])
			continue
		}
		i++
		if i >= len(text) {
			builder.WriteByte('\\')
			break
		}
		switch text[i] {
		case 't':
			builder.WriteByte('\t')
		case 'n':
			builder.WriteByte('\n')
		case 'r':
			builder.WriteByte('\r')
		case 'f':
			builder.WriteByte('\f')
		case '\\', ':', '=', '#', '!', ' ':
			builder.WriteByte(text[i])
		case 'u':
			r, consumed, err := parseUnicodeEscape(text[i+1:])
			if err != nil {
				return "", err
			}
			builder.WriteRune(r)
			i += consumed
		default:
			builder.WriteByte(text[i])
		}
	}
	return builder.String(), nil
}

func parseUnicodeEscape(text string) (rune, int, error) {
	if len(text) < 4 {
		return 0, 0, arkerrors.New(arkerrors.CodeInvalidArgument, "unicode escape is incomplete")
	}
	high, err := strconv.ParseUint(text[:4], 16, 16)
	if err != nil {
		return 0, 0, arkerrors.Wrap(arkerrors.CodeInvalidArgument, err, "unicode escape is invalid")
	}
	r := rune(high)
	if utf16.IsSurrogate(r) {
		if len(text) < 10 || text[4] != '\\' || text[5] != 'u' {
			return 0, 0, arkerrors.New(arkerrors.CodeInvalidArgument, "unicode surrogate pair is incomplete")
		}
		low, err := strconv.ParseUint(text[6:10], 16, 16)
		if err != nil {
			return 0, 0, arkerrors.Wrap(arkerrors.CodeInvalidArgument, err, "unicode surrogate pair is invalid")
		}
		return utf16.DecodeRune(r, rune(low)), 10, nil
	}
	return r, 4, nil
}
