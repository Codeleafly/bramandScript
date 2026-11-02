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
	"path/filepath"
	"runtime"
	"strings"
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

	case "pkg", "lib", "package":
		handlePkgCommand(os.Args[2:])

	case "install":
		handlePkgCommand(append([]string{"install"}, os.Args[2:]...))

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
	fmt.Println("  bramand pkg list          List installed packages in ~/.bramand_libs")
	fmt.Println("  bramand pkg install <src> Install package to centralized registry")
	fmt.Println("  bramand pkg remove <name> Remove package from centralized registry")
	fmt.Println("  bramand pkg init <name>   Initialize package scaffold")
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

func handlePkgCommand(args []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not determine user home directory: %s\n", err)
		os.Exit(1)
	}
	libsDir := filepath.Join(homeDir, ".bramand_libs")
	if err := os.MkdirAll(libsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not create library directory: %s\n", err)
		os.Exit(1)
	}

	if len(args) < 1 {
		printPkgHelp()
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "path":
		fmt.Println(libsDir)

	case "list", "ls":
		entries, err := os.ReadDir(libsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to list packages: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s✦ BramandScript Centralized Package Registry%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
		fmt.Printf("Location: %s\n", libsDir)
		fmt.Println("──────────────────────────────────────────────────")
		count := 0
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			sizeStr := fmt.Sprintf("%d B", info.Size())
			if e.IsDir() {
				fmt.Printf("  ❯ %s (directory)\n", e.Name())
			} else {
				fmt.Printf("  ❯ %s (%s)\n", e.Name(), sizeStr)
			}
			count++
		}
		if count == 0 {
			fmt.Println("  (no packages currently installed)")
		}
		fmt.Println("──────────────────────────────────────────────────")

	case "install", "add":
		if len(args) < 2 {
			fmt.Println("Usage: bramand pkg install <source_file_or_dir> [name@version]")
			os.Exit(1)
		}
		src := args[1]
		targetName := ""
		if len(args) >= 3 {
			targetName = args[2]
		} else {
			targetName = filepath.Base(src)
			targetName = strings.TrimSuffix(targetName, ".bmd")
		}

		destDir := filepath.Join(libsDir, targetName)
		srcInfo, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Source file or directory '%s' not found\n", src)
			os.Exit(1)
		}

		if srcInfo.IsDir() {
			if err := copyDir(src, destDir); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Installation failed: %s\n", err)
				os.Exit(1)
			}
		} else {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Could not create destination directory: %s\n", err)
				os.Exit(1)
			}
			destFile := filepath.Join(destDir, "index.bmd")
			data, err := os.ReadFile(src)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed reading source: %s\n", err)
				os.Exit(1)
			}
			if err := os.WriteFile(destFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed writing destination: %s\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("[OK] Successfully installed '%s' to %s\n", targetName, destDir)

	case "remove", "rm", "uninstall":
		if len(args) < 2 {
			fmt.Println("Usage: bramand pkg remove <name@version>")
			os.Exit(1)
		}
		pkgName := args[1]
		targetDir := filepath.Join(libsDir, pkgName)
		targetFile := filepath.Join(libsDir, pkgName+".bmd")

		removed := false
		if _, err := os.Stat(targetDir); err == nil {
			_ = os.RemoveAll(targetDir)
			removed = true
		}
		if _, err := os.Stat(targetFile); err == nil {
			_ = os.Remove(targetFile)
			removed = true
		}

		if removed {
			fmt.Printf("[OK] Package '%s' removed successfully\n", pkgName)
		} else {
			fmt.Printf("[WARN] Package '%s' not found in %s\n", pkgName, libsDir)
		}

	case "init":
		if len(args) < 2 {
			fmt.Println("Usage: bramand pkg init <name@version>")
			os.Exit(1)
		}
		pkgName := args[1]
		if err := os.MkdirAll(pkgName, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Could not create directory '%s': %s\n", pkgName, err)
			os.Exit(1)
		}
		sampleFile := filepath.Join(pkgName, "index.bmd")
		sampleContent := fmt.Sprintf("// BramandScript Package: %s\n\nfn info() {\n    return \"Package %s initialized\";\n}\n", pkgName, pkgName)
		if err := os.WriteFile(sampleFile, []byte(sampleContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed writing index.bmd: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Initialized package directory '%s' with index.bmd\n", pkgName)

	default:
		printPkgHelp()
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func printPkgHelp() {
	fmt.Printf("%s✦ BramandScript Package & Library Manager CLI%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Println("Centralized package registry in $HOME/.bramand_libs/ (Deno-style zero node_modules)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  bramand pkg list                        List installed packages")
	fmt.Println("  bramand pkg install <path> [name@ver]   Install package into centralized cache")
	fmt.Println("  bramand pkg remove <name@ver>           Remove package from centralized cache")
	fmt.Println("  bramand pkg path                        Show cache directory path")
	fmt.Println("  bramand pkg init <name@ver>             Scaffold new package with index.bmd")
}
