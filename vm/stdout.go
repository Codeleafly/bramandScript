package vm

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	stdoutMu       sync.Mutex
	stdoutWriter   io.Writer = os.Stdout
	lastStdoutByte byte      = '\n'
)

// SetStdout redirects vm standard output. Passing nil resets to os.Stdout.
func SetStdout(w io.Writer) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	if w == nil {
		stdoutWriter = os.Stdout
	} else {
		stdoutWriter = w
	}
	lastStdoutByte = '\n'
}

// GetStdout returns the active stdout writer.
func GetStdout() io.Writer {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	return stdoutWriter
}

// StdoutWrite writes string s to the active stdout writer and tracks the trailing byte.
func StdoutWrite(s string) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	if len(s) > 0 {
		lastStdoutByte = s[len(s)-1]
	}
	fmt.Fprint(stdoutWriter, s)
}

// EnsureStdoutNewline writes a newline to the active stdout writer if the last written byte was not a newline.
func EnsureStdoutNewline() bool {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	if lastStdoutByte != '\n' {
		fmt.Fprintln(stdoutWriter)
		lastStdoutByte = '\n'
		return true
	}
	return false
}

// ResetStdoutNewlineState resets the tracking state so subsequent prompts or values know the line is fresh.
func ResetStdoutNewlineState() {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	lastStdoutByte = '\n'
}
