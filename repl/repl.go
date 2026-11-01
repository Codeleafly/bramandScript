package repl

import (
	"bufio"
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"bramandscript/vm"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	fmt.Fprintf(out, "BramandScript 0.1.0 (%s/%s)\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(out, "Type \":help\" for help, \":editor\" for multi-line editor, \":exit\" to quit.\n\n")

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
			fmt.Fprint(out, "bmd > ")
		} else {
			indent := strings.Repeat("  ", depth)
			fmt.Fprintf(out, "... %s", indent)
		}

		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// REPL Commands
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
				fmt.Fprintln(out, "[OK] Session state reset.")
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
		}

		if trimmed == "" && depth == 0 {
			continue
		}

		// Calculate block depth
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

		executeCode(source, out, symbolTable, &constants, globals)
	}
}

func runEditorMode(scanner *bufio.Scanner, out io.Writer, symbolTable *compiler.SymbolTable, constants *[]object.Object, globals []object.Object, cmd string) {
	fmt.Fprintln(out, "// Entering editor mode. Type ':done' to execute, ':cancel' to abort.")
	fmt.Fprintln(out, "// To save to file, type ':done <filename.bmd>' or ':save <filename.bmd>'.")

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
		fmt.Fprintf(out, "%02d | ", lineNum)
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == ":cancel" || trimmed == ":abort" {
			fmt.Fprintln(out, "[ABORTED] Editor buffer discarded.")
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
					fmt.Fprintf(out, "[ERROR] Could not save file %s: %s\n", savePath, err)
				} else {
					fmt.Fprintf(out, "[OK] Saved %d bytes to %s\n", len(source), savePath)
				}
			}

			fmt.Fprintln(out, "// Executing editor buffer...")
			executeCode(source, out, symbolTable, constants, globals)
			return
		}

		editorBuffer.WriteString(line)
		editorBuffer.WriteString("\n")
		lineNum++
	}
}

func executeCode(source string, out io.Writer, symbolTable *compiler.SymbolTable, constants *[]object.Object, globals []object.Object) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintln(out, "SyntaxError:")
		for _, msg := range p.Errors() {
			fmt.Fprintf(out, "  %s\n", msg)
		}
		return
	}

	comp := compiler.NewWithState(symbolTable, *constants)
	err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(out, "CompileError: %s\n", err)
		return
	}

	code := comp.Bytecode()
	*constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	err = machine.Run()
	if err != nil {
		fmt.Fprintf(out, "RuntimeError: %s\n", err)
		return
	}

	lastPopped := machine.LastPoppedStackElem()
	if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
		fmt.Fprintln(out, lastPopped.Inspect())
	}
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `
BramandScript REPL Help:
  Variables:     let x = 10, const PI = 3.14159
  Conditionals:  if x > 5 { ... } else if x == 5 { ... } else { ... }
  Loops:         while count < 10 { count = count + 1 } (break, continue)
  Functions:     fn greet(name) { return "Hello " + name }
  Builtins:      print(...), input(prompt), len(arr), type(v), keys(m), values(m)
  Types:         int(v), float(v), str(v), bool(v), slice(arr, 0, 2)

REPL Commands:
  :editor                 Enter multi-line editor mode
  :editor save <file.bmd> Save buffer to file and execute
  :clear                  Clear terminal screen
  :reset                  Reset runtime variables
  :help                   Show this help message
  :exit                   Exit REPL`)
}
