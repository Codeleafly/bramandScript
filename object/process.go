package object

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// Process represents an active or finished child OS process with streamed stdio pipes.
type Process struct {
	Cmd      *exec.Cmd
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	PID      int
	Exited   bool
	ExitCode int

	Mu      sync.Mutex
	stdinMu sync.Mutex
}

// Type returns the ObjectType for Process.
func (p *Process) Type() ObjectType { return PROCESS_OBJ }

// Inspect returns the string representation of the Process.
func (p *Process) Inspect() string {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	if p.Exited {
		return fmt.Sprintf("process(pid=%d, exited=true, exit_code=%d)", p.PID, p.ExitCode)
	}
	return fmt.Sprintf("process(pid=%d, running=true)", p.PID)
}

// WriteStdin writes data into the process's standard input pipe.
func (p *Process) WriteStdin(data []byte) (int, error) {
	p.stdinMu.Lock()
	defer p.stdinMu.Unlock()

	if p.Stdin == nil {
		return 0, fmt.Errorf("process stdin is closed")
	}
	return p.Stdin.Write(data)
}

// CloseStdin closes the process's standard input pipe.
func (p *Process) CloseStdin() error {
	p.stdinMu.Lock()
	defer p.stdinMu.Unlock()

	if p.Stdin != nil {
		err := p.Stdin.Close()
		p.Stdin = nil
		return err
	}
	return nil
}

// ReadStdout reads up to maxBytes from the process's stdout pipe.
func (p *Process) ReadStdout(maxBytes int) ([]byte, error) {
	if p.Stdout == nil {
		return nil, io.EOF
	}
	buf := make([]byte, maxBytes)
	n, err := p.Stdout.Read(buf)
	if n > 0 {
		return buf[:n], nil
	}
	return nil, err
}

// ReadStdoutAll reads all remaining bytes from stdout until EOF.
func (p *Process) ReadStdoutAll() ([]byte, error) {
	if p.Stdout == nil {
		return []byte{}, nil
	}
	return io.ReadAll(p.Stdout)
}

// ReadStderr reads up to maxBytes from the process's stderr pipe.
func (p *Process) ReadStderr(maxBytes int) ([]byte, error) {
	if p.Stderr == nil {
		return nil, io.EOF
	}
	buf := make([]byte, maxBytes)
	n, err := p.Stderr.Read(buf)
	if n > 0 {
		return buf[:n], nil
	}
	return nil, err
}

// ReadStderrAll reads all remaining bytes from stderr until EOF.
func (p *Process) ReadStderrAll() ([]byte, error) {
	if p.Stderr == nil {
		return []byte{}, nil
	}
	return io.ReadAll(p.Stderr)
}

// Wait waits for the process to exit and returns the exit code and PID.
func (p *Process) Wait() (int, int, error) {
	p.Mu.Lock()
	if p.Exited {
		code := p.ExitCode
		pid := p.PID
		p.Mu.Unlock()
		return code, pid, nil
	}
	p.Mu.Unlock()

	err := p.Cmd.Wait()

	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.Exited = true

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			p.ExitCode = exitErr.ExitCode()
			return p.ExitCode, p.PID, nil
		}
		p.ExitCode = -1
		return -1, p.PID, err
	}

	p.ExitCode = 0
	return 0, p.PID, nil
}

// Kill sends an OS signal to terminate the process.
func (p *Process) Kill(sig os.Signal) error {
	p.Mu.Lock()
	defer p.Mu.Unlock()

	if p.Exited || p.Cmd == nil || p.Cmd.Process == nil {
		return nil
	}

	if sig == nil {
		sig = syscall.SIGTERM
	}

	err := p.Cmd.Process.Signal(sig)
	if err != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}
