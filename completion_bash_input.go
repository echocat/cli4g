package cli

import (
	"fmt"
	"strings"
)

// Bash owns expansion and execution. This lexer only removes simple quoting and
// identifies the simple command surrounding the cursor; it never evaluates code.
func bashCompletionRequest(line string, point int) (CompletionRequest, error) {
	if point < 0 || point > len(line) {
		return CompletionRequest{}, fmt.Errorf("invalid Bash completion cursor")
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
		if line[i] == ' ' || line[i] == '\t' || line[i] == '\r' {
			if i >= point && current < 0 {
				current, offset = len(words), 0
				words = append(words, "")
			}
			i++
			continue
		}
		start := i
		var word strings.Builder
		var quote byte
		prefix := 0
		for i < len(line) {
			char := line[i]
			if quote == 0 && strings.ContainsRune(" \t\r\n;|&", rune(char)) {
				break
			}
			if char == '\\' && quote != '\'' && i+1 < len(line) && (quote == 0 || strings.ContainsRune("\\\"$`\n", rune(line[i+1]))) {
				i++
				if line[i] != '\n' {
					word.WriteByte(line[i])
				}
			} else if (char == '\'' || char == '"') && (quote == 0 || quote == char) {
				if quote == 0 {
					quote = char
				} else {
					quote = 0
				}
			} else {
				word.WriteByte(char)
			}
			i++
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
	if current == 0 || len(words) == 0 {
		return CompletionRequest{}, fmt.Errorf("bash completion has no argument target")
	}
	return CompletionRequest{Words: words[1:], Word: current - 1, Offset: offset}, nil
}
