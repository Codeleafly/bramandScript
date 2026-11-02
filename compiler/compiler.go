package compiler

import (
	"bramandscript/ast"
	"bramandscript/code"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CompilerError struct {
	Line    int
	Col     int
	Message string
}

func (ce *CompilerError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", ce.Line, ce.Col, ce.Message)
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
	sourceLines         []int
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
	SourceMap    []int
}

type LoopContext struct {
	continuePos   int
	breakJumps    []int
	continueJumps []int
}

type Compiler struct {
	constants       []object.Object
	symbolTable     *SymbolTable
	scopes          []CompilationScope
	scopeIndex      int
	loops           []LoopContext
	iterCount       int
	currentLine     int
	importedModules map[string]bool
	BaseDir         string
}

func (c *Compiler) SetBaseDir(dir string) {
	c.BaseDir = dir
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		sourceLines:         []int{},
	}

	symbolTable := NewSymbolTable()
	for i, v := range Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	return &Compiler{
		constants:       []object.Object{},
		symbolTable:     symbolTable,
		scopes:          []CompilationScope{mainScope},
		scopeIndex:      0,
		currentLine:     1,
		importedModules: make(map[string]bool),
	}
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	c := New()
	c.symbolTable = s
	c.constants = constants
	return c
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		sourceLines:         []int{},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() (code.Instructions, []int) {
	instructions := c.currentInstructions()
	sourceLines := c.scopes[c.scopeIndex].sourceLines

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions, sourceLines
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		if fnLit, ok := node.Expression.(*ast.FunctionLiteral); ok && fnLit.Name != "" {
			symbol := c.symbolTable.Define(fnLit.Name, false)
			err := c.Compile(node.Expression)
			if err != nil {
				return err
			}
			if symbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, symbol.Index)
			} else {
				c.emit(code.OpSetLocal, symbol.Index)
			}
			c.emit(code.OpNull)
			return nil
		}
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)

	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.LetStatement:
		symbol := c.symbolTable.Define(node.Name.Value, false)
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.ConstStatement:
		symbol := c.symbolTable.Define(node.Name.Value, true)
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.ReassignStatement:
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot assign to undefined variable '%s'", node.Name.Value)}
		}
		if symbol.IsConst {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot reassign to constant variable '%s'", node.Name.Value)}
		}

		if node.Operator == "=" {
			err := c.Compile(node.Value)
			if err != nil {
				return err
			}
		} else {
			c.loadSymbol(symbol)
			err := c.Compile(node.Value)
			if err != nil {
				return err
			}
			switch node.Operator {
			case "+=":
				c.emit(code.OpAdd)
			case "-=":
				c.emit(code.OpSub)
			case "*=":
				c.emit(code.OpMul)
			case "/=":
				c.emit(code.OpDiv)
			default:
				return fmt.Errorf("unsupported compound operator %s", node.Operator)
			}
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else if symbol.Scope == LocalScope {
			c.emit(code.OpSetLocal, symbol.Index)
		} else if symbol.Scope == FreeScope {
			c.emit(code.OpSetFree, symbol.Index)
		} else {
			return fmt.Errorf("cannot assign to %s variable %s", symbol.Scope, node.Name.Value)
		}

	case *ast.IndexAssignStatement:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		err = c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(code.OpSetIndex)

	case *ast.ForInStatement:
		c.currentLine = node.Token.Line
		c.iterCount++
		iterName := fmt.Sprintf("__iter_%d", c.iterCount)
		idxName := fmt.Sprintf("__idx_%d", c.iterCount)

		// 1. Compile iterable & store in iterName
		err := c.Compile(node.Iterable)
		if err != nil {
			return err
		}
		iterSym := c.symbolTable.Define(iterName, false)
		if iterSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, iterSym.Index)
		} else {
			c.emit(code.OpSetLocal, iterSym.Index)
		}

		// 2. Initialize idx = 0
		c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: 0}))
		idxSym := c.symbolTable.Define(idxName, false)
		if idxSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, idxSym.Index)
		} else {
			c.emit(code.OpSetLocal, idxSym.Index)
		}

		// 3. Define item and optional index variable in current scope
		itemSym := c.symbolTable.Define(node.Item.Value, false)
		var userIdxSym Symbol
		if node.Index != nil {
			userIdxSym = c.symbolTable.Define(node.Index.Value, false)
		}

		// 4. Loop start condition: len(iter) > idx
		loopStart := len(c.currentInstructions())
		c.loops = append(c.loops, LoopContext{continuePos: -1, breakJumps: []int{}, continueJumps: []int{}})

		lenSym, ok := c.symbolTable.Resolve("len")
		if !ok {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "builtin 'len' not found for loop iteration"}
		}
		c.loadSymbol(lenSym)
		c.loadSymbol(iterSym)
		c.emit(code.OpCall, 1)

		c.loadSymbol(idxSym)
		c.emit(code.OpGreaterThan)

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		// 5. Fetch item: iter[idx]
		c.loadSymbol(iterSym)
		c.loadSymbol(idxSym)
		c.emit(code.OpIndex)
		if itemSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, itemSym.Index)
		} else {
			c.emit(code.OpSetLocal, itemSym.Index)
		}

		if node.Index != nil {
			c.loadSymbol(idxSym)
			if userIdxSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, userIdxSym.Index)
			} else {
				c.emit(code.OpSetLocal, userIdxSym.Index)
			}
		}

		// 6. Compile body
		err = c.Compile(node.Body)
		if err != nil {
			return err
		}

		// 7. Increment idx (continue jump target)
		continuePos := len(c.currentInstructions())
		c.loadSymbol(idxSym)
		c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: 1}))
		c.emit(code.OpAdd)
		if idxSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, idxSym.Index)
		} else {
			c.emit(code.OpSetLocal, idxSym.Index)
		}

		c.emit(code.OpJump, loopStart)

		// 8. Backpatch break, continue, and loop-exit jumps
		afterLoopPos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterLoopPos)

		currentLoop := c.loops[len(c.loops)-1]
		c.loops = c.loops[:len(c.loops)-1]

		for _, bPos := range currentLoop.breakJumps {
			c.changeOperand(bPos, afterLoopPos)
		}
		for _, cPos := range currentLoop.continueJumps {
			c.changeOperand(cPos, continuePos)
		}

	case *ast.StructStatement:
		c.currentLine = node.Token.Line
		c.enterScope()

		for _, f := range node.Fields {
			c.symbolTable.Define(f.Value, false)
		}

		for _, m := range node.Methods {
			err := c.Compile(m)
			if err != nil {
				return err
			}
			fnSym := c.symbolTable.Define(m.Name, false)
			c.emit(code.OpSetLocal, fnSym.Index)
		}

		numPairs := 1 + len(node.Fields) + len(node.Methods)

		c.emit(code.OpConstant, c.addConstant(&object.String{Value: "__struct__"}))
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Name.Value}))

		for _, f := range node.Fields {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: f.Value}))
			sym, _ := c.symbolTable.Resolve(f.Value)
			c.loadSymbol(sym)
		}

		for _, m := range node.Methods {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: m.Name}))
			sym, _ := c.symbolTable.Resolve(m.Name)
			c.loadSymbol(sym)
		}

		c.emit(code.OpHash, numPairs*2)
		c.emit(code.OpReturnValue)

		numLocals := c.symbolTable.numDefinitions
		instructions, sourceLines := c.leaveScope()

		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Fields),
			SourceMap:     sourceLines,
		}

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, 0)

		structSym := c.symbolTable.Define(node.Name.Value, false)
		if structSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, structSym.Index)
		} else {
			c.emit(code.OpSetLocal, structSym.Index)
		}

	case *ast.ImportStatement:
		c.currentLine = node.Token.Line
		filePath, err := c.resolveModulePath(node.Path)
		if err != nil {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot import module '%s': %s", node.Path, err)}
		}

		if !c.importedModules[filePath] {
			c.importedModules[filePath] = true

			data, err := os.ReadFile(filePath)
			if err != nil {
				return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot import module '%s': %s", node.Path, err)}
			}

			l := lexer.New(string(data))
			p := parser.New(l)
			moduleProgram := p.ParseProgram()
			if len(p.Errors()) > 0 {
				return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("syntax error in module '%s': %s", node.Path, p.Errors()[0])}
			}

			prevBaseDir := c.BaseDir
			c.BaseDir = filepath.Dir(filePath)

			if node.Alias == "" {
				for _, stmt := range moduleProgram.Statements {
					err := c.Compile(stmt)
					if err != nil {
						c.BaseDir = prevBaseDir
						return err
					}
				}
				c.BaseDir = prevBaseDir
			} else {
				exportedNames := []string{}
				for _, stmt := range moduleProgram.Statements {
					switch s := stmt.(type) {
					case *ast.LetStatement:
						exportedNames = append(exportedNames, s.Name.Value)
					case *ast.ExpressionStatement:
						if fnLit, ok := s.Expression.(*ast.FunctionLiteral); ok && fnLit.Name != "" {
							exportedNames = append(exportedNames, fnLit.Name)
						}
					case *ast.StructStatement:
						exportedNames = append(exportedNames, s.Name.Value)
					}
					err := c.Compile(stmt)
					if err != nil {
						c.BaseDir = prevBaseDir
						return err
					}
				}
				c.BaseDir = prevBaseDir

				for _, name := range exportedNames {
					c.emit(code.OpConstant, c.addConstant(&object.String{Value: name}))
					sym, _ := c.symbolTable.Resolve(name)
					c.loadSymbol(sym)
				}
				c.emit(code.OpHash, len(exportedNames)*2)

				aliasSym := c.symbolTable.Define(node.Alias, false)
				if aliasSym.Scope == GlobalScope {
					c.emit(code.OpSetGlobal, aliasSym.Index)
				} else {
					c.emit(code.OpSetLocal, aliasSym.Index)
				}
			}
		}

	case *ast.InfixExpression:
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThan)
			return nil
		}

		if node.Operator == "<=" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThanEqual)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "%":
			c.emit(code.OpMod)
		case "**":
			c.emit(code.OpPow)
		case ">":
			c.emit(code.OpGreaterThan)
		case ">=":
			c.emit(code.OpGreaterThanEqual)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		case "&&":
			c.emit(code.OpAnd)
		case "||":
			c.emit(code.OpOr)
		case "<-":
			c.emit(code.OpSend)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		case "<-":
			c.emit(code.OpReceive)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		} else if !c.lastInstructionIs(code.OpReturnValue) && !c.lastInstructionIs(code.OpReturn) {
			c.emit(code.OpNull)
		}

		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			} else if !c.lastInstructionIs(code.OpReturnValue) && !c.lastInstructionIs(code.OpReturn) {
				c.emit(code.OpNull)
			}
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.WhileStatement:
		loopStart := len(c.currentInstructions())
		c.loops = append(c.loops, LoopContext{continuePos: loopStart, breakJumps: []int{}, continueJumps: []int{}})

		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Body)
		if err != nil {
			return err
		}

		c.emit(code.OpJump, loopStart)

		afterBodyPos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterBodyPos)

		currentLoop := c.loops[len(c.loops)-1]
		c.loops = c.loops[:len(c.loops)-1]
		for _, breakPos := range currentLoop.breakJumps {
			c.changeOperand(breakPos, afterBodyPos)
		}

	case *ast.BreakStatement:
		if len(c.loops) == 0 {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "'break' statement outside of loop"}
		}
		breakPos := c.emit(code.OpJump, 9999)
		c.loops[len(c.loops)-1].breakJumps = append(c.loops[len(c.loops)-1].breakJumps, breakPos)

	case *ast.ContinueStatement:
		if len(c.loops) == 0 {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "'continue' statement outside of loop"}
		}
		curr := &c.loops[len(c.loops)-1]
		if curr.continuePos != -1 {
			c.emit(code.OpJump, curr.continuePos)
		} else {
			pos := c.emit(code.OpJump, 9999)
			curr.continueJumps = append(curr.continueJumps, pos)
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))

	case *ast.FloatLiteral:
		floatVal := &object.Float{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(floatVal))

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.BooleanLiteral:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.NullLiteral:
		c.emit(code.OpNull)

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("undefined variable '%s'", node.Value)}
		}
		c.loadSymbol(symbol)

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpArray, len(node.Elements))

	case *ast.HashLiteral:
		for k, v := range node.Pairs {
			err := c.Compile(k)
			if err != nil {
				return err
			}
			err = c.Compile(v)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpHash, len(node.Pairs)*2)

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(code.OpIndex)

	case *ast.DotExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		key := &object.String{Value: node.Field.Value}
		c.emit(code.OpConstant, c.addConstant(key))
		c.emit(code.OpIndex)

	case *ast.MatchExpression:
		c.currentLine = node.Token.Line
		c.iterCount++
		subjName := fmt.Sprintf("__match_subj_%d", c.iterCount)

		err := c.Compile(node.Subject)
		if err != nil {
			return err
		}
		subjSym := c.symbolTable.Define(subjName, false)
		if subjSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, subjSym.Index)
		} else {
			c.emit(code.OpSetLocal, subjSym.Index)
		}

		endJumps := []int{}
		var defaultClause *ast.MatchClause

		for _, clause := range node.Clauses {
			if len(clause.Values) == 0 {
				defaultClause = clause
				continue
			}

			for i, val := range clause.Values {
				c.loadSymbol(subjSym)
				err := c.Compile(val)
				if err != nil {
					return err
				}
				c.emit(code.OpEqual)
				if i > 0 {
					c.emit(code.OpOr)
				}
			}

			nextCaseJump := c.emit(code.OpJumpNotTruthy, 9999)

			err := c.Compile(clause.Body)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}

			endJump := c.emit(code.OpJump, 9999)
			endJumps = append(endJumps, endJump)

			c.changeOperand(nextCaseJump, len(c.currentInstructions()))
		}

		if defaultClause != nil {
			err := c.Compile(defaultClause.Body)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		} else {
			c.emit(code.OpNull)
		}

		afterMatchPos := len(c.currentInstructions())
		for _, ej := range endJumps {
			c.changeOperand(ej, afterMatchPos)
		}

	case *ast.FunctionLiteral:
		c.enterScope()

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value, false)
		}

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}
		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}

		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions, sourceLines := c.leaveScope()

		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}

		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
			SourceMap:     sourceLines,
		}

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}
		c.emit(code.OpReturnValue)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}
		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpCall, len(node.Arguments))

	case *ast.SpawnExpression:
		c.currentLine = node.Token.Line
		if call, ok := node.Call.(*ast.CallExpression); ok {
			err := c.Compile(call.Function)
			if err != nil {
				return err
			}
			for _, a := range call.Arguments {
				err := c.Compile(a)
				if err != nil {
					return err
				}
			}
			c.emit(code.OpSpawn, len(call.Arguments))
		} else {
			err := c.Compile(node.Call)
			if err != nil {
				return err
			}
			c.emit(code.OpSpawn, 0)
		}
	}

	return nil
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	}
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)
	c.setLastInstruction(op, pos)
	line := c.currentLine
	if line < 1 {
		line = 1
	}
	for i := 0; i < len(ins); i++ {
		c.scopes[c.scopeIndex].sourceLines = append(c.scopes[c.scopeIndex].sourceLines, line)
	}
	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)
	c.scopes[c.scopeIndex].instructions = updatedInstructions
	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}
	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	c.scopes[c.scopeIndex].instructions = c.scopes[c.scopeIndex].instructions[:last.Position]
	if len(c.scopes[c.scopeIndex].sourceLines) >= last.Position {
		c.scopes[c.scopeIndex].sourceLines = c.scopes[c.scopeIndex].sourceLines[:last.Position]
	}
	c.scopes[c.scopeIndex].lastInstruction = c.scopes[c.scopeIndex].previousInstruction
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))
	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	newInstruction := code.Make(op, operand)
	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
		SourceMap:    c.scopes[c.scopeIndex].sourceLines,
	}
}

func (c *Compiler) resolveModulePath(importPath string) (string, error) {
	candidates := []string{}

	// 1. Check relative to c.BaseDir if specified
	if c.BaseDir != "" {
		if strings.HasSuffix(importPath, ".bmd") {
			candidates = append(candidates, filepath.Join(c.BaseDir, importPath))
		} else {
			candidates = append(candidates,
				filepath.Join(c.BaseDir, importPath+".bmd"),
				filepath.Join(c.BaseDir, importPath, "index.bmd"))
		}
	}

	// 2. Check local path directly (relative to current working directory)
	if strings.HasSuffix(importPath, ".bmd") {
		candidates = append(candidates, importPath)
	} else {
		candidates = append(candidates, importPath+".bmd", filepath.Join(importPath, "index.bmd"))
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			abs, err := filepath.Abs(cand)
			if err == nil {
				return abs, nil
			}
			return cand, nil
		}
	}

	// 2. Centralized package cache check ($HOME/.bramand_libs)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		libsDir := filepath.Join(homeDir, ".bramand_libs")
		libCandidates := []string{
			filepath.Join(libsDir, importPath, "index.bmd"),
			filepath.Join(libsDir, importPath+".bmd"),
			filepath.Join(libsDir, importPath),
		}
		for _, cand := range libCandidates {
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				return cand, nil
			}
		}
	}

	return "", fmt.Errorf("module '%s' not found locally or in ~/.bramand_libs", importPath)
}
