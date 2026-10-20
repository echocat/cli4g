package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// Fish supplies the native process boundaries. This best-effort lexer removes
// literal quoting/escapes, retaining expressions as data without evaluating them.
func fishCompletionRequest(line string, point int) (CompletionRequest, error) {
	if point < 0 || point > len(line) {
		return CompletionRequest{}, fmt.Errorf("invalid Fish completion cursor")
	}
	var words []string
	current, offset := -1, 0
	for i := 0; i < len(line); {
		if strings.ContainsRune(";|&\n", rune(line[i])) {
			if i >= point {
				break
			}
			words, current = nil, -1
			i++
			continue
		}
		if strings.ContainsRune(" \t\r", rune(line[i])) {
			if i >= point && current < 0 {
				current, offset = len(words), 0
				words = append(words, "")
			}
			i++
			continue
		}
		start, prefix := i, 0
		var word strings.Builder
		var quote byte
		for i < len(line) {
			char := line[i]
			if quote == 0 && strings.ContainsRune(" \t\r\n;|&", rune(char)) {
				break
			}
			switch {
			case char == '\\' && i+1 < len(line) && (quote == 0 || strings.ContainsRune("\\'", rune(line[i+1])) && quote == '\'' || strings.ContainsRune("\\\"$\n", rune(line[i+1])) && quote == '"'):
				value, size := fishCompletionEscape(line[i:], quote)
				word.WriteString(value)
				i += size
			case (char == '\'' || char == '"') && (quote == 0 || quote == char):
				if quote == 0 {
					quote = char
				} else {
					quote = 0
				}
				i++
			case char == '(' && quote == 0:
				end := fishCompletionExpressionEnd(line, i)
				if point >= i && point < end {
					prefix = word.Len() + point - i
				}
				word.WriteString(line[i:end])
				i = end
			default:
				word.WriteByte(char)
				i++
			}
			if i <= point {
				prefix = word.Len()
			}
		}
		if point >= start && point <= i {
			current, offset = len(words), prefix
		}
		words = append(words, word.String())
	}
	if current < 0 {
		current, offset = len(words), 0
		words = append(words, "")
	}
	command := 0
	for command < len(words) && (words[command] == "command" || words[command] == "exec" || fishCompletionAssignment(words[command])) {
		command++
	}
	command++ // Exclude the executable as well as decorators/leading assignments.
	if current < command || command > len(words) {
		return CompletionRequest{}, fmt.Errorf("fish completion has no argument target")
	}
	return CompletionRequest{Words: words[command:], Word: current - command, Offset: offset}, nil
}

func fishCompletionEscape(input string, quote byte) (string, int) {
	char := input[1]
	if char == '\n' {
		return "", 2
	}
	if quote != 0 {
		return string(char), 2
	}
	if index := strings.IndexByte("abefnrtv", char); index >= 0 {
		return string("\a\b\x1b\f\n\r\t\v"[index]), 2
	}
	if char == 'c' && len(input) >= 3 {
		return string(input[2] & 0x1f), 3
	}
	base, begin, limit := 16, 2, 0
	switch char {
	case 'x', 'X':
		limit = 2
	case 'u':
		limit = 4
	case 'U':
		limit = 8
	default:
		if char >= '0' && char <= '7' {
			base, begin, limit = 8, 1, 3
		}
	}
	end := begin
	for end < len(input) && end < begin+limit {
		if _, err := strconv.ParseUint(input[end:end+1], base, 8); err != nil {
			break
		}
		end++
	}
	if end > begin {
		if value, err := strconv.ParseUint(input[begin:end], base, 32); err == nil {
			if char == 'x' || char == 'X' || base == 8 {
				return string([]byte{byte(value)}), end
			}
			return string(rune(value)), end
		}
	}
	return string(char), 2
}

func fishCompletionExpressionEnd(line string, start int) int {
	depth := 0
	var quote byte
	for i := start; i < len(line); i++ {
		char := line[i]
		if char == '\\' {
			i++
			continue
		}
		if char == '\'' || char == '"' {
			if quote == 0 {
				quote = char
			} else if quote == char {
				quote = 0
			}
		} else if quote == 0 {
			if char == '(' {
				depth++
			} else if char == ')' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(line)
}

func fishCompletionAssignment(word string) bool {
	name, _, found := strings.Cut(word, "=")
	if !found || name == "" {
		return false
	}
	for i, char := range name {
		if char != '_' && !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') && !(i > 0 && char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}
