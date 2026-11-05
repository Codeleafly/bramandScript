package vm

import (
	"sync"
	"time"
)

type TestCaseResult struct {
	Name     string
	Suite    string
	Passed   bool
	Duration time.Duration
	Error    string
	Line     int
}

type TestSuiteContext struct {
	mu           sync.Mutex
	ActiveSuite  string
	Cases        []TestCaseResult
	TotalPassed  int
	TotalFailed  int
	SilentMode   bool
	InTestRunner bool
}

var globalTestCtx = &TestSuiteContext{
	Cases: make([]TestCaseResult, 0),
}

func GetTestContext() *TestSuiteContext {
	return globalTestCtx
}

func SetTestRunnerActive(active bool) {
	globalTestCtx.mu.Lock()
	defer globalTestCtx.mu.Unlock()
	globalTestCtx.InTestRunner = active
}

func IsTestRunnerActive() bool {
	globalTestCtx.mu.Lock()
	defer globalTestCtx.mu.Unlock()
	return globalTestCtx.InTestRunner
}

func ResetTestContext(silent bool) {
	globalTestCtx.mu.Lock()
	defer globalTestCtx.mu.Unlock()
	globalTestCtx.ActiveSuite = ""
	globalTestCtx.Cases = make([]TestCaseResult, 0)
	globalTestCtx.TotalPassed = 0
	globalTestCtx.TotalFailed = 0
	globalTestCtx.SilentMode = silent
}
