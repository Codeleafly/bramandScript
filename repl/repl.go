package repl

import (
	"bramandscript/compiler"
	"bramandscript/diagnostics"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"bramandscript/vm"
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const DIVIDER = "__________________________________________________________________"

func Start(in io.Reader, out io.Writer) {
	vm.SetStdout(out)
	defer vm.SetStdout(nil)

	scanner := bufio.NewScanner(in)

	fmt.Fprintf(out, "%s✦ %sBramandScript 0.1.1%s %s(%s-%s) · High-Performance Bytecode VM%s\n",
		diagnostics.ColorCyan, diagnostics.ColorBold, diagnostics.ColorReset,
		diagnostics.ColorDim, runtime.GOARCH, runtime.GOOS, diagnostics.ColorReset)
	fmt.Fprintf(out, "%s  Type %s:help%s for documentation, %s:editor%s for multi-line, %s:exit%s to quit.%s\n\n",
		diagnostics.ColorDim, diagnostics.ColorCyan, diagnostics.ColorDim,
		diagnostics.ColorCyan, diagnostics.ColorDim,
		diagnostics.ColorCyan, diagnostics.ColorDim, diagnostics.ColorReset)

	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalsSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range compiler.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	var buffer strings.Builder
	depth := 0

	for {
		if depth == 0 {
			fmt.Fprintf(out, "%s✦ %sbmd %s❯%s ", diagnostics.ColorCyan, diagnostics.ColorBold, diagnostics.ColorGreen, diagnostics.ColorReset)
		} else {
			indent := strings.Repeat("  ", depth)
			fmt.Fprintf(out, "%s· ···%s %s", diagnostics.ColorDim, diagnostics.ColorReset, indent)
		}

		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Top-level REPL commands
		if depth == 0 {
			if trimmed == ":exit" || trimmed == ":quit" || trimmed == "exit" || trimmed == "quit" {
				return
			}
			if trimmed == ":clear" {
				fmt.Fprint(out, "\033[H\033[2J")
				continue
			}
			if trimmed == ":reset" {
				constants = []object.Object{}
				globals = make([]object.Object, vm.GlobalsSize)
				symbolTable = compiler.NewSymbolTable()
				for i, v := range compiler.Builtins {
					symbolTable.DefineBuiltin(i, v.Name)
				}
				fmt.Fprintf(out, "%s[OK] Runtime session reset.%s\n", diagnostics.ColorGreen, diagnostics.ColorReset)
				continue
			}
			if trimmed == ":help" {
				printHelp(out)
				continue
			}
			if strings.HasPrefix(trimmed, ":editor") || strings.HasPrefix(trimmed, ":edit") {
				runEditorMode(scanner, out, symbolTable, &constants, globals, trimmed)
				continue
			}
			if strings.HasPrefix(trimmed, ":disasm") {
				codeArg := strings.TrimSpace(strings.TrimPrefix(trimmed, ":disasm"))
				if codeArg == "" {
					fmt.Fprintf(out, "%sUsage: :disasm <expression or code>%s\n", diagnostics.ColorYellow, diagnostics.ColorReset)
					continue
				}
				runDisasm(codeArg, out, symbolTable, constants)
				continue
			}
			if strings.HasPrefix(trimmed, ":profile") {
				codeArg := strings.TrimSpace(strings.TrimPrefix(trimmed, ":profile"))
				if codeArg == "" {
					fmt.Fprintf(out, "%sUsage: :profile <expression or code>%s\n", diagnostics.ColorYellow, diagnostics.ColorReset)
					continue
				}
				runProfile(codeArg, out, symbolTable, &constants, globals)
				continue
			}
		}

		if trimmed == "" && depth == 0 {
			continue
		}

		// Calculate block depth: tracking { and (
		for _, ch := range line {
			if ch == '{' || ch == '(' {
				depth++
			} else if ch == '}' || ch == ')' {
				if depth > 0 {
					depth--
				}
			}
		}

		buffer.WriteString(line)
		buffer.WriteString("\n")

		if depth > 0 {
			continue
		}

		source := buffer.String()
		buffer.Reset()

		executeCode(source, "<repl>", out, symbolTable, &constants, globals)
	}
}

func runEditorMode(scanner *bufio.Scanner, out io.Writer, symbolTable *compiler.SymbolTable, constants *[]object.Object, globals []object.Object, cmd string) {
	fmt.Fprintf(out, "%s%s%s\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
	fmt.Fprintf(out, "%sMulti-Line Editor Mode%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Fprintf(out, "Type code line-by-line. Type %s:done%s to execute, %s:cancel%s to abort.\n",
		diagnostics.ColorCyan, diagnostics.ColorReset, diagnostics.ColorYellow, diagnostics.ColorReset)
	fmt.Fprintf(out, "To save to file on disk: %s:done <filename.bmd>%s\n",
		diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "%s%s%s\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)

	var defaultSavePath string
	cmdParts := strings.Fields(cmd)
	if len(cmdParts) > 1 {
		if cmdParts[1] == "save" && len(cmdParts) > 2 {
			defaultSavePath = cmdParts[2]
		} else if cmdParts[1] != "save" {
			defaultSavePath = cmdParts[1]
		}
	}

	var editorBuffer strings.Builder
	lineNum := 1

	for {
		fmt.Fprintf(out, "%s%02d |%s ", diagnostics.ColorDim, lineNum, diagnostics.ColorReset)
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == ":cancel" || trimmed == ":abort" {
			fmt.Fprintf(out, "%s[Aborted] Editor code discarded.%s\n", diagnostics.ColorYellow, diagnostics.ColorReset)
			return
		}

		if trimmed == ":done" || strings.HasPrefix(trimmed, ":done ") || strings.HasPrefix(trimmed, ":save ") {
			savePath := defaultSavePath
			parts := strings.Fields(trimmed)
			if len(parts) > 1 {
				savePath = parts[1]
			}

			source := editorBuffer.String()
			if savePath != "" {
				err := os.WriteFile(savePath, []byte(source), 0644)
				if err != nil {
					fmt.Fprintf(out, "%s[Error] Could not save file: %s%s\n", diagnostics.ColorRed, err, diagnostics.ColorReset)
				} else {
					fmt.Fprintf(out, "%s[OK] Saved %d bytes to %s%s\n", diagnostics.ColorGreen, len(source), savePath, diagnostics.ColorReset)
				}
			}

			fmt.Fprintf(out, "%sExecuting buffer...%s\n", diagnostics.ColorDim, diagnostics.ColorReset)
			executeCode(source, "<editor>", out, symbolTable, constants, globals)
			return
		}

		editorBuffer.WriteString(line)
		editorBuffer.WriteString("\n")
		lineNum++
	}
}

func executeCode(source, filename string, out io.Writer, symbolTable *compiler.SymbolTable, constants *[]object.Object, globals []object.Object) {
	defer vm.EnsureStdoutNewline()

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.SyntaxErrors) > 0 {
		firstErr := p.SyntaxErrors[0]
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Line:     firstErr.Line,
			Col:      firstErr.Col,
			Kind:     "SyntaxError",
			Message:  firstErr.Message,
			Hint:     diagnostics.GenerateHint(firstErr.Message),
		}
		fmt.Fprint(out, diag.Format())
		vm.EnsureStdoutNewline()
		return
	} else if len(p.Errors()) > 0 {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Line:     1,
			Col:      1,
			Kind:     "SyntaxError",
			Message:  p.Errors()[0],
			Hint:     diagnostics.GenerateHint(p.Errors()[0]),
		}
		fmt.Fprint(out, diag.Format())
		vm.EnsureStdoutNewline()
		return
	}

	comp := compiler.NewWithState(symbolTable, *constants)
	comp.SetCurrentFile(filename, source)
	err := comp.Compile(program)
	if err != nil {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Kind:     "CompileError",
			Message:  err.Error(),
			Hint:     diagnostics.GenerateHint(err.Error()),
		}

		if ce, ok := err.(*compiler.CompilerError); ok {
			diag.Line = ce.Line
			diag.Col = ce.Col
			diag.Message = ce.Message
			if ce.File != "" {
				diag.Filename = ce.File
			}
			if ce.Source != "" {
				diag.Source = ce.Source
			}
			diag.Hint = diagnostics.GenerateHint(ce.Message)
		} else {
			diag.Line = 1
			diag.Col = 1
		}

		fmt.Fprint(out, diag.Format())
		vm.EnsureStdoutNewline()
		return
	}

	code := comp.Bytecode()
	*constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	err = machine.Run()
	vm.EnsureStdoutNewline()
	if err != nil {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Line:     1,
			Col:      1,
			Kind:     "RuntimeError",
			Message:  err.Error(),
			Hint:     diagnostics.GenerateHint(err.Error()),
		}
		fmt.Fprint(out, diag.Format())
		vm.EnsureStdoutNewline()
		return
	}

	lastPopped := machine.LastPoppedStackElem()
	if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
		fmt.Fprintf(out, "%s\n", FormatReplValue(lastPopped))
		vm.ResetStdoutNewlineState()
	}
}

// FormatReplValue renders runtime objects with syntax highlighting for REPL sessions.
func FormatReplValue(obj object.Object) string {
	return formatReplValueDepth(obj, 0)
}

func formatReplValueDepth(obj object.Object, depth int) string {
	if obj == nil {
		return diagnostics.ColorDim + "null" + diagnostics.ColorReset
	}
	if depth >= 16 {
		return diagnostics.ColorDim + "..." + diagnostics.ColorReset
	}

	switch o := obj.(type) {
	case *object.Integer:
		return fmt.Sprintf("%s%d%s", diagnostics.ColorYellow, o.Value, diagnostics.ColorReset)
	case *object.Float:
		s := strconv.FormatFloat(o.Value, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return fmt.Sprintf("%s%s%s", diagnostics.ColorYellow, s, diagnostics.ColorReset)
	case *object.BigInt:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorYellow, o.Inspect(), diagnostics.ColorReset)
	case *object.BigFloat:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorYellow, o.Inspect(), diagnostics.ColorReset)
	case *object.String:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorGreen, strconv.Quote(o.Value), diagnostics.ColorReset)
	case *object.Bytes:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorGreen, o.Inspect(), diagnostics.ColorReset)
	case *object.Boolean:
		return fmt.Sprintf("%s%t%s", diagnostics.ColorMagenta, o.Value, diagnostics.ColorReset)
	case *object.Null:
		return diagnostics.ColorDim + "null" + diagnostics.ColorReset
	case *object.CompiledFunction, *object.Closure:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorCyan, o.Inspect(), diagnostics.ColorReset)
	case *object.Builtin:
		return fmt.Sprintf("%s<builtin %s>%s", diagnostics.ColorCyan, o.Inspect(), diagnostics.ColorReset)
	case *object.Error:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorRed, o.Inspect(), diagnostics.ColorReset)
	case *object.Vector:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorYellow, o.Inspect(), diagnostics.ColorReset)
	case *object.Matrix:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorYellow, o.Inspect(), diagnostics.ColorReset)
	case *object.Channel:
		return fmt.Sprintf("%s%s%s", diagnostics.ColorCyan, o.Inspect(), diagnostics.ColorReset)
	case *object.Array:
		o.Mu.RLock()
		defer o.Mu.RUnlock()
		if len(o.Elements) == 0 {
			return "[]"
		}
		var elems []string
		for _, el := range o.Elements {
			elems = append(elems, formatReplValueDepth(el, depth+1))
		}
		return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
	case *object.Hash:
		o.Mu.RLock()
		defer o.Mu.RUnlock()

		// Check if it's a struct instance
		structNameKey := (&object.String{Value: "__struct__"}).HashKey()
		if structPair, isStruct := o.Pairs[structNameKey]; isStruct {
			structName := ""
			if strObj, ok := structPair.Value.(*object.String); ok {
				structName = strObj.Value
			}
			var fields []string
			for _, pair := range o.Pairs {
				keyStr := ""
				if sKey, ok := pair.Key.(*object.String); ok {
					keyStr = sKey.Value
				} else {
					keyStr = pair.Key.Inspect()
				}
				if strings.HasPrefix(keyStr, "__") {
					continue
				}
				if _, isClosure := pair.Value.(*object.Closure); isClosure {
					continue
				}
				valStr := formatReplValueDepth(pair.Value, depth+1)
				fields = append(fields, fmt.Sprintf("%s%s%s: %s", diagnostics.ColorCyan, keyStr, diagnostics.ColorReset, valStr))
			}
			sort.Strings(fields)
			if structName != "" {
				if len(fields) == 0 {
					return fmt.Sprintf("%s%s%s {}", diagnostics.ColorBold, structName, diagnostics.ColorReset)
				}
				return fmt.Sprintf("%s%s%s { %s }", diagnostics.ColorBold, structName, diagnostics.ColorReset, strings.Join(fields, ", "))
			}
		}

		if len(o.Pairs) == 0 {
			return "{}"
		}
		var pairs []string
		for _, pair := range o.Pairs {
			keyStr := formatReplValueDepth(pair.Key, depth+1)
			valStr := formatReplValueDepth(pair.Value, depth+1)
			pairs = append(pairs, fmt.Sprintf("%s: %s", keyStr, valStr))
		}
		sort.Strings(pairs)
		return fmt.Sprintf("{ %s }", strings.Join(pairs, ", "))
	default:
		return o.Inspect()
	}
}

func printHelp(out io.Writer) {
	fmt.Fprintf(out, "\n%sBramandScript Language Guide & Reference%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Fprintf(out, "%s%s%s\n\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)

	fmt.Fprintf(out, "%s1. Variables & Constants%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   let x = 100                     %s// Mutable variable (semicolons optional)%s\n", diagnostics.ColorDim, diagnostics.ColorReset)
	fmt.Fprintf(out, "   const PI = 3.14159              %s// Immutable constant%s\n", diagnostics.ColorDim, diagnostics.ColorReset)
	fmt.Fprintf(out, "   x += 5                          %s// Compound assignment (+=, -=, *=, /=)%s\n\n", diagnostics.ColorDim, diagnostics.ColorReset)

	fmt.Fprintf(out, "%s2. Control Flow%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   if x > 50 { print(\"High\") }      %s// Parentheses optional!%s\n", diagnostics.ColorDim, diagnostics.ColorReset)
	fmt.Fprintf(out, "   else if x == 50 { print(\"Mid\") }\n")
	fmt.Fprintf(out, "   else { print(\"Low\") }\n\n")

	fmt.Fprintf(out, "   let count = 0\n")
	fmt.Fprintf(out, "   while count < 5 { count += 1 }  %s// Loop with 'break' and 'continue'%s\n\n", diagnostics.ColorDim, diagnostics.ColorReset)

	fmt.Fprintf(out, "%s3. Functions & Closures%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   fn add(a, b) {\n")
	fmt.Fprintf(out, "       return a + b\n")
	fmt.Fprintf(out, "   }\n")
	fmt.Fprintf(out, "   print(\"Sum:\", add(10, 20))\n\n")

	fmt.Fprintf(out, "%s4. Collections & Dot Access%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   let list = [10, 20, 30]\n")
	fmt.Fprintf(out, "   list[0] = 99                    %s// Index assignment supported%s\n", diagnostics.ColorDim, diagnostics.ColorReset)
	fmt.Fprintf(out, "   let user = {\"name\": \"Aryan\", \"role\": \"developer\"}\n")
	fmt.Fprintf(out, "   print(user.name)                %s// Dot property access%s\n\n", diagnostics.ColorDim, diagnostics.ColorReset)

	fmt.Fprintf(out, "%s5. Standard Built-in Functions%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   print(a, b, ...)   Print values separated by space without newline\n")
	fmt.Fprintf(out, "   println(a, b, ...) Print values separated by space with newline\n")
	fmt.Fprintf(out, "   input(prompt)      Read interactive input from terminal\n")
	fmt.Fprintf(out, "   len(collection)    Length of string, array, or map\n")
	fmt.Fprintf(out, "   type(value)        Return type name as string (e.g., 'INTEGER')\n")
	fmt.Fprintf(out, "   slice(arr, 1, 3)   Sub-array slice\n")
	fmt.Fprintf(out, "   keys(map)          Array of map keys\n")
	fmt.Fprintf(out, "   values(map)        Array of map values\n")
	fmt.Fprintf(out, "   int(v), float(v)   Convert to numeric types\n")
	fmt.Fprintf(out, "   str(v), bool(v)    Convert to string or boolean\n")
	fmt.Fprintf(out, "   now()              Current Unix timestamp in seconds\n\n")

	fmt.Fprintf(out, "%s6. REPL Commands%s\n", diagnostics.ColorCyan, diagnostics.ColorReset)
	fmt.Fprintf(out, "   :editor            Multi-line code editor mode\n")
	fmt.Fprintf(out, "   :editor <file.bmd> Multi-line editor and save to file\n")
	fmt.Fprintf(out, "   :disasm <code>     Disassemble code into bytecode instructions\n")
	fmt.Fprintf(out, "   :profile <code>    Profile code execution with opcode hotspot analysis\n")
	fmt.Fprintf(out, "   :clear             Clear terminal screen\n")
	fmt.Fprintf(out, "   :reset             Reset runtime state and variables\n")
	fmt.Fprintf(out, "   :help              Show this guide\n")
	fmt.Fprintf(out, "   :exit              Quit REPL\n")
	fmt.Fprintf(out, "%s%s%s\n\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
}

func runDisasm(source string, out io.Writer, symbolTable *compiler.SymbolTable, constants []object.Object) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.SyntaxErrors) > 0 {
		firstErr := p.SyntaxErrors[0]
		diag := diagnostics.Diagnostic{
			Filename: "<repl:disasm>",
			Source:   source,
			Line:     firstErr.Line,
			Col:      firstErr.Col,
			Kind:     "SyntaxError",
			Message:  firstErr.Message,
			Hint:     diagnostics.GenerateHint(firstErr.Message),
		}
		fmt.Fprint(out, diag.Format())
		return
	}

	comp := compiler.NewWithState(symbolTable, constants)
	comp.SetCurrentFile("<repl:disasm>", source)
	err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(out, "%s[CompileError] %s%s\n", diagnostics.ColorRed, err.Error(), diagnostics.ColorReset)
		return
	}

	fmt.Fprint(out, compiler.FormatBytecode(comp.Bytecode()))
}

func runProfile(source string, out io.Writer, symbolTable *compiler.SymbolTable, constants *[]object.Object, globals []object.Object) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.SyntaxErrors) > 0 {
		firstErr := p.SyntaxErrors[0]
		diag := diagnostics.Diagnostic{
			Filename: "<repl:profile>",
			Source:   source,
			Line:     firstErr.Line,
			Col:      firstErr.Col,
			Kind:     "SyntaxError",
			Message:  firstErr.Message,
			Hint:     diagnostics.GenerateHint(firstErr.Message),
		}
		fmt.Fprint(out, diag.Format())
		return
	}

	comp := compiler.NewWithState(symbolTable, *constants)
	comp.SetCurrentFile("<repl:profile>", source)
	err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(out, "%s[CompileError] %s%s\n", diagnostics.ColorRed, err.Error(), diagnostics.ColorReset)
		return
	}

	code := comp.Bytecode()
	*constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	report, err := machine.RunProfiled()
	if err != nil {
		fmt.Fprintf(out, "%s[RuntimeError] %s%s\n", diagnostics.ColorRed, err.Error(), diagnostics.ColorReset)
		return
	}

	lastPopped := machine.LastPoppedStackElem()
	if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
		fmt.Fprintf(out, "%sResult:%s %s\n", diagnostics.ColorCyan, diagnostics.ColorReset, FormatReplValue(lastPopped))
	}

	fmt.Fprint(out, report.String())
}
