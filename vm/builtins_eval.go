package vm

import (
	"fmt"

	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
)

// EvalSession maintains state across dynamic eval calls.
type EvalSession struct {
	SymbolTable *compiler.SymbolTable
	Constants   []object.Object
	Globals     []object.Object
}

func (s *EvalSession) Type() object.ObjectType { return object.ObjectType("EVAL_SESSION") }
func (s *EvalSession) Inspect() string         { return "eval_session()" }

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"eval": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("eval expected 1 or 2 arguments (code string, [session]), got %d", len(args))}
				}
				str, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("eval argument must be STRING, got %s", args[0].Type())}
				}
				l := lexer.New(str.Value)
				p := parser.New(l)
				program := p.ParseProgram()
				if len(p.Errors()) > 0 {
					return &object.Error{Message: fmt.Sprintf("eval syntax error: %s", p.Errors()[0])}
				}

				var comp *compiler.Compiler
				var sess *EvalSession
				if len(args) == 2 {
					var okSess bool
					sess, okSess = args[1].(*EvalSession)
					if !okSess {
						return &object.Error{Message: fmt.Sprintf("eval session must be EVAL_SESSION, got %s", args[1].Type())}
					}
					comp = compiler.NewWithState(sess.SymbolTable, sess.Constants)
				} else {
					comp = compiler.New()
				}

				err := comp.Compile(program)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("eval compile error: %s", err)}
				}

				bytecode := comp.Bytecode()
				var childVM *VM
				if sess != nil {
					sess.Constants = bytecode.Constants
					childVM = NewWithGlobalsStore(bytecode, sess.Globals)
				} else {
					childVM = New(bytecode)
				}

				err = childVM.Run()
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("eval runtime error: %s", err)}
				}

				res := childVM.LastPoppedStackElem()
				if res == nil {
					return Null
				}
				return res
			},
		},
		"create_session": {
			Fn: func(args ...object.Object) object.Object {
				symTable := compiler.NewSymbolTable()
				for i, v := range compiler.Builtins {
					symTable.DefineBuiltin(i, v.Name)
				}
				return &EvalSession{
					SymbolTable: symTable,
					Constants:   []object.Object{},
					Globals:     make([]object.Object, GlobalsSize),
				}
			},
		},
		"session_vars": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("session_vars requires 1 argument, got %d", len(args))}
				}
				sess, ok := args[0].(*EvalSession)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument must be EVAL_SESSION, got %s", args[0].Type())}
				}
				pairs := make(map[object.HashKey]object.HashPair)
				for name, sym := range sess.SymbolTable.Store() {
					if sym.Scope == compiler.GlobalScope && sym.Index < len(sess.Globals) {
						val := sess.Globals[sym.Index]
						if val != nil {
							key := &object.String{Value: name}
							pairs[key.HashKey()] = object.HashPair{Key: key, Value: val}
						}
					}
				}
				return &object.Hash{Pairs: pairs}
			},
		},
		"disasm": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("disasm requires 1 argument, got %d", len(args))}
				}
				switch target := args[0].(type) {
				case *object.String:
					l := lexer.New(target.Value)
					p := parser.New(l)
					prog := p.ParseProgram()
					if len(p.Errors()) > 0 {
						return &object.Error{Message: fmt.Sprintf("syntax error: %s", p.Errors()[0])}
					}
					comp := compiler.New()
					if err := comp.Compile(prog); err != nil {
						return &object.Error{Message: fmt.Sprintf("compile error: %s", err)}
					}
					return &object.String{Value: compiler.FormatBytecode(comp.Bytecode())}
				case *object.Closure:
					if target.Fn != nil {
						bc := &compiler.Bytecode{
							Instructions: target.Fn.Instructions,
							Constants:    nil,
						}
						return &object.String{Value: compiler.FormatInstructions(bc.Instructions, nil, "")}
					}
					return &object.String{Value: "closure(empty)"}
				default:
					return &object.Error{Message: fmt.Sprintf("disasm requires STRING or CLOSURE, got %s", target.Type())}
				}
			},
		},
		"implements": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("implements expected 2 arguments (object, required_methods), got %d", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return False
				}
				var requiredMethods []string
				switch req := args[1].(type) {
				case *object.String:
					requiredMethods = append(requiredMethods, req.Value)
				case *object.Array:
					req.Mu.RLock()
					for _, elem := range req.Elements {
						if s, ok := elem.(*object.String); ok {
							requiredMethods = append(requiredMethods, s.Value)
						} else {
							req.Mu.RUnlock()
							return &object.Error{Message: fmt.Sprintf("implements required_methods array element must be STRING, got %s", elem.Type())}
						}
					}
					req.Mu.RUnlock()
				default:
					return &object.Error{Message: fmt.Sprintf("implements second argument must be STRING or ARRAY of strings, got %s", args[1].Type())}
				}

				hash.Mu.RLock()
				defer hash.Mu.RUnlock()
				for _, method := range requiredMethods {
					mKey := (&object.String{Value: method}).HashKey()
					if _, found := hash.Pairs[mKey]; found {
						continue
					}
					foundInEmbed := false
					if embedPair, hasEmbeds := hash.Pairs[(&object.String{Value: "__embeds__"}).HashKey()]; hasEmbeds {
						if embedArr, isArr := embedPair.Value.(*object.Array); isArr {
							embedArr.Mu.RLock()
							for _, embObj := range embedArr.Elements {
								if embHash, isH := embObj.(*object.Hash); isH {
									embHash.Mu.RLock()
									if _, hasM := embHash.Pairs[mKey]; hasM {
										foundInEmbed = true
										embHash.Mu.RUnlock()
										break
									}
									embHash.Mu.RUnlock()
								}
							}
							embedArr.Mu.RUnlock()
						}
					}
					if !foundInEmbed {
						return False
					}
				}
				return True
			},
		},
	})
}
