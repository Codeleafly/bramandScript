package main

import (
	"bramandscript/compiler"
	"bramandscript/diagnostics"
	"bramandscript/formatter"
	"bramandscript/lexer"
	"bramandscript/parser"
	"bramandscript/pkg"
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

var (
	flagMaxFrames = 1024
	flagMaxStack  = 1048576
)

func safeExit(code int) {
	vm.RestoreTerm()
	os.Exit(code)
}

func main() {
	defer vm.RestoreTerm()

	if checkEmbeddedStandalone() {
		return
	}

	if len(os.Args) < 2 {
		repl.Start(os.Stdin, os.Stdout)
		return
	}

	command := os.Args[1]

	switch command {
	case "cli":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/cli" as cli; let code = cli.run_cli_main(args()); exit(code);`, "<cli>")

	case "bmd-repl":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/repl" as repl; repl.start_repl();`, "<repl>")

	case "bmd-test":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/test_runner" as tr; let code = tr.run_test_suites(args()); exit(code);`, "<test_runner>")

	case "bmd-pkg":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/pkg" as pm; let code = pm.run_pkg_cli(args()); exit(code);`, "<pkg_manager>")

	case "bmd-fmt":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/fmt" as fmt_pkg; let code = fmt_pkg.run_fmt_cli(args(), false); exit(code);`, "<formatter>")

	case "bmd-disasm":
		vm.SetProgramArgs(os.Args[2:])
		executeSource(`import "std/tooling/disasm" as dis; let a = args(); if len(a) == 0 { println("Usage: bramand bmd-disasm <file | expr>"); exit(1); } let code = dis.disassemble_target(a[0]); exit(code);`, "<disasm>")

	case "repl":
		repl.Start(os.Stdin, os.Stdout)

	case "run":
		handleRunCommand(os.Args[2:])

	case "-e":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand -e <code> [args...]")
			safeExit(1)
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
		safeExit(exitCode)

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
		safeExit(exitCode)

	case "build":
		handleBuildCommand(os.Args[2:])

	case "disasm":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand disasm [-O] <filename.bmd>")
			safeExit(1)
		}
		handleDisasmCommand(os.Args[2:])

	case "profile":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand profile <filename.bmd> [args...]")
			safeExit(1)
		}
		vm.SetProgramArgs(os.Args[3:])
		handleProfileCommand(os.Args[2])

	case "bench":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand bench <filename.bmd> [args...]")
			safeExit(1)
		}
		vm.SetProgramArgs(os.Args[3:])
		handleBenchCommand(os.Args[2])

	case "fuzz":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand fuzz <filename.bmd> [args...]")
			safeExit(1)
		}
		vm.SetProgramArgs(os.Args[3:])
		handleFuzzCommand(os.Args[2])

	case "record":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand record <filename.bmd> [-o trace.json]")
			safeExit(1)
		}
		handleRecordCommand(os.Args[2:])

	case "replay":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bramand replay <trace.json>")
			safeExit(1)
		}
		handleReplayCommand(os.Args[2])

	case "pkg", "lib", "package":
		handlePkgCommand(os.Args[2:])

	case "install":
		exitCode := pkg.InstallCLI(os.Args[2:])
		safeExit(exitCode)

	case "get":
		exitCode := pkg.GetCLI(os.Args[2:])
		safeExit(exitCode)

	case "cache":
		exitCode := pkg.CacheCLI(os.Args[2:])
		safeExit(exitCode)

	case "version", "-v", "--version":
		fmt.Printf("BramandScript %s (%s-%s)\n", VERSION, runtime.GOARCH, runtime.GOOS)

	case "help", "-h", "--help":
		printHelp()

	default:
		// If argument ends with .bmd or is a file, run it directly with full flag support
		if _, err := os.Stat(command); err == nil || strings.HasSuffix(command, ".bmd") {
			handleRunCommand(os.Args[1:])
		} else {
			fmt.Printf("Unknown command or file: %s\n", command)
			printHelp()
			safeExit(1)
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

	// Check if payload is a self-contained dependency bundle
	if pkg.IsBundleData(payload) {
		bundle, err := pkg.DeserializeBundle(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Corrupted standalone bundle: %s\n", err)
			safeExit(1)
		}
		pkg.RegisterActiveBundle(bundle)
		executeSource(bundle.MainSource, exePath)
		return true
	}

	executeSource(string(payload), exePath)
	return true
}

func handleBuildCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bramand build [-O] <source.bmd> [-o output_binary]")
		safeExit(1)
	}

	inputFile := ""
	outputFile := ""
	optimize := false

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		} else if args[i] == "-O" || args[i] == "--optimize" {
			optimize = true
		} else if inputFile == "" {
			inputFile = args[i]
		}
	}

	if inputFile == "" {
		fmt.Println("Usage: bramand build [-O] <source.bmd> [-o output_binary]")
		safeExit(1)
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
		safeExit(1)
	}
	source := string(data)

	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		fmt.Fprintf(os.Stderr, "[ERROR] Syntax error in '%s': %s (line %d, col %d)\n", inputFile, p.SyntaxErrors[0].Message, p.SyntaxErrors[0].Line, p.SyntaxErrors[0].Col)
		safeExit(1)
	}
	var comp *compiler.Compiler
	if optimize {
		comp = compiler.NewOptimizingCompiler()
	} else {
		comp = compiler.New()
	}
	comp.SetBaseDir(filepath.Dir(inputFile))
	if err := comp.Compile(prog); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Compile error in '%s': %s\n", inputFile, err)
		safeExit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not locate current executable: %s\n", err)
		safeExit(1)
	}
	exeBytes, err := os.ReadFile(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read current executable: %s\n", err)
		safeExit(1)
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

	bundle, bundleBytes, err := pkg.BuildBundle(
		inputFile,
		source,
		comp.ImportedModulePaths(),
		comp.ImportedModuleSources(),
		comp.ImportedModuleManifests(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to construct standalone bundle: %s\n", err)
		safeExit(1)
	}

	payload := bundleBytes

	outF, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not create output file '%s': %s\n", outputFile, err)
		safeExit(1)
	}
	defer outF.Close()

	if _, err := outF.Write(exeBytes); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing binary header: %s\n", err)
		safeExit(1)
	}

	if _, err := outF.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing bundle payload: %s\n", err)
		safeExit(1)
	}

	footer := make([]byte, StandaloneFooterLen)
	binary.BigEndian.PutUint64(footer[0:8], uint64(len(payload)))
	copy(footer[8:24], []byte(StandaloneSignature))

	if _, err := outF.Write(footer); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed writing binary footer: %s\n", err)
		safeExit(1)
	}

	totalSize := int64(len(exeBytes) + len(payload) + StandaloneFooterLen)
	fmt.Printf("\n%s✦ BramandScript Standalone Executable Builder%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("Source File:  %s\n", inputFile)
	fmt.Printf("Target Exe:   %s\n", outputFile)
	if len(bundle.Files) > 0 {
		fmt.Printf("Dependencies: %d embedded module(s)\n", len(bundle.Files))
		for impPath, f := range bundle.Files {
			origin := "Local"
			if strings.Contains(impPath, "github.com") || strings.Contains(impPath, "cdn") {
				origin = "Registry"
			}
			if f.PackageName != "" {
				fmt.Printf("  ❯ %s [%s: %s]\n", impPath, origin, f.PackageName)
			} else {
				fmt.Printf("  ❯ %s [%s]\n", impPath, origin)
			}
		}
	} else {
		fmt.Println("Dependencies: 0 (pure standalone script)")
	}
	fmt.Printf("Payload Size: %d bytes (compressed bundle)\n", len(payload))
	fmt.Printf("Total Binary: %.2f MB\n", float64(totalSize)/(1024*1024))
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("%s[OK] Standalone executable created successfully: %s%s\n\n", diagnostics.ColorGreen, outputFile, diagnostics.ColorReset)
}

func executeFile(filename string) {
	executeFileOpt(filename, false)
}

func executeFileOpt(filename string, optimize bool) {
	data, err := os.ReadFile(filename)
	if err != nil {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Kind:     "IOError",
			Message:  fmt.Sprintf("Could not read file: %s", err),
			Hint:     "Verify that the file path exists and is readable",
		}
		fmt.Fprint(os.Stderr, diag.Format())
		safeExit(1)
	}
	executeSourceOpt(string(data), filename, optimize)
}

func executeSource(source, filename string) {
	executeSourceOpt(source, filename, false)
}

func executeSourceOpt(source, filename string, optimize bool) {
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
		safeExit(1)
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
		safeExit(1)
	}

	var comp *compiler.Compiler
	if optimize {
		comp = compiler.NewOptimizingCompiler()
	} else {
		comp = compiler.New()
	}
	comp.SetBaseDir(filepath.Dir(filename))
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
		fmt.Fprint(os.Stderr, diag.Format())
		safeExit(1)
	}

	machine := vm.New(comp.Bytecode())
	if flagMaxFrames > 0 {
		machine.SetMaxFrames(flagMaxFrames)
	}
	if flagMaxStack > 0 {
		machine.SetMaxStackSize(flagMaxStack)
	}
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
		safeExit(1)
	}
}

func handleRunCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bramand run [options] <filename.bmd> [args...]")
		safeExit(1)
	}

	optimize := false
	var programArgs []string
	filename := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-O" || arg == "--optimize":
			optimize = true
		case arg == "--allow-all":
			vm.GetPermissions().AllowAll = true
		case arg == "--allow-run" || arg == "--allow-exec":
			vm.GetPermissions().AllowRun = true
		case arg == "--allow-write":
			vm.GetPermissions().AllowWrite = true
		case arg == "--allow-net":
			vm.GetPermissions().AllowNet = true
		case arg == "--allow-read":
			vm.GetPermissions().AllowRead = true
		case strings.HasPrefix(arg, "--max-frames="):
			val := strings.TrimPrefix(arg, "--max-frames=")
			var n int
			fmt.Sscanf(val, "%d", &n)
			if n > 0 {
				flagMaxFrames = n
			}
		case arg == "--max-frames" && i+1 < len(args):
			i++
			var n int
			fmt.Sscanf(args[i], "%d", &n)
			if n > 0 {
				flagMaxFrames = n
			}
		case strings.HasPrefix(arg, "--max-stack="):
			val := strings.TrimPrefix(arg, "--max-stack=")
			var n int
			fmt.Sscanf(val, "%d", &n)
			if n > 0 {
				flagMaxStack = n
			}
		case arg == "--max-stack" && i+1 < len(args):
			i++
			var n int
			fmt.Sscanf(args[i], "%d", &n)
			if n > 0 {
				flagMaxStack = n
			}
		case strings.HasPrefix(arg, "-"):
			fmt.Printf("Unknown option: %s\n", arg)
			safeExit(1)
		default:
			if filename == "" {
				filename = arg
				programArgs = args[i+1:]
				i = len(args)
			}
		}
	}

	if filename == "" {
		fmt.Println("Usage: bramand run [options] <filename.bmd> [args...]")
		safeExit(1)
	}

	vm.SetProgramArgs(programArgs)
	executeFileOpt(filename, optimize)
}

func printHelp() {
	fmt.Printf("%s✦ BramandScript Language CLI (v%s)%s\n", diagnostics.ColorBold, VERSION, diagnostics.ColorReset)
	fmt.Printf("%s%s%s\n\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  bramand                   Start interactive REPL")
	fmt.Println("  bramand repl              Start interactive REPL")
	fmt.Println("  bramand run [opts] <file> Run source file with 64-bit bytecode VM")
	fmt.Println("  bramand <file.bmd>        Direct execution shortcut")
	fmt.Println("  bramand test [paths...]   Execute test suites (*_test.bmd)")
	fmt.Println("  bramand fmt [paths...]    Format source code canonically")
	fmt.Println("  bramand fmt -c [paths...] Check if files are formatted")
	fmt.Println("  bramand build [-O] <src>  Compile into standalone native binary (-O for optimized bytecode)")
	fmt.Println("  bramand disasm [-O] <file> Disassemble source file to bytecode (-O for optimized)")
	fmt.Println("  bramand profile <src.bmd> Profile execution with opcode timing hotspots")
	fmt.Println("  bramand install <target>  Install package from GitHub, CDN/URL, or local directory")
	fmt.Println("  bramand get               Install missing dependencies declared in bmdpkg.json")
	fmt.Println("  bramand cache <cmd>       Manage cache and integrity (info, prune, verify)")
	fmt.Println("  bramand pkg <cmd>         Package manager CLI commands")
	fmt.Println("  bramand version           Show version")
	fmt.Println("  bramand help              Show this help message")
	fmt.Println()
	fmt.Println("Execution & Permission Options:")
	fmt.Println("  -O, --optimize            Enable Ahead-of-Time compiler optimizations")
	fmt.Println("  --allow-all               Grant all execution, file system, and network permissions")
	fmt.Println("  --allow-run               Grant external subprocess execution permission (exec)")
	fmt.Println("  --allow-write             Grant file write, append, and deletion permission")
	fmt.Println("  --allow-net               Grant network listening permission (tcp_listen)")
	fmt.Println("  --allow-read              Grant file reading permission")
	fmt.Println("  --max-frames <N>          Configure maximum call stack frames depth (default: 1024)")
	fmt.Println("  --max-stack <N>           Configure maximum value stack size (default: 1048576)")
	fmt.Println()
	fmt.Println("REPL Commands:")
	fmt.Println("  :editor                   Open multi-line editor mode")
	fmt.Println("  :editor <file.bmd>        Write & save file from REPL")
	fmt.Println("  :disasm <code>            Disassemble snippet into opcodes")
	fmt.Println("  :profile <code>           Profile snippet execution hotspots")
	fmt.Println("  :clear                    Clear screen")
	fmt.Println("  :reset                    Reset runtime state")
	fmt.Println("  :help                     Documentation and language reference")
	fmt.Println("  :exit                     Exit REPL")
	fmt.Printf("%s%s%s\n", diagnostics.ColorDim, DIVIDER, diagnostics.ColorReset)
}

func handleDisasmCommand(args []string) {
	optimize := false
	filename := ""
	for _, arg := range args {
		if arg == "-O" || arg == "--optimize" {
			optimize = true
		} else if filename == "" {
			filename = arg
		}
	}
	if filename == "" {
		fmt.Println("Usage: bramand disasm [-O] <filename.bmd>")
		safeExit(1)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read file '%s': %s\n", filename, err)
		safeExit(1)
	}
	source := string(data)
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Line:     p.SyntaxErrors[0].Line,
			Col:      p.SyntaxErrors[0].Col,
			Kind:     "SyntaxError",
			Message:  p.SyntaxErrors[0].Message,
		}
		fmt.Fprint(os.Stderr, diag.Format())
		safeExit(1)
	}

	var comp *compiler.Compiler
	if optimize {
		comp = compiler.NewOptimizingCompiler()
	} else {
		comp = compiler.New()
	}
	comp.SetBaseDir(filepath.Dir(filename))
	comp.SetCurrentFile(filename, source)
	if err := comp.Compile(prog); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Compilation failed: %s\n", err)
		safeExit(1)
	}

	fmt.Print(compiler.FormatBytecode(comp.Bytecode()))
}

func handleProfileCommand(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read file '%s': %s\n", filename, err)
		safeExit(1)
	}
	source := string(data)
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		diag := diagnostics.Diagnostic{
			Filename: filename,
			Source:   source,
			Line:     p.SyntaxErrors[0].Line,
			Col:      p.SyntaxErrors[0].Col,
			Kind:     "SyntaxError",
			Message:  p.SyntaxErrors[0].Message,
		}
		fmt.Fprint(os.Stderr, diag.Format())
		safeExit(1)
	}

	comp := compiler.New()
	comp.SetBaseDir(filepath.Dir(filename))
	comp.SetCurrentFile(filename, source)
	if err := comp.Compile(prog); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Compilation failed: %s\n", err)
		safeExit(1)
	}

	machine := vm.New(comp.Bytecode())
	report, err := machine.RunProfiled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Runtime failed: %s\n", err)
		safeExit(1)
	}

	fmt.Print(report.String())
}

func handleBenchCommand(filename string) {
	fmt.Printf("✦ BramandScript Benchmark Runner: %s\n", filename)
	fmt.Println("──────────────────────────────────────────────────────────────────")
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read file '%s': %s\n", filename, err)
		safeExit(1)
	}
	executeSource(string(data), filename)
}

func handleFuzzCommand(filename string) {
	fmt.Printf("✦ BramandScript Fuzz Testing Engine: %s\n", filename)
	fmt.Println("──────────────────────────────────────────────────────────────────")
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read file '%s': %s\n", filename, err)
		safeExit(1)
	}
	executeSource(string(data), filename)
}

func handleRecordCommand(args []string) {
	sourceFile := args[0]
	outputTrace := "trace.json"
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputTrace = args[i+1]
			i++
		}
	}

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not read file '%s': %s\n", sourceFile, err)
		safeExit(1)
	}

	source := string(data)
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.SyntaxErrors) > 0 {
		fmt.Fprintf(os.Stderr, "[ERROR] Syntax error in '%s': %s\n", sourceFile, p.SyntaxErrors[0].Message)
		safeExit(1)
	}

	comp := compiler.New()
	comp.SetBaseDir(filepath.Dir(sourceFile))
	comp.SetCurrentFile(sourceFile, source)
	if err := comp.Compile(prog); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Compilation failed: %s\n", err)
		safeExit(1)
	}

	machine := vm.New(comp.Bytecode())
	trace, err := machine.RunRecorded(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Execution error: %s\n", err)
	}

	if trace != nil {
		if err := trace.Save(outputTrace); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to save trace: %s\n", err)
			safeExit(1)
		}
		fmt.Printf("[OK] Execution recorded: %d steps written to %s\n", trace.TotalSteps, outputTrace)
	}
}

func handleReplayCommand(traceFile string) {
	trace, err := vm.LoadTrace(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to load trace: %s\n", err)
		safeExit(1)
	}

	debugger := vm.NewReplayDebugger(trace)
	if err := debugger.RunInteractive(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Replay error: %s\n", err)
		safeExit(1)
	}
}

func handlePkgCommand(args []string) {
	if len(args) < 1 {
		printPkgHelp()
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "install", "add":
		exitCode := pkg.InstallCLI(args[1:])
		safeExit(exitCode)

	case "get":
		exitCode := pkg.GetCLI(args[1:])
		safeExit(exitCode)

	case "cache":
		exitCode := pkg.CacheCLI(args[1:])
		safeExit(exitCode)

	case "info":
		exitCode := pkg.CacheInfoCLI(args[1:])
		safeExit(exitCode)

	case "prune":
		exitCode := pkg.CachePruneCLI(args[1:])
		safeExit(exitCode)

	case "verify":
		exitCode := pkg.CacheVerifyCLI(args[1:])
		safeExit(exitCode)

	case "path":
		fmt.Println(pkg.GetBmdLibsDir())

	case "list", "ls":
		exitCode := pkg.PkgListCLI(args[1:])
		safeExit(exitCode)

	case "init":
		exitCode := pkg.PkgInitCLI(args[1:])
		safeExit(exitCode)

	case "remove", "rm", "uninstall":
		if len(args) < 2 {
			fmt.Println("Usage: bramand pkg remove <name@version>")
			safeExit(1)
		}
		pkgName := args[1]
		libsDir := pkg.GetRegistryDir()
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
	fmt.Printf("%s✦ BramandScript Centralized Package & Module Manager CLI%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Println("Centralized package registry in $HOME/.bmd_libs/ (Zero node_modules, zero duplicate storage)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  bmd install <target> [ver]     Install package from GitHub, CDN/URL, or local directory")
	fmt.Println("  bmd get                        Install missing dependencies declared in bmdpkg.json")
	fmt.Println("  bmd cache info                 Display centralized cache and registry metrics")
	fmt.Println("  bmd cache prune                Clean orphaned tarballs while preserving registry trees")
	fmt.Println("  bmd cache verify               Validate SHA-256 package checksums against integrity.db")
	fmt.Println("  bmd pkg list                   List installed packages in centralized registry")
	fmt.Println("  bmd pkg remove <name>          Remove package from centralized registry")
	fmt.Println("  bmd pkg path                   Print centralized library directory path")
	fmt.Println("  bmd pkg init [name]            Scaffold new package with bmdpkg.json and main.bmd")
}
