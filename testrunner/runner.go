package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bramandscript/compiler"
	"bramandscript/diagnostics"
	"bramandscript/lexer"
	"bramandscript/parser"
	"bramandscript/vm"
)

type FileTestResult struct {
	Path     string
	Passed   bool
	Duration time.Duration
	Cases    []vm.TestCaseResult
	Error    string
}

type RunnerOptions struct {
	Paths   []string
	Verbose bool
}

func Run(opts RunnerOptions) int {
	vm.SetTestRunnerActive(true)
	defer vm.SetTestRunnerActive(false)

	files := discoverTestFiles(opts.Paths)
	if len(files) == 0 {
		fmt.Printf("%s[WARN] No test files found matching '*_test.bmd' or 'test_*.bmd'%s\n", diagnostics.ColorYellow, diagnostics.ColorReset)
		return 0
	}

	fmt.Printf("\n%s✦ BramandScript Test Runner%s\n", diagnostics.ColorBold, diagnostics.ColorReset)
	fmt.Printf("Discovered %d test file(s)\n", len(files))
	fmt.Println("──────────────────────────────────────────────────")

	startTotal := time.Now()
	var results []FileTestResult
	totalPassedCases := 0
	totalFailedCases := 0
	filesPassed := 0
	filesFailed := 0

	for _, file := range files {
		fileRes := runSingleTestFile(file, opts.Verbose)
		results = append(results, fileRes)

		if fileRes.Passed {
			filesPassed++
		} else {
			filesFailed++
		}

		for _, tc := range fileRes.Cases {
			if tc.Passed {
				totalPassedCases++
			} else {
				totalFailedCases++
			}
		}
		// If file had no explicit test() blocks, count file as 1 test case
		if len(fileRes.Cases) == 0 {
			if fileRes.Passed {
				totalPassedCases++
			} else {
				totalFailedCases++
			}
		}
	}

	totalDuration := time.Since(startTotal)

	fmt.Println("──────────────────────────────────────────────────")
	if filesFailed == 0 {
		fmt.Printf("%sTest Files: %s%d passed%s, %d total\n", diagnostics.ColorBold, diagnostics.ColorGreen, filesPassed, diagnostics.ColorReset, len(files))
		fmt.Printf("Tests:      %s%d passed%s, %d total\n", diagnostics.ColorGreen, totalPassedCases, diagnostics.ColorReset, totalPassedCases+totalFailedCases)
		fmt.Printf("Duration:   %v\n\n", totalDuration)
		fmt.Printf("%s[OK] All test suites passed!%s\n\n", diagnostics.ColorGreen, diagnostics.ColorReset)
		return 0
	} else {
		fmt.Printf("%sTest Files: %s%d failed%s, %d passed, %d total\n", diagnostics.ColorBold, diagnostics.ColorRed, filesFailed, diagnostics.ColorReset, filesPassed, len(files))
		fmt.Printf("Tests:      %s%d failed%s, %d passed, %d total\n", diagnostics.ColorRed, totalFailedCases, diagnostics.ColorReset, totalPassedCases, totalPassedCases+totalFailedCases)
		fmt.Printf("Duration:   %v\n\n", totalDuration)
		fmt.Printf("%s[ERROR] Some test suites failed.%s\n\n", diagnostics.ColorRed, diagnostics.ColorReset)
		return 1
	}
}

func runSingleTestFile(filePath string, verbose bool) FileTestResult {
	start := time.Now()
	fmt.Printf("\n• %s%s%s\n", diagnostics.ColorBold, filePath, diagnostics.ColorReset)

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("  %s[FAIL] File read error: %s%s\n", diagnostics.ColorRed, err, diagnostics.ColorReset)
		return FileTestResult{Path: filePath, Passed: false, Duration: time.Since(start), Error: err.Error()}
	}

	source := string(data)
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.SyntaxErrors) > 0 {
		for _, se := range p.SyntaxErrors {
			fmt.Printf("  %s[FAIL] SyntaxError (%s:%d:%d): %s%s\n", diagnostics.ColorRed, filePath, se.Line, se.Col, se.Message, diagnostics.ColorReset)
		}
		return FileTestResult{Path: filePath, Passed: false, Duration: time.Since(start), Error: "SyntaxError"}
	}

	comp := compiler.New()
	comp.SetBaseDir(filepath.Dir(filePath))
	if err := comp.Compile(program); err != nil {
		fmt.Printf("  %s[FAIL] CompileError: %s%s\n", diagnostics.ColorRed, err, diagnostics.ColorReset)
		return FileTestResult{Path: filePath, Passed: false, Duration: time.Since(start), Error: err.Error()}
	}

	vm.ResetTestContext(false)
	machine := vm.New(comp.Bytecode())
	runErr := machine.Run()

	fileDuration := time.Since(start)
	ctx := vm.GetTestContext()
	cases := make([]vm.TestCaseResult, len(ctx.Cases))
	copy(cases, ctx.Cases)

	if runErr != nil {
		fmt.Printf("  %s[FAIL] RuntimeError: %s%s\n", diagnostics.ColorRed, runErr.Error(), diagnostics.ColorReset)
		return FileTestResult{Path: filePath, Passed: false, Duration: fileDuration, Cases: cases, Error: runErr.Error()}
	}

	if ctx.TotalFailed > 0 {
		return FileTestResult{Path: filePath, Passed: false, Duration: fileDuration, Cases: cases, Error: "Test block failure"}
	}

	if len(cases) == 0 {
		fmt.Printf("  %s[OK] All top-level assertions passed (%v)%s\n", diagnostics.ColorGreen, fileDuration, diagnostics.ColorReset)
	}

	return FileTestResult{Path: filePath, Passed: true, Duration: fileDuration, Cases: cases}
}

func discoverTestFiles(paths []string) []string {
	var files []string
	if len(paths) == 0 {
		if _, err := os.Stat("tests"); err == nil {
			paths = []string{"tests"}
		} else {
			paths = []string{"."}
		}
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
			base := filepath.Base(path)
			if strings.HasSuffix(base, "_test.bmd") || strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".bmd") {
				files = append(files, path)
			}
			return nil
		})
	}

	return files
}
