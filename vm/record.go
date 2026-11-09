package vm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"bramandscript/code"
)

// StepCheckpoint captures an execution snapshot at a single VM step.
type StepCheckpoint struct {
	Step     int      `json:"step"`
	IP       int      `json:"ip"`
	Opcode   string   `json:"opcode"`
	Stack    []string `json:"stack"`
	Locals   []string `json:"locals"`
	Function string   `json:"function"`
	Line     int      `json:"line"`
}

// ExecutionTrace stores a sequence of step checkpoints captured during VM execution.
type ExecutionTrace struct {
	SourceFile string           `json:"source_file"`
	StartTime  string           `json:"start_time"`
	TotalSteps int              `json:"total_steps"`
	Steps      []StepCheckpoint `json:"steps"`
}

// Save writes the trace to a file in JSON format.
func (t *ExecutionTrace) Save(filepath string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}

// LoadTrace reads an ExecutionTrace from a file.
func LoadTrace(filepath string) (*ExecutionTrace, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var trace ExecutionTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("corrupted trace file: %w", err)
	}
	return &trace, nil
}

// ExecutionRecorder records VM step checkpoints during execution.
type ExecutionRecorder struct {
	Trace    *ExecutionTrace
	MaxSteps int
}

// NewExecutionRecorder creates an ExecutionRecorder for a target source file.
func NewExecutionRecorder(sourceFile string) *ExecutionRecorder {
	return &ExecutionRecorder{
		Trace: &ExecutionTrace{
			SourceFile: sourceFile,
			StartTime:  time.Now().Format(time.RFC3339),
			Steps:      make([]StepCheckpoint, 0, 1024),
		},
		MaxSteps: 50000,
	}
}

// RecordStep captures a checkpoint at the current VM step.
func (rec *ExecutionRecorder) RecordStep(vm *VM, op code.Opcode, ip int) {
	if rec == nil || len(rec.Trace.Steps) >= rec.MaxSteps {
		return
	}

	frame := vm.currentFrame()
	fnName := "<main>"
	line := 0
	if frame != nil && frame.cl != nil && frame.cl.Fn != nil {
		if frame.cl.Fn.Name != "" {
			fnName = frame.cl.Fn.Name
		}
		if ip < len(frame.cl.Fn.SourceMap) {
			line = frame.cl.Fn.SourceMap[ip]
		}
	}

	// Capture stack snapshot (top 8 elements)
	var stackSnap []string
	stackStart := 0
	if vm.sp > 8 {
		stackStart = vm.sp - 8
	}
	for i := stackStart; i < vm.sp; i++ {
		if vm.stack[i] != nil {
			stackSnap = append(stackSnap, vm.stack[i].Inspect())
		}
	}

	// Capture locals snapshot (first 8 locals)
	var localsSnap []string
	if frame != nil {
		numLocals := 0
		if frame.cl != nil && frame.cl.Fn != nil {
			numLocals = frame.cl.Fn.NumLocals
		}
		if numLocals > 8 {
			numLocals = 8
		}
		base := frame.basePointer
		for i := 0; i < numLocals && base+i < len(vm.stack); i++ {
			if vm.stack[base+i] != nil {
				localsSnap = append(localsSnap, vm.stack[base+i].Inspect())
			}
		}
	}

	opName := fmt.Sprintf("Op(%d)", op)
	def, err := code.Lookup(byte(op))
	if err == nil {
		opName = def.Name
	}

	chk := StepCheckpoint{
		Step:     len(rec.Trace.Steps) + 1,
		IP:       ip,
		Opcode:   opName,
		Stack:    stackSnap,
		Locals:   localsSnap,
		Function: fnName,
		Line:     line,
	}

	rec.Trace.Steps = append(rec.Trace.Steps, chk)
	rec.Trace.TotalSteps = len(rec.Trace.Steps)
}

// ReplayDebugger provides time-travel execution replay and inspection.
type ReplayDebugger struct {
	Trace        *ExecutionTrace
	CurrentIndex int
}

// NewReplayDebugger constructs a ReplayDebugger from an ExecutionTrace.
func NewReplayDebugger(trace *ExecutionTrace) *ReplayDebugger {
	return &ReplayDebugger{
		Trace:        trace,
		CurrentIndex: 0,
	}
}

// Step advances 1 step forward in the execution history.
func (d *ReplayDebugger) Step() *StepCheckpoint {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		return nil
	}
	if d.CurrentIndex < len(d.Trace.Steps)-1 {
		d.CurrentIndex++
	}
	return &d.Trace.Steps[d.CurrentIndex]
}

// StepBack moves 1 step backward in time.
func (d *ReplayDebugger) StepBack() *StepCheckpoint {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		return nil
	}
	if d.CurrentIndex > 0 {
		d.CurrentIndex--
	}
	return &d.Trace.Steps[d.CurrentIndex]
}

// Goto jumps directly to a specific step number.
func (d *ReplayDebugger) Goto(stepNum int) *StepCheckpoint {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		return nil
	}
	idx := stepNum - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(d.Trace.Steps) {
		idx = len(d.Trace.Steps) - 1
	}
	d.CurrentIndex = idx
	return &d.Trace.Steps[d.CurrentIndex]
}

// Current returns the current step checkpoint.
func (d *ReplayDebugger) Current() *StepCheckpoint {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		return nil
	}
	if d.CurrentIndex >= len(d.Trace.Steps) {
		d.CurrentIndex = len(d.Trace.Steps) - 1
	}
	return &d.Trace.Steps[d.CurrentIndex]
}

// InspectHistory returns the last N checkpoints up to the current step.
func (d *ReplayDebugger) InspectHistory(limit int) []StepCheckpoint {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	start := d.CurrentIndex - limit + 1
	if start < 0 {
		start = 0
	}
	end := d.CurrentIndex + 1
	if end > len(d.Trace.Steps) {
		end = len(d.Trace.Steps)
	}

	result := make([]StepCheckpoint, end-start)
	copy(result, d.Trace.Steps[start:end])
	return result
}

// RunInteractive starts an interactive replay REPL session.
func (d *ReplayDebugger) RunInteractive(in io.Reader, out io.Writer) error {
	if d.Trace == nil || len(d.Trace.Steps) == 0 {
		fmt.Fprintln(out, "[ERROR] Empty trace, nothing to replay.")
		return nil
	}

	fmt.Fprintf(out, "✦ Bramand Time-Travel Replay Debugger (Trace: %s, Total Steps: %d)\n", d.Trace.SourceFile, d.Trace.TotalSteps)
	fmt.Fprintln(out, "Commands: [s]tep, [b]ack/step_back, [p]rint/inspect, [h]istory, [g]oto <step>, [c]ontinue, [q]uit")
	fmt.Fprintln(out, "──────────────────────────────────────────────────────────────────────────")

	scanner := bufio.NewScanner(in)
	d.printCurrent(out)

	for {
		fmt.Fprintf(out, "replay[step %d/%d]> ", d.CurrentIndex+1, d.Trace.TotalSteps)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			d.Step()
			d.printCurrent(out)
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "s", "step":
			d.Step()
			d.printCurrent(out)

		case "b", "back", "step_back":
			d.StepBack()
			d.printCurrent(out)

		case "p", "print", "inspect":
			cur := d.Current()
			if cur != nil {
				fmt.Fprintf(out, "Step: %d | Function: %s | Line: %d | IP: %d | Opcode: %s\n", cur.Step, cur.Function, cur.Line, cur.IP, cur.Opcode)
				fmt.Fprintf(out, "  Stack (%d items): %v\n", len(cur.Stack), cur.Stack)
				fmt.Fprintf(out, "  Locals (%d items): %v\n", len(cur.Locals), cur.Locals)
			}

		case "h", "history", "inspect_history":
			history := d.InspectHistory(10)
			fmt.Fprintln(out, "Recent Checkpoints:")
			for _, chk := range history {
				marker := "  "
				if chk.Step == d.CurrentIndex+1 {
					marker = "❯ "
				}
				fmt.Fprintf(out, "%sStep %-5d | IP: %-4d | %-16s | Stack: %v\n", marker, chk.Step, chk.IP, chk.Opcode, chk.Stack)
			}

		case "g", "goto":
			if len(parts) >= 2 {
				var target int
				fmt.Sscanf(parts[1], "%d", &target)
				d.Goto(target)
				d.printCurrent(out)
			} else {
				fmt.Fprintln(out, "Usage: goto <step_number>")
			}

		case "c", "continue":
			d.CurrentIndex = d.Trace.TotalSteps - 1
			d.printCurrent(out)

		case "q", "quit", "exit":
			fmt.Fprintln(out, "Exiting replay debugger.")
			return nil

		default:
			fmt.Fprintf(out, "Unknown command: %s. Type s, b, p, h, g <step>, or q.\n", cmd)
		}
	}
	return nil
}

func (d *ReplayDebugger) printCurrent(out io.Writer) {
	cur := d.Current()
	if cur == nil {
		return
	}
	fmt.Fprintf(out, "❯ Step %-5d | %-16s | Func: %-12s | Line: %-3d | Stack: %v\n",
		cur.Step, cur.Opcode, cur.Function, cur.Line, cur.Stack)
}
