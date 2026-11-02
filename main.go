package main

import (
	"bramandscript/compiler"
	"bramandscript/diagnostics"
	"bramandscript/lexer"
	"bramandscript/parser"
	"bramandscript/repl"
	"bramandscript/vm"
	"fmt"
	"os"
	"runtime"
)

const VERSION = "0.1.0"
const DIVIDER = "__________________________________________________________________"

func main() {
	if len(os.Args) < 2 {
		repl.Start(os.Stdin, os.Stdout)
		return
	}

	command := os.Args[1]

	switch command {
	case "repl":
		repl.Start(os.Stdin, os.Stdout)

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand run <filename.bmd>")
			os.Exit(1)
		}
		filename := os.Args[2]
		executeFile(filename)

	case "version", "-v", "--version":
		fmt.Printf("BramandScript %s (%s-%s)\n", VERSION, runtime.GOARCH, runtime.GOOS)

	case "help", "-h", "--help":
		printHelp()

	default:
		// If argument ends with .bmd or is a file, run it directly
		if _, err := os.Stat(command); err == nil {
			executeFile(command)
		} else {
			fmt.Printf("Unknown command or file: %s\n", command)
			printHelp()
			os.Exit(1)
		}
	}
}

func executeFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Kind:     "IOError",
			Message:  fmt.Sprintf("Could not read file: %s", err),
			Hint:     "Verify that the file path exists and is readable",
		}
		fmt.Fprint(os.Stderr, diag.Format())
		os.Exit(1)
	}

	source := string(data)
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.SyntaxErrors) > 0 {
		for _, se := range p.SyntaxErrors {
			diag := diagnostics.Diagnostic{
				Filename: filename,
				Source:   source,
				Line:     se.Line,
				Col:      se.Col,
				Kind:     "SyntaxError",
				Message:  se.Message,
				Hint:     diagnostics.GenerateHint(se.Message),
			}
			fmt.Fprint(os.Stderr, diag.Format())
		}
		os.Exit(1)
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
		fmt.Fprint(os.Stderr, diag.Format())
		os.Exit(1)
	}

	comp := compiler.New()
	err = comp.Compile(program)
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
			diag.Hint = diagnostics.GenerateHint(ce.Message)
		} else {
			diag.Line = 1
			diag.Col = 1
		}
		fmt.Fprint(os.Stderr, diag.Format())
		os.Exit(1)
	}

	machine := vm.New(comp.Bytecode())
	err = machine.Run()
	if err != nil {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Kind:     "RuntimeError",
			Message:  err.Error(),
			Hint:     diagnostics.GenerateHint(err.Error()),
		}
		if re, ok := err.(*vm.RuntimeError); ok {
			diag.Line = re.Line
			diag.Col = re.Col
			diag.Message = re.Message
			diag.Hint = diagnostics.GenerateHint(re.Message)
		} else {
			diag.Line = 1
			diag.Col = 1
		}
		fmt.Fprint(os.Stderr, diag.Format())
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%sBramandScript Language CLI%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Printf("%s%s%s\n\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  bramand                   Start interactive REPL")
	fmt.Println("  bramand repl              Start interactive REPL")
	fmt.Println("  bramand run <file.bmd>    Run source file")
	fmt.Println("  bramand <file.bmd>        Direct execution shortcut")
	fmt.Println("  bramand version           Show version")
	fmt.Println("  bramand help              Show this help message")
	fmt.Println()
	fmt.Println("REPL Commands:")
	fmt.Println("  :editor                   Open multi-line editor mode")
	fmt.Println("  :editor <file.bmd>        Write & save file from REPL")
	fmt.Println("  :clear                    Clear screen")
	fmt.Println("  :reset                    Reset runtime state")
	fmt.Println("  :help                     Documentation and language reference")
	fmt.Println("  :exit                     Exit REPL")
	fmt.Printf("%s%s%s\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
}
