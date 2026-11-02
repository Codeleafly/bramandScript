package diagnostics

import (
	"fmt"
	"strings"
)

// ANSI Color Codes
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorRed     = "\033[1;31m"
	ColorGreen   = "\033[1;32m"
	ColorYellow  = "\033[1;33m"
	ColorBlue    = "\033[1;34m"
	ColorMagenta = "\033[1;35m"
	ColorCyan    = "\033[1;36m"
	ColorWhite   = "\033[1;37m"
)

type Diagnostic struct {
	Filename string
	Source   string
	Line     int
	Col      int
	Kind     string // "SyntaxError", "CompileError", "RuntimeError"
	Message  string
	Hint     string
}

// Format renders a compiler diagnostic with 5 lines of context above and 5 below
// with caret pointer, column highlight, and zero box borders.
func (d *Diagnostic) Format() string {
	var out strings.Builder

	// Header: error[Kind]: Message
	out.WriteString(fmt.Sprintf("%s%serror[%s]%s: %s%s%s\n",
		ColorBold, ColorRed, d.Kind, ColorReset,
		ColorBold, d.Message, ColorReset))

	// Location link: --> filename:line:col
	out.WriteString(fmt.Sprintf("  %s-->%s %s:%d:%d\n",
		ColorCyan, ColorReset, d.Filename, d.Line, d.Col))

	if d.Source == "" {
		return out.String()
	}

	lines := strings.Split(d.Source, "\n")
	totalLines := len(lines)

	if d.Line < 1 || d.Line > totalLines {
		return out.String()
	}

	// 5 lines before and 5 lines after
	startLine := d.Line - 5
	if startLine < 1 {
		startLine = 1
	}

	endLine := d.Line + 5
	if endLine > totalLines {
		endLine = totalLines
	}

	// Calculate gutter width for line numbers
	gutterWidth := len(fmt.Sprintf("%d", endLine))
	if gutterWidth < 2 {
		gutterWidth = 2
	}

	out.WriteString(fmt.Sprintf("  %s%s%s\n", ColorCyan, strings.Repeat(" ", gutterWidth)+" |", ColorReset))

	for l := startLine; l <= endLine; l++ {
		lineIdx := l - 1
		lineText := lines[lineIdx]
		// Convert tabs to 4 spaces for consistent visual alignment
		sanitizedLine := strings.ReplaceAll(lineText, "\t", "    ")

		if l == d.Line {
			// The exact error line: highlight in white/bold
			lineNumStr := fmt.Sprintf("%*d", gutterWidth, l)
			out.WriteString(fmt.Sprintf("  %s%s |%s %s%s%s\n",
				ColorCyan, lineNumStr, ColorReset,
				ColorBold, sanitizedLine, ColorReset))

			// Caret underline pointer
			caretCol := d.Col
			if caretCol < 1 {
				caretCol = 1
			}
			if caretCol > len(sanitizedLine)+1 {
				caretCol = len(sanitizedLine) + 1
			}

			indent := strings.Repeat(" ", caretCol-1)
			out.WriteString(fmt.Sprintf("  %s%s |%s %s%s^%s",
				ColorCyan, strings.Repeat(" ", gutterWidth), ColorReset,
				indent, ColorRed, ColorReset))

			// Underline additional characters if token word detected
			wordLen := getWordLength(sanitizedLine, caretCol-1)
			if wordLen > 1 {
				out.WriteString(fmt.Sprintf("%s%s%s", ColorRed, strings.Repeat("~", wordLen-1), ColorReset))
			}

			// Inline hint or note
			if d.Hint != "" {
				out.WriteString(fmt.Sprintf(" %s%s[Hint: %s]%s", ColorYellow, ColorBold, d.Hint, ColorReset))
			}
			out.WriteString("\n")
		} else {
			// Context lines (5 lines above / 5 lines below): dim line numbers
			lineNumStr := fmt.Sprintf("%*d", gutterWidth, l)
			out.WriteString(fmt.Sprintf("  %s%s |%s %s\n",
				ColorDim, lineNumStr, ColorReset,
				lineText))
		}
	}

	out.WriteString(fmt.Sprintf("  %s%s%s\n", ColorCyan, strings.Repeat(" ", gutterWidth)+" |", ColorReset))

	return out.String()
}

func getWordLength(line string, offset int) int {
	if offset < 0 || offset >= len(line) {
		return 1
	}

	length := 0
	for i := offset; i < len(line); i++ {
		ch := line[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			length++
		} else {
			break
		}
	}

	if length == 0 {
		return 1
	}
	return length
}

// GenerateHint provides intelligent suggestions based on error message and token
func GenerateHint(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "undefined variable"):
		varName := extractQuoted(msg)
		if varName != "" {
			return fmt.Sprintf("Declare variable using 'let %s = ...' or check spelling", varName)
		}
		return "Ensure variable is declared with 'let' or 'const' before use"
	case strings.Contains(lower, "cannot reassign to constant"):
		return "Variables declared with 'const' cannot be modified. Use 'let' instead"
	case strings.Contains(lower, "division by zero"):
		return "Denominator cannot be 0"
	case strings.Contains(lower, "index out of bounds"):
		return "Array index is outside the valid range (0 to len - 1)"
	case strings.Contains(lower, "expected next token"):
		return "Check syntax around this token"
	default:
		return ""
	}
}

func extractQuoted(s string) string {
	parts := strings.Fields(s)
	for _, p := range parts {
		if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") {
			return strings.Trim(p, "'")
		}
	}
	return ""
}
