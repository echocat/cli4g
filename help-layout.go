package cli

import (
	"bytes"
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

const helpColumnPadding = 2

type helpLayoutRow struct {
	cells  []string
	indent int
}

type helpLayoutKey struct {
	indent, columns int
}

type helpLayoutColumns struct {
	widths []int
	end    int
}

// Format the template's text rather than interpreting its template syntax or
// metadata. Tab-separated runs are tables; different indentation levels have
// independent columns (e.g. enum choices nested within a flag table).
func formatHelp(ctx context.Context, input []byte, width int) ([]byte, error) {
	if width <= 0 {
		width = defaultHelpWidth
	}
	var output bytes.Buffer
	output.Grow(len(input))
	lines := strings.Split(string(input), "\n")
	for i := 0; i < len(lines); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.ContainsRune(lines[i], '\t') {
			line := strings.TrimSuffix(lines[i], "\r")
			indent := line[:helpIndent(line)]
			output.WriteString(indent)
			writeHelpText(&output, line[len(indent):], len(indent), indent, width)
			i++
		} else {
			end := i + 1
			for end < len(lines) && strings.ContainsRune(lines[end], '\t') {
				end++
			}
			if err := writeHelpTable(ctx, &output, lines[i:end], width); err != nil {
				return nil, err
			}
			i = end
		}
		if i < len(lines) {
			output.WriteByte('\n')
		}
	}
	return output.Bytes(), ctx.Err()
}

func helpIndent(text string) int {
	return len(text) - len(strings.TrimLeft(text, " "))
}

func writeHelpTable(ctx context.Context, output *bytes.Buffer, lines []string, width int) error {
	rows := make([]helpLayoutRow, len(lines))
	columns := make(map[helpLayoutKey]*helpLayoutColumns)
	for i, line := range lines {
		if err := ctx.Err(); err != nil {
			return err
		}
		cells := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		row := helpLayoutRow{cells: cells, indent: helpIndent(cells[0])}
		rows[i] = row
		key := helpLayoutKey{row.indent, len(cells)}
		group := columns[key]
		if group == nil {
			group = &helpLayoutColumns{widths: make([]int, len(cells)-1)}
			columns[key] = group
		}
		// Reserve at least half the width for descriptions. Oversized labels
		// get their own line and must not widen every other row in the table.
		prefixWidth := helpColumnPadding * (len(cells) - 1)
		for _, cell := range cells[:len(cells)-1] {
			prefixWidth += utf8.RuneCountInString(cell)
		}
		if prefixWidth <= width/2 {
			for j, cell := range cells[:len(cells)-1] {
				group.widths[j] = max(group.widths[j], utf8.RuneCountInString(cell))
			}
		}
	}
	for key, group := range columns {
		for _, w := range group.widths {
			group.end += w + helpColumnPadding
		}
		group.end = max(key.indent+helpColumnPadding, min(group.end, width/2))
	}
	for i, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if i > 0 {
			output.WriteByte('\n')
		}
		writeHelpRow(output, row, columns[helpLayoutKey{row.indent, len(row.cells)}], width)
	}
	return nil
}

func writeHelpRow(output *bytes.Buffer, row helpLayoutRow, columns *helpLayoutColumns, width int) {
	position, next := 0, 0
	for i, cell := range row.cells[:len(row.cells)-1] {
		if i > 0 {
			padding := max(helpColumnPadding, next-position)
			output.WriteString(strings.Repeat(" ", padding))
			position += padding
		}
		output.WriteString(cell)
		position += utf8.RuneCountInString(cell)
		next += columns.widths[i] + helpColumnPadding
	}
	text := strings.TrimSpace(row.cells[len(row.cells)-1])
	if text == "" {
		return
	}
	indent := columns.end
	if position+helpColumnPadding > indent {
		output.WriteByte('\n')
		indent = row.indent + helpColumnPadding
		position = 0
	}
	padding := strings.Repeat(" ", max(0, indent-position))
	output.WriteString(padding)
	writeHelpText(output, text, indent, strings.Repeat(" ", indent), width)
}

// Preserve whitespace between words unless a wrap replaces it. Every source
// newline remains a hard break. Long indivisible words are never truncated or
// split, which keeps paths, placeholders and identifiers usable for copying.
func writeHelpText(output *bytes.Buffer, text string, position int, indent string, width int) {
	hasWord := false
	for text != "" {
		start := strings.IndexFunc(text, func(r rune) bool { return !unicode.IsSpace(r) })
		if start < 0 {
			return
		}
		spaces := text[:start]
		text = text[start:]
		end := strings.IndexFunc(text, unicode.IsSpace)
		if end < 0 {
			end = len(text)
		}
		word := text[:end]
		wordWidth := utf8.RuneCountInString(word)
		spaceWidth := utf8.RuneCountInString(spaces)
		if hasWord && position+spaceWidth+wordWidth > width {
			output.WriteByte('\n')
			output.WriteString(indent)
			position = len(indent)
			spaces, spaceWidth = "", 0
		}
		output.WriteString(spaces)
		output.WriteString(word)
		position += spaceWidth + wordWidth
		hasWord = true
		text = text[end:]
	}
}
