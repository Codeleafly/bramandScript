package formatter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bramandscript/lexer"
	"bramandscript/parser"
)

type FormatOptions struct {
	Paths []string
	Check bool
	Write bool
	Diff  bool
}

// Format formats the provided BramandScript source code canonically.
func Format(source string) (string, error) {
	lines := strings.Split(source, "\n")
	var formattedLines []string

	indentLevel := 0
	indentStr := "    "
	inBlockComment := false
	prevLineWasBlank := false

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)

		// Handle empty lines
		if trimmed == "" {
			if !prevLineWasBlank && len(formattedLines) > 0 {
				formattedLines = append(formattedLines, "")
				prevLineWasBlank = true
			}
			continue
		}
		prevLineWasBlank = false

		// Handle block comments
		if inBlockComment {
			formattedLines = append(formattedLines, strings.Repeat(indentStr, indentLevel)+trimmed)
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlockComment = true
			formattedLines = append(formattedLines, strings.Repeat(indentStr, indentLevel)+trimmed)
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}

		// Handle single line comments
		if strings.HasPrefix(trimmed, "//") {
			comment := formatComment(trimmed)
			formattedLines = append(formattedLines, strings.Repeat(indentStr, indentLevel)+comment)
			continue
		}

		// Count closing tokens at start of line to decrease indent before printing
		leadingClosers := 0
		for _, ch := range trimmed {
			if ch == '}' || ch == ']' || ch == ')' {
				leadingClosers++
			} else {
				break
			}
		}

		currentIndent := indentLevel - leadingClosers
		if currentIndent < 0 {
			currentIndent = 0
		}

		// Format line content
		formattedContent := formatLineContent(trimmed)

		// Combine closing brace and else if applicable
		if strings.HasPrefix(formattedContent, "} else") || strings.HasPrefix(formattedContent, "} catch") {
			currentIndent = indentLevel - 1
			if currentIndent < 0 {
				currentIndent = 0
			}
		}

		lineWithIndent := strings.Repeat(indentStr, currentIndent) + formattedContent
		formattedLines = append(formattedLines, lineWithIndent)

		// Adjust indentLevel for next lines based on net braces/brackets
		netOpen := countNetOpen(trimmed)
		indentLevel += netOpen
		if indentLevel < 0 {
			indentLevel = 0
		}
	}

	result := strings.Join(formattedLines, "\n")
	result = strings.TrimRight(result, "\n\r\t ") + "\n"

	// Safety verification: Ensure syntax is preserved
	l := lexer.New(result)
	p := parser.New(l)
	p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		return source, fmt.Errorf("formatting verification failed: syntax errors detected in formatted code: %s", p.SyntaxErrors[0].Message)
	}

	return result, nil
}

func isQuoteEscaped(s string, i int) bool {
	count := 0
	for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
		count++
	}
	return count%2 == 1
}

func countNetOpen(line string) int {
	net := 0
	inString := false
	var strQuote byte = 0

	for i := 0; i < len(line); i++ {
		ch := line[i]

		// Check for inline comment start
		if !inString && ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		// Handle string literals
		if (ch == '"' || ch == '`') && !isQuoteEscaped(line, i) {
			if !inString {
				inString = true
				strQuote = ch
			} else if strQuote == ch {
				inString = false
			}
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{', '[', '(':
			net++
		case '}', ']', ')':
			net--
		}
	}
	return net
}

func formatComment(comment string) string {
	if strings.HasPrefix(comment, "//") {
		after := strings.TrimPrefix(comment, "//")
		if len(after) > 0 && after[0] != ' ' && after[0] != '/' && after[0] != '!' {
			return "// " + after
		}
	}
	return comment
}

func formatLineContent(line string) string {
	// If line has trailing single-line comment, preserve it
	codePart := line
	commentPart := ""

	inStr := false
	var q byte
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if (ch == '"' || ch == '`') && !isQuoteEscaped(line, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			continue
		}
		if !inStr && ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			codePart = strings.TrimSpace(line[:i])
			commentPart = "  " + formatComment(strings.TrimSpace(line[i:]))
			break
		}
	}

	formattedCode := formatCodeTokens(codePart)
	return formattedCode + commentPart
}

func formatCodeTokens(code string) string {
	if code == "" {
		return ""
	}

	var buf bytes.Buffer
	inStr := false
	var q byte

	for i := 0; i < len(code); i++ {
		ch := code[i]

		if (ch == '"' || ch == '`') && !isQuoteEscaped(code, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			buf.WriteByte(ch)
			continue
		}

		if inStr {
			buf.WriteByte(ch)
			continue
		}

		// Operator and punctuation spacing
		switch ch {
		case ',':
			buf.WriteByte(',')
			if i+1 < len(code) && code[i+1] != ' ' && code[i+1] != '\n' {
				buf.WriteByte(' ')
			}
		case ';':
			// Remove space before semicolon
			s := buf.String()
			for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
				buf.Truncate(buf.Len() - 1)
				s = buf.String()
			}
			buf.WriteByte(';')
			if i+1 < len(code) && code[i+1] != ' ' && code[i+1] != '\n' && code[i+1] != '}' {
				buf.WriteByte(' ')
			}
		case ':':
			// For match cases or hash keys
			s := buf.String()
			for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
				buf.Truncate(buf.Len() - 1)
				s = buf.String()
			}
			buf.WriteByte(':')
			if i+1 < len(code) && code[i+1] != ' ' && code[i+1] != ':' {
				buf.WriteByte(' ')
			}
		default:
			buf.WriteByte(ch)
		}
	}

	res := buf.String()

	// Normalize spaces around binary operators
	operators := []string{
		"|>", "<<", ">>", "**", "==", "!=", "<=", ">=", "+=", "-=", "*=", "/=", "=>", "<-", "&&", "||",
	}
	for _, op := range operators {
		res = normalizeOpSpacing(res, op)
	}

	// Single char operators with care
	singleOps := []string{"=", "+", "*", "/", "%", "<", ">", "&", "|", "^"}
	for _, op := range singleOps {
		res = normalizeSingleOpSpacing(res, op)
	}

	// Clean duplicate spaces
	for strings.Contains(res, "  ") {
		// Only replace spaces outside of strings
		res = cleanDuplicateSpacesOutsideStrings(res)
		break
	}

	res = cleanParenBracketSpacing(res)

	// Keyword spacing
	if strings.Contains(res, "}else") {
		res = strings.ReplaceAll(res, "}else", "} else")
	}
	if strings.Contains(res, "else{") {
		res = strings.ReplaceAll(res, "else{", "else {")
	}

	// Standard control keywords space before parenthesis
	keywords := []string{"if", "while", "for", "match"}
	for _, kw := range keywords {
		pattern := kw + "("
		if strings.Contains(res, pattern) {
			res = strings.ReplaceAll(res, pattern, kw+" (")
		}
	}

	// Ensure space before opening brace
	if strings.Contains(res, "){") {
		res = strings.ReplaceAll(res, "){", ") {")
	}

	return strings.TrimSpace(res)
}

func normalizeOpSpacing(input, op string) string {
	var buf bytes.Buffer
	inStr := false
	var q byte
	opLen := len(op)

	for i := 0; i < len(input); {
		ch := input[i]
		if (ch == '"' || ch == '`') && !isQuoteEscaped(input, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			buf.WriteByte(ch)
			i++
			continue
		}
		if inStr {
			buf.WriteByte(ch)
			i++
			continue
		}

		if i+opLen <= len(input) && input[i:i+opLen] == op {
			// Trim space before op in buf
			s := buf.String()
			for len(s) > 0 && s[len(s)-1] == ' ' {
				buf.Truncate(buf.Len() - 1)
				s = buf.String()
			}
			buf.WriteByte(' ')
			buf.WriteString(op)
			buf.WriteByte(' ')
			i += opLen
			// Skip spaces after op in input
			for i < len(input) && input[i] == ' ' {
				i++
			}
			continue
		}

		buf.WriteByte(ch)
		i++
	}
	return buf.String()
}

func normalizeSingleOpSpacing(input, op string) string {
	var buf bytes.Buffer
	inStr := false
	var q byte
	opByte := op[0]

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if (ch == '"' || ch == '`') && !isQuoteEscaped(input, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			buf.WriteByte(ch)
			continue
		}
		if inStr {
			buf.WriteByte(ch)
			continue
		}

		// Check if part of a compound operator
		if ch == opByte {
			prev := byte(0)
			if i > 0 {
				prev = input[i-1]
			}
			next := byte(0)
			if i+1 < len(input) {
				next = input[i+1]
			}

			// Don't format inside compound operators
			if prev == '=' || prev == '!' || prev == '<' || prev == '>' || prev == '+' || prev == '-' || prev == '*' || prev == '/' || prev == '|' || prev == '&' || prev == '^' {
				buf.WriteByte(ch)
				continue
			}
			if next == '=' || next == '>' || next == '<' || next == '*' || next == '+' || next == '-' || next == '|' || next == '&' {
				buf.WriteByte(ch)
				continue
			}

			// Trim spaces before
			s := buf.String()
			for len(s) > 0 && s[len(s)-1] == ' ' {
				buf.Truncate(buf.Len() - 1)
				s = buf.String()
			}
			buf.WriteByte(' ')
			buf.WriteByte(ch)
			buf.WriteByte(' ')
			for i+1 < len(input) && input[i+1] == ' ' {
				i++
			}
			continue
		}

		buf.WriteByte(ch)
	}
	return buf.String()
}

func cleanDuplicateSpacesOutsideStrings(input string) string {
	var buf bytes.Buffer
	inStr := false
	var q byte

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if (ch == '"' || ch == '`') && !isQuoteEscaped(input, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			buf.WriteByte(ch)
			continue
		}
		if inStr {
			buf.WriteByte(ch)
			continue
		}

		if ch == ' ' && i+1 < len(input) && input[i+1] == ' ' {
			continue
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}

func cleanParenBracketSpacing(input string) string {
	var buf bytes.Buffer
	inStr := false
	var q byte

	for i := 0; i < len(input); {
		ch := input[i]
		if (ch == '"' || ch == '`') && !isQuoteEscaped(input, i) {
			if !inStr {
				inStr = true
				q = ch
			} else if q == ch {
				inStr = false
			}
			buf.WriteByte(ch)
			i++
			continue
		}
		if inStr {
			buf.WriteByte(ch)
			i++
			continue
		}

		if ch == '(' || ch == '[' {
			if ch == '(' {
				s := buf.String()
				if len(s) > 0 && s[len(s)-1] == ' ' {
					trimmedBefore := strings.TrimRight(s, " \t")
					lastWord := getLastWord(trimmedBefore)
					keywordsWithSpace := map[string]bool{
						"if": true, "while": true, "for": true, "match": true,
						"in": true, "case": true, "and": true, "or": true, "not": true,
						"return": true,
					}
					if !keywordsWithSpace[lastWord] && lastWord != "" {
						buf.Truncate(len(trimmedBefore))
					}
				}
			}
			buf.WriteByte(ch)
			i++
			for i < len(input) && input[i] == ' ' {
				i++
			}
			continue
		}

		if ch == ')' || ch == ']' {
			s := buf.String()
			for len(s) > 0 && s[len(s)-1] == ' ' {
				buf.Truncate(buf.Len() - 1)
				s = buf.String()
			}
			buf.WriteByte(ch)
			i++
			continue
		}

		buf.WriteByte(ch)
		i++
	}

	return buf.String()
}

func getLastWord(s string) string {
	i := len(s) - 1
	for i >= 0 && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_') {
		i--
	}
	return s[i+1:]
}

// RunCLI handles the `bramand fmt` CLI sub-command.
func RunCLI(opts FormatOptions) int {
	files := discoverBmdFiles(opts.Paths)
	if len(files) == 0 {
		fmt.Println("[WARN] No .bmd files found to format")
		return 0
	}

	needsFormat := 0
	formattedCount := 0

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Could not read %s: %s\n", file, err)
			continue
		}
		original := string(data)
		formatted, err := Format(original)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Could not format %s: %s\n", file, err)
			continue
		}

		if original != formatted {
			needsFormat++
			if opts.Check {
				fmt.Printf("  • %s needs formatting\n", file)
			} else {
				if err := os.WriteFile(file, []byte(formatted), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] Failed to write %s: %s\n", file, err)
				} else {
					fmt.Printf("  [OK] Formatted %s\n", file)
					formattedCount++
				}
			}
		}
	}

	if opts.Check {
		if needsFormat > 0 {
			fmt.Printf("\n[FAIL] %d file(s) need formatting. Run 'bramand fmt' to fix.\n", needsFormat)
			return 1
		} else {
			fmt.Println("[OK] All files properly formatted.")
			return 0
		}
	}

	if formattedCount > 0 {
		fmt.Printf("\n[OK] Formatted %d file(s).\n", formattedCount)
	} else {
		fmt.Println("[OK] All files already formatted.")
	}
	return 0
}

func discoverBmdFiles(paths []string) []string {
	var files []string
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if strings.HasSuffix(p, ".bmd") {
				files = append(files, p)
			}
			continue
		}

		_ = filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				name := fi.Name()
				if name == ".git" || name == "node_modules" || name == ".bramand_libs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".bmd") {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}
