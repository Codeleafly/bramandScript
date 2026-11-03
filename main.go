package main

import (
	"bramandscript/compiler"
	"bramandscript/diagnostics"
	"bramandscript/formatter"
	"bramandscript/lexer"
	"bramandscript/parser"
	"bramandscript/repl"
	"bramandscript/testrunner"
	"bramandscript/vm"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const VERSION = "0.1.1"
const DIVIDER = "__________________________________________________________________"
const StandaloneSignature = "BMD_STANDALONE_1"
const StandaloneFooterLen = 24

func main() {
	if checkEmbeddedStandalone() {
		return
	}

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
			fmt.Println("Usage: bramand run <filename.bmd> [args...]")
			os.Exit(1)
		}
		filename := os.Args[2]
		vm.SetProgramArgs(os.Args[3:])
		executeFile(filename)

	case "-e":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand -e <code> [args...]")
			os.Exit(1)
		}
		vm.SetProgramArgs(os.Args[3:])
		executeSource(os.Args[2], "<eval>")

	case "test":
		var testPaths []string
		verbose := false
		for _, arg := range os.Args[2:] {
			if arg == "-v" || arg == "--verbose" {
				verbose = true
			} else {
				testPaths = append(testPaths, arg)
			}
		}
		exitCode := testrunner.Run(testrunner.RunnerOptions{
			Paths:   testPaths,
			Verbose: verbose,
		})
		os.Exit(exitCode)

	case "fmt":
		var fmtPaths []string
		check := false
		diff := false
		for _, arg := range os.Args[2:] {
			if arg == "-c" || arg == "--check" {
				check = true
			} else if arg == "--diff" {
				diff = true
			} else {
				fmtPaths = append(fmtPaths, arg)
			}
		}
		exitCode := formatter.RunCLI(formatter.FormatOptions{
			Paths: fmtPaths,
			Check: check,
			Write: !check,
			Diff:  diff,
		})
		os.Exit(exitCode)

	case "build":
		handleBuildCommand(os.Args[2:])

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
			vm.SetProgramArgs(os.Args[2:])
			executeFile(command)
		} else {
			fmt.Printf("Unknown command or file: %s\n", command)
			printHelp()
			os.Exit(1)
		}
	}
}

func checkEmbeddedStandalone() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	f, err := os.Open(exePath)
	if err != nil {
		return false
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || fi.Size() < StandaloneFooterLen {
		return false
	}

	footer := make([]byte, StandaloneFooterLen)
	_, err = f.ReadAt(footer, fi.Size()-StandaloneFooterLen)
	if err != nil {
		return false
	}

	sig := string(footer[8:24])
	if sig != StandaloneSignature {
		return false
	}

	payloadLen := binary.BigEndian.Uint64(footer[0:8])
	if payloadLen == 0 || int64(payloadLen) > fi.Size()-StandaloneFooterLen {
		return false
	}

	payload := make([]byte, payloadLen)
	_, err = f.ReadAt(payload, fi.Size()-StandaloneFooterLen-int64(payloadLen))
	if err != nil {
		return false
	}

	vm.SetProgramArgs(os.Args[1:])
	executeSource(string(payload), exePath)
	return true
}

func handleBuildCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bramand build <source.bmd> [-o output_binary]")
		os.Exit(1)
	}

	inputFile := ""
	outputFile := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		} else if inputFile == "" {
			inputFile = args[i]
		}
	}

	if inputFile == "" {
		fmt.Println("Usage: bramand build <source.bmd> [-o output_binary]")
		os.Exit(1)
	}

	if outputFile == "" {
		base := filepath.Base(inputFile)
		ext := filepath.Ext(base)
		outputFile = strings.TrimSuffix(base, ext)
		if runtime.GOOS == "windows" {
			outputFile += ".exe"
		}
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read source file '%s': %s\n", inputFile, err)
		os.Exit(1)
	}
	source := string(data)

	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		fmt.Fprintf(os.Stderr, "[ERROR] Syntax error in '%s': %s (line %d, col %d)\n", inputFile, p.SyntaxErrors[0].Message, p.SyntaxErrors[0].Line, p.SyntaxErrors[0].Col)
		os.Exit(1)
	}
	comp := compiler.New()
	comp.SetBaseDir(filepath.Dir(inputFile))
	if err := comp.Compile(prog); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Compile error in '%s': %s\n", inputFile, err)
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not locate current executable: %s\n", err)
		os.Exit(1)
	}
	exeBytes, err := os.ReadFile(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read current executable: %s\n", err)
		os.Exit(1)
	}

	if len(exeBytes) >= StandaloneFooterLen {
		tail := exeBytes[len(exeBytes)-StandaloneFooterLen:]
		if string(tail[8:24]) == StandaloneSignature {
			oldPayloadLen := binary.BigEndian.Uint64(tail[0:8])
			baseLen := len(exeBytes) - StandaloneFooterLen - int(oldPayloadLen)
			if baseLen > 0 {
				exeBytes = exeBytes[:baseLen]
			}
		}
	}

	outF, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not create output file '%s': %s\n", outputFile, err)
		os.Exit(1)
	}
	defer outF.Close()

	if _, err := outF.Write(exeBytes); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing binary header: %s\n", err)
		os.Exit(1)
	}

	if _, err := outF.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing source payload: %s\n", err)
		os.Exit(1)
	}

	footer := make([]byte, StandaloneFooterLen)
	binary.BigEndian.PutUint64(footer[0:8], uint64(len(data)))
	copy(footer[8:24], []byte(StandaloneSignature))

	if _, err := outF.Write(footer); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing binary footer: %s\n", err)
		os.Exit(1)
	}

	totalSize := int64(len(exeBytes) + len(data) + StandaloneFooterLen)
	fmt.Printf("\n%s✦ BramandScript Standalone Executable Builder%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("Source File:  %s\n", inputFile)
	fmt.Printf("Target Exe:   %s\n", outputFile)
	fmt.Printf("Payload Size: %d bytes\n", len(data))
	fmt.Printf("Total Binary: %.2f MB\n", float64(totalSize)/(1024*1024))
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("%s[OK] Standalone executable created successfully: %s%s\n\n", diagnostics.ColorGreen, outputFile, diagnostics.ColorReset)
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
	executeSource(string(data), filename)
}

func executeSource(source, filename string) {
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
	comp.SetBaseDir(filepath.Dir(filename))
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
	fmt.Printf("%s✦ BramandScript Language CLI (v%s)%s\n", diagnostics.ColorBold, VERSION, diagnostics.ColorReset)
	fmt.Printf("%s%s%s\n\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  bramand                   Start interactive REPL")
	fmt.Println("  bramand repl              Start interactive REPL")
	fmt.Println("  bramand run <file.bmd>    Run source file")
	fmt.Println("  bramand <file.bmd>        Direct execution shortcut")
	fmt.Println("  bramand test [paths...]   Execute test suites (*_test.bmd)")
	fmt.Println("  bramand fmt [paths...]    Format source code canonically")
	fmt.Println("  bramand fmt -c [paths...] Check if files are formatted")
	fmt.Println("  bramand build <src.bmd>   Compile into standalone native binary")
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
