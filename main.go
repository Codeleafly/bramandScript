package main

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/parser"
	"bramandscript/repl"
	"bramandscript/vm"
	"fmt"
	"os"
)

const VERSION = "0.1.0"

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
		fmt.Printf("BramandScript version %s (Universe Bytecode VM)\n", VERSION)

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
		fmt.Fprintf(os.Stderr, "Error reading file %s: %s\n", filename, err)
		os.Exit(1)
	}

	input := string(data)
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintf(os.Stderr, "Syntax errors in %s:\n", filename)
		for _, msg := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", msg)
		}
		os.Exit(1)
	}

	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %s\n", err)
		os.Exit(1)
	}

	machine := vm.New(comp.Bytecode())
	err = machine.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "VM execution error: %s\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`BramandScript — A High-Performance Universal Programming Language

Usage:
  bramand                   Start interactive REPL
  bramand repl              Start interactive REPL
  bramand run <file.bmd>    Execute BramandScript source file
  bramand version           Show version information
  bramand help              Show this help message`)
}
