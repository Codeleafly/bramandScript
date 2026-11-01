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
	"strings"
)

// ANSI Color Codes
const (
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorRed     = "\033[1;31m"
	colorGreen   = "\033[1;32m"
	colorYellow  = "\033[1;33m"
	colorBlue    = "\033[1;34m"
	colorMagenta = "\033[1;35m"
	colorCyan    = "\033[1;36m"
	colorWhite   = "\033[1;37m"
)

const BANNER = "\033[1;36m========================================================\033[0m\n" +
	"\033[1m   ____                                          __    \033[0m\n" +
	"\033[1m  / __ )_________ _____ ___  ____ _____  ____   / /    \033[0m\n" +
	"\033[1m / __  / ___/ __ `/ __ `__ \\/ __ `/ __ \\/ __ \\ / /     \033[0m\n" +
	"\033[1m/ /_/ / /  / /_/ / / / / / / /_/ / / / / /_/ // /      \033[0m\n" +
	"\033[1m/_____/_/   \\__,_/_/ /_/ /_/\\__,_/_/ /_/\\__,_//_/       \033[0m\n" +
	"\033[1;36m========================================================\033[0m\n" +
	"\033[1m✦ BramandScript v0.1.0\033[0m \033[2m(Universe Bytecode Virtual Machine)\033[0m\n" +
	"\033[2mCommands: :help, :clear, :reset, :exit\033[0m\n"

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	fmt.Fprint(out, BANNER)

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
			fmt.Fprintf(out, "%s%sbmd%s %s❯%s ", colorBold, colorCyan, colorReset, colorGreen, colorReset)
		} else {
			indent := strings.Repeat("  ", depth)
			fmt.Fprintf(out, "%s···%s %s", colorDim, colorReset, indent)
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
				fmt.Fprintf(out, "%s[Terminated] Farewell from BramandScript.%s\n", colorDim, colorReset)
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
				fmt.Fprintf(out, "%s[OK] Runtime state reset.%s\n", colorGreen, colorReset)
				continue
			}
			if trimmed == ":help" {
				printHelp(out)
				continue
			}
		}

		if trimmed == "" && depth == 0 {
			continue
		}

		// Calculate block depth: count unclosed '{' or '('
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

		// If still inside open block, continue reading
		if depth > 0 {
			continue
		}

		source := buffer.String()
		buffer.Reset()

		l := lexer.New(source)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Fprintf(out, "%s[SYNTAX ERROR]%s\n", colorRed, colorReset)
			for _, msg := range p.Errors() {
				fmt.Fprintf(out, "  %s%s%s\n", colorRed, msg, colorReset)
			}
			continue
		}

		comp := compiler.NewWithState(symbolTable, constants)
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "%s[COMPILATION ERROR]%s %s\n", colorRed, colorReset, err)
			continue
		}

		code := comp.Bytecode()
		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "%s[RUNTIME ERROR]%s %s\n", colorRed, colorReset, err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
			fmt.Fprintf(out, "%s→%s %s%s%s\n", colorDim, colorReset, colorYellow, lastPopped.Inspect(), colorReset)
		}
	}
}

func printHelp(out io.Writer) {
	fmt.Fprintf(out, "\n%sBramandScript Interactive REPL Guide:%s\n", colorBold, colorReset)
	fmt.Fprintf(out, "  %sVariables:%s    let x = 10, const PI = 3.14159 (semicolons optional)\n", colorCyan, colorReset)
	fmt.Fprintf(out, "  %sConditionals:%s if x > 5 { print(\"Big\") } else { print(\"Small\") }\n", colorCyan, colorReset)
	fmt.Fprintf(out, "  %sFunctions:%s    fn add(a, b) { return a + b }\n", colorCyan, colorReset)
	fmt.Fprintf(out, "  %sPrinting:%s     print(\"Value is:\", x)\n", colorCyan, colorReset)
	fmt.Fprintf(out, "  %sLists & Maps:%s [1, 2, 3], {\"name\": \"Earth\"}\n", colorCyan, colorReset)
	fmt.Fprintf(out, "\n%sREPL Commands:%s\n", colorBold, colorReset)
	fmt.Fprintf(out, "  %s:help%s         Show this guide\n", colorGreen, colorReset)
	fmt.Fprintf(out, "  %s:clear%s        Clear screen\n", colorGreen, colorReset)
	fmt.Fprintf(out, "  %s:reset%s        Reset VM state & variables\n", colorGreen, colorReset)
	fmt.Fprintf(out, "  %s:exit%s         Quit REPL\n\n", colorGreen, colorReset)
}
