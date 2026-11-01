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
)

const PROMPT = "bmd >> "

const BANNER = "========================================================\n" +
	"   ____                                          __    \n" +
	"  / __ )_________ _____ ___  ____ _____  ____   / /    \n" +
	" / __  / ___/ __ `/ __ `__ \\/ __ `/ __ \\/ __ \\ / /     \n" +
	"/ /_/ / /  / /_/ / / / / / / /_/ / / / / /_/ // /      \n" +
	"/_____/_/   \\__,_/_/ /_/ /_/\\__,_/_/ /_/\\__,_//_/       \n" +
	"========================================================\n" +
	"🌌 BramandScript v0.1.0 (Bytecode Virtual Machine)\n" +
	"Universe Standard Language | Pure Go Stdlib (0 Dependencies)\n" +
	"Type 'exit' or 'quit' to leave REPL.\n"

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	fmt.Fprint(out, BANNER)

	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalsSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range compiler.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		if line == "exit" || line == "quit" {
			fmt.Fprintln(out, "Farewell from BramandScript! 🌌")
			return
		}

		if line == "" {
			continue
		}

		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithState(symbolTable, constants)
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "Compilation error:\n %s\n", err)
			continue
		}

		code := comp.Bytecode()
		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Runtime error:\n %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
			fmt.Fprintf(out, "%s\n", lastPopped.Inspect())
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	fmt.Fprintf(out, "Syntax error:\n")
	for _, msg := range errors {
		fmt.Fprintf(out, "\t%s\n", msg)
	}
}
