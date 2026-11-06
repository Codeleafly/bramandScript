package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"bramandscript/object"
)

var (
	builtinRegistry   = make(map[string]*object.Builtin)
	builtinRegistryMu sync.RWMutex

	programArgs     = []string{}
	stdinReader     = bufio.NewReader(os.Stdin)
	stdinReaderLock sync.Mutex
)

// RegisterBuiltin registers a single builtin function into the global VM registry.
func RegisterBuiltin(name string, b *object.Builtin) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	builtinRegistry[name] = b
}

// RegisterBuiltins registers a batch of builtins into the global VM registry.
func RegisterBuiltins(entries map[string]*object.Builtin) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	for k, v := range entries {
		builtinRegistry[k] = v
	}
}

// RegisterBuiltinAliases registers a single builtin under multiple aliases.
func RegisterBuiltinAliases(b *object.Builtin, names ...string) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	for _, name := range names {
		builtinRegistry[name] = b
	}
}

// getBuiltinByName retrieves a builtin object by its identifier name.
func getBuiltinByName(name string) *object.Builtin {
	builtinRegistryMu.RLock()
	b, ok := builtinRegistry[name]
	builtinRegistryMu.RUnlock()
	if ok {
		return b
	}
	return nil
}

// SetProgramArgs configures argv / args accessible from BramandScript.
func SetProgramArgs(args []string) {
	programArgs = make([]string, len(args))
	copy(programArgs, args)
}

// GetProgramArgs retrieves a copy of current program args.
func GetProgramArgs() []string {
	res := make([]string, len(programArgs))
	copy(res, programArgs)
	return res
}

func readLineFromStdin(args []object.Object) object.Object {
	if len(args) > 0 {
		if s, ok := args[0].(*object.String); ok {
			StdoutWrite(s.Value)
		} else {
			StdoutWrite(args[0].Inspect())
		}
	}
	if IsTermReaderActive() {
		return &object.String{Value: ReadLineFromTermChan()}
	}
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	text, err := stdinReader.ReadString('\n')
	if err == io.EOF && len(text) == 0 {
		return Null
	}
	return &object.String{Value: strings.TrimRight(text, "\r\n")}
}

// ResetStdin resets the standard input buffer (used for testing).
func ResetStdin() {
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	stdinReader = bufio.NewReader(os.Stdin)
}

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"len": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.String:
					return &object.Integer{Value: int64(len([]rune(arg.Value)))}
				case *object.Array:
					arg.Mu.RLock()
					defer arg.Mu.RUnlock()
					return &object.Integer{Value: int64(len(arg.Elements))}
				case *object.Hash:
					arg.Mu.RLock()
					defer arg.Mu.RUnlock()
					return &object.Integer{Value: int64(len(arg.Pairs))}
				case *object.Bytes:
					arg.Mu.RLock()
					defer arg.Mu.RUnlock()
					return &object.Integer{Value: int64(len(arg.Value))}
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Type())}
				}
			},
		},
		"print": {
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						StdoutWrite(" ")
					}
					if s, ok := arg.(*object.String); ok {
						StdoutWrite(s.Value)
						continue
					}
					StdoutWrite(arg.Inspect())
				}
				return Null
			},
		},
		"println": {
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						StdoutWrite(" ")
					}
					if s, ok := arg.(*object.String); ok {
						StdoutWrite(s.Value)
						continue
					}
					StdoutWrite(arg.Inspect())
				}
				StdoutWrite("\n")
				return Null
			},
		},
		"print_raw": {
			Fn: func(args ...object.Object) object.Object {
				for _, arg := range args {
					if s, ok := arg.(*object.String); ok {
						StdoutWrite(s.Value)
						continue
					}
					StdoutWrite(arg.Inspect())
				}
				return Null
			},
		},
		"ensure_newline": {
			Fn: func(args ...object.Object) object.Object {
				if EnsureStdoutNewline() {
					return True
				}
				return False
			},
		},
		"exit": {
			Fn: func(args ...object.Object) object.Object {
				code := 0
				if len(args) > 0 {
					if intObj, ok := args[0].(*object.Integer); ok {
						code = int(intObj.Value)
					}
				}
				if IsTestRunnerActive() {
					if code != 0 {
						return &object.Error{Message: fmt.Sprintf("exit called with code %d during test run", code)}
					}
					return Null
				}
				os.Exit(code)
				return Null
			},
		},
		"args": {
			Fn: func(args ...object.Object) object.Object {
				elements := make([]object.Object, len(programArgs))
				for i, a := range programArgs {
					elements[i] = &object.String{Value: a}
				}
				return &object.Array{Elements: elements}
			},
		},
		"argv": {
			Fn: func(args ...object.Object) object.Object {
				elements := make([]object.Object, len(programArgs))
				for i, a := range programArgs {
					elements[i] = &object.String{Value: a}
				}
				return &object.Array{Elements: elements}
			},
		},
	})
}
