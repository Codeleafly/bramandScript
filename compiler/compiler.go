package compiler

import (
	"fmt"

	"bramandscript/ast"
	"bramandscript/code"
	"bramandscript/object"
	"bramandscript/pkg"
)

type CompilerError struct {
	File    string
	Line    int
	Col     int
	Message string
	Source  string
}

func (ce *CompilerError) Error() string {
	if ce.File != "" {
		return fmt.Sprintf("%s:%d:%d: %s", ce.File, ce.Line, ce.Col, ce.Message)
	}
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
	constants        []object.Object
	symbolTable      *SymbolTable
	scopes           []CompilationScope
	scopeIndex       int
	loops            []LoopContext
	iterCount        int
	destructCount    int
	currentLine      int
	importedModules  map[string]bool
	importingModules map[string]bool
	moduleExports    map[string][]string
	moduleSingletons map[string]Symbol
	compileErr       error
	BaseDir          string
	currentFile      string
	currentSource    string
	Optimize         bool

	currentContract         *object.PackageContract
	importedModulePaths     map[string]string
	importedModuleSources   map[string][]byte
	importedModuleManifests map[string]*pkg.Manifest
}

func (c *Compiler) ImportedModulePaths() map[string]string {
	return c.importedModulePaths
}

func (c *Compiler) ImportedModuleSources() map[string][]byte {
	return c.importedModuleSources
}

func (c *Compiler) ImportedModuleManifests() map[string]*pkg.Manifest {
	return c.importedModuleManifests
}

func (c *Compiler) SetBaseDir(dir string) {
	c.BaseDir = dir
}

func (c *Compiler) SetCurrentFile(file, source string) {
	c.currentFile = file
	c.currentSource = source
	if c.symbolTable != nil {
		c.symbolTable.CurrentFile = file
	}
}

func (c *Compiler) makeError(line, col int, format string, args ...interface{}) *CompilerError {
	return &CompilerError{
		File:    c.currentFile,
		Line:    line,
		Col:     col,
		Message: fmt.Sprintf(format, args...),
		Source:  c.currentSource,
	}
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
		constants:        []object.Object{},
		symbolTable:      symbolTable,
		scopes:           []CompilationScope{mainScope},
		scopeIndex:       0,
		currentLine:      1,
		importedModules:  make(map[string]bool),
		importingModules: make(map[string]bool),
		moduleExports:           make(map[string][]string),
		moduleSingletons:        make(map[string]Symbol),
		importedModulePaths:     make(map[string]string),
		importedModuleSources:   make(map[string][]byte),
		importedModuleManifests: make(map[string]*pkg.Manifest),
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

func (c *Compiler) enterBlockScope() {
	c.symbolTable = NewEnclosedBlockSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveBlockScope() {
	if c.symbolTable.Outer != nil {
		c.symbolTable = c.symbolTable.Outer
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	if c.compileErr != nil {
		return c.compileErr
	}
	if node == nil {
		return fmt.Errorf("compile error: unexpected nil AST node")
	}
	if c.Optimize {
		node = FoldConstants(node)
	}
	if handled, err := c.compileStatement(node); handled {
		if err != nil {
			return err
		}
		return c.compileErr
	}
	switch node := node.(type) {
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

		if node.Operator == "&&" {
			err := c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpDup)
			jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)
			c.emit(code.OpPop)

			err = c.Compile(node.Right)
			if err != nil {
				return err
			}

			afterRightPos := len(c.currentInstructions())
			c.changeOperand(jumpNotTruthyPos, afterRightPos)
			return nil
		}

		if node.Operator == "||" {
			err := c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpDup)
			jumpTruthyPos := c.emit(code.OpJumpTruthy, 9999)
			c.emit(code.OpPop)

			err = c.Compile(node.Right)
			if err != nil {
				return err
			}

			afterRightPos := len(c.currentInstructions())
			c.changeOperand(jumpTruthyPos, afterRightPos)
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
		case "&":
			c.emit(code.OpBitAnd)
		case "|":
			c.emit(code.OpBitOr)
		case "^":
			c.emit(code.OpBitXor)
		case "<<":
			c.emit(code.OpShiftLeft)
		case ">>":
			c.emit(code.OpShiftRight)
		case ">":
			c.emit(code.OpGreaterThan)
		case ">=":
			c.emit(code.OpGreaterThanEqual)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
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
		case "~":
			c.emit(code.OpBitNot)
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

	case *ast.TernaryExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		err = c.Compile(node.Alternative)
		if err != nil {
			return err
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.NullCoalesceExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		c.emit(code.OpDup)
		jumpNotNullPos := c.emit(code.OpJumpNotNull, 9999)
		c.emit(code.OpPop)

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		afterRightPos := len(c.currentInstructions())
		c.changeOperand(jumpNotNullPos, afterRightPos)


	case *ast.IntegerLiteral:
		if node.BigValue != nil {
			c.emit(code.OpConstant, c.addConstant(&object.BigInt{Value: node.BigValue}))
		} else {
			integer := &object.Integer{Value: node.Value}
			c.emit(code.OpConstant, c.addConstant(integer))
		}

	case *ast.FloatLiteral:
		floatVal := &object.Float{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(floatVal))

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.InterpolatedStringLiteral:
		c.currentLine = node.Token.Line
		if len(node.Parts) == 0 {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: ""}))
			return nil
		}
		if len(node.Parts) == 1 {
			if strLit, ok := node.Parts[0].(*ast.StringLiteral); ok {
				c.emit(code.OpConstant, c.addConstant(&object.String{Value: strLit.Value}))
				return nil
			}
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: ""}))
			err := c.Compile(node.Parts[0])
			if err != nil {
				return err
			}
			c.emit(code.OpAdd)
			return nil
		}

		startIdx := 0
		if strLit, ok := node.Parts[0].(*ast.StringLiteral); ok {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: strLit.Value}))
			startIdx = 1
		} else {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: ""}))
			startIdx = 0
		}

		for i := startIdx; i < len(node.Parts); i++ {
			err := c.Compile(node.Parts[i])
			if err != nil {
				return err
			}
			c.emit(code.OpAdd)
		}

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
			return c.makeError(node.Token.Line, node.Token.Col, "undefined variable '%s'", node.Value)
		}
		c.loadSymbol(symbol)

	case *ast.ArrayLiteral:
		if len(node.Elements) > 65535 {
			return c.makeError(node.Token.Line, node.Token.Col, "array literal exceeds maximum size of 65535 elements")
		}
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpArray, len(node.Elements))

	case *ast.SpreadExpression:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(code.OpSpread)

	case *ast.HashLiteral:
		if len(node.Entries) > 0 {
			if len(node.Entries)*2 > 65535 {
				return c.makeError(node.Token.Line, node.Token.Col, "hash literal exceeds maximum size of 32767 entries")
			}
			for _, entry := range node.Entries {
				err := c.Compile(entry.Key)
				if err != nil {
					return err
				}
				if entry.Value != nil {
					err = c.Compile(entry.Value)
					if err != nil {
						return err
					}
				} else {
					c.emit(code.OpNull)
				}
			}
			c.emit(code.OpHash, len(node.Entries)*2)
		} else {
			if len(node.Pairs)*2 > 65535 {
				return c.makeError(node.Token.Line, node.Token.Col, "hash literal exceeds maximum size of 32767 pairs")
			}
			for k, v := range node.Pairs {
				err := c.Compile(k)
				if err != nil {
					return err
				}
				if v != nil {
					err = c.Compile(v)
					if err != nil {
						return err
					}
				} else {
					c.emit(code.OpNull)
				}
			}
			c.emit(code.OpHash, len(node.Pairs)*2)
		}

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

	case *ast.SliceExpression:
		sliceBuiltin, ok := c.symbolTable.Resolve("slice")
		if !ok {
			return fmt.Errorf("undefined builtin 'slice'")
		}
		c.loadSymbol(sliceBuiltin)

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		if node.Start != nil {
			err = c.Compile(node.Start)
			if err != nil {
				return err
			}
		} else {
			c.emit(code.OpNull)
		}

		if node.End != nil {
			err = c.Compile(node.End)
			if err != nil {
				return err
			}
		} else {
			c.emit(code.OpNull)
		}

		if node.Step != nil {
			err = c.Compile(node.Step)
			if err != nil {
				return err
			}
			c.emit(code.OpCall, 4)
		} else {
			c.emit(code.OpCall, 3)
		}

	case *ast.DotExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		key := &object.String{Value: node.Field.Value}
		c.emit(code.OpConstant, c.addConstant(key))
		c.emit(code.OpIndex)

	case *ast.OptionalDotExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		c.emit(code.OpDup)
		jumpNullPos := c.emit(code.OpJumpNull, 9999)

		key := &object.String{Value: node.Field.Value}
		c.emit(code.OpConstant, c.addConstant(key))
		c.emit(code.OpIndex)

		afterChainPos := len(c.currentInstructions())
		c.changeOperand(jumpNullPos, afterChainPos)

	case *ast.OptionalIndexExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		c.emit(code.OpDup)
		jumpNullPos := c.emit(code.OpJumpNull, 9999)

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(code.OpIndex)

		afterChainPos := len(c.currentInstructions())
		c.changeOperand(jumpNullPos, afterChainPos)

	case *ast.OptionalCallExpression:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}
		c.emit(code.OpDup)
		jumpNullPos := c.emit(code.OpJumpNull, 9999)

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpCall, len(node.Arguments))

		afterCallPos := len(c.currentInstructions())
		c.changeOperand(jumpNullPos, afterCallPos)

	case *ast.PipeExpression:
		c.currentLine = node.Token.Line
		var call *ast.CallExpression
		switch right := node.Right.(type) {
		case *ast.CallExpression:
			newArgs := make([]ast.Expression, 0, len(right.Arguments)+1)
			newArgs = append(newArgs, node.Left)
			newArgs = append(newArgs, right.Arguments...)
			call = &ast.CallExpression{
				Token:     right.Token,
				Function:  right.Function,
				Arguments: newArgs,
			}
		default:
			call = &ast.CallExpression{
				Token:     node.Token,
				Function:  right,
				Arguments: []ast.Expression{node.Left},
			}
		}
		return c.Compile(call)

	case *ast.YieldExpression:
		c.currentLine = node.Token.Line
		var args []ast.Expression
		if node.Value != nil {
			args = []ast.Expression{node.Value}
		}
		call := &ast.CallExpression{
			Token:     node.Token,
			Function:  &ast.Identifier{Token: node.Token, Value: "yield"},
			Arguments: args,
		}
		return c.Compile(call)

	case *ast.MatchExpression:
		c.enterBlockScope()
		c.currentLine = node.Token.Line
		c.iterCount++
		subjName := fmt.Sprintf("__match_subj_%d", c.iterCount)

		err := c.Compile(node.Subject)
		if err != nil {
			c.leaveBlockScope()
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
		c.leaveBlockScope()

	case *ast.FunctionLiteral:
		outerLoops := c.loops
		c.loops = []LoopContext{}

		c.enterScope()

		if node.Name != "" {
			c.symbolTable.DefineFunctionName(node.Name)
		}

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value, false)
		}

		err := c.Compile(node.Body)
		if err != nil {
			c.loops = outerLoops
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
		if numLocals > 65535 {
			c.loops = outerLoops
			return c.makeError(node.Token.Line, node.Token.Col, "function scope exceeds maximum of 65535 local variables")
		}
		instructions, sourceLines := c.leaveScope()
		c.loops = outerLoops

		for _, s := range freeSymbols {
			switch s.Scope {
			case LocalScope:
				c.emit(code.OpGetLocalCell, s.Index)
			case FreeScope:
				c.emit(code.OpGetFreeCell, s.Index)
			case FunctionScope:
				c.emit(code.OpCurrentClosure)
			default:
				c.loadSymbol(s)
			}
		}

		freeNames := make([]string, len(freeSymbols))
		for i, s := range freeSymbols {
			freeNames[i] = s.Name
		}

		fnName := node.Name
		if fnName == "" {
			fnName = "<anonymous>"
		}

		compiledFn := &object.CompiledFunction{
			Name:          fnName,
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
			SourceMap:     sourceLines,
			FreeNames:     freeNames,
			Contract:      c.currentContract,
		}

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))


	case *ast.CallExpression:
		if optDot, ok := node.Function.(*ast.OptionalDotExpression); ok {
			c.currentLine = optDot.Token.Line
			err := c.Compile(optDot.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpDup)
			jumpNullPos := c.emit(code.OpJumpNull, 9999)

			key := &object.String{Value: optDot.Field.Value}
			c.emit(code.OpConstant, c.addConstant(key))
			c.emit(code.OpIndex)

			for _, a := range node.Arguments {
				err := c.Compile(a)
				if err != nil {
					return err
				}
			}
			c.emit(code.OpCall, len(node.Arguments))

			afterCallPos := len(c.currentInstructions())
			c.changeOperand(jumpNullPos, afterCallPos)
			return nil
		}

		if optIdx, ok := node.Function.(*ast.OptionalIndexExpression); ok {
			c.currentLine = optIdx.Token.Line
			err := c.Compile(optIdx.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpDup)
			jumpNullPos := c.emit(code.OpJumpNull, 9999)

			err = c.Compile(optIdx.Index)
			if err != nil {
				return err
			}
			c.emit(code.OpIndex)

			for _, a := range node.Arguments {
				err := c.Compile(a)
				if err != nil {
					return err
				}
			}
			c.emit(code.OpCall, len(node.Arguments))

			afterCallPos := len(c.currentInstructions())
			c.changeOperand(jumpNullPos, afterCallPos)
			return nil
		}

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

	return c.compileErr
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
	case FunctionScope:
		c.emit(code.OpCurrentClosure)
	}
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	if c.compileErr == nil {
		if def, err := code.Lookup(byte(op)); err == nil {
			for i, o := range operands {
				if i < len(def.OperandWidths) {
					w := def.OperandWidths[i]
					if w == 1 && (o < 0 || o > 255) {
						c.compileErr = fmt.Errorf("operand %d for opcode %s overflows 8-bit limit", o, def.Name)
					} else if w == 2 && (o < 0 || o > 65535) {
						c.compileErr = fmt.Errorf("operand %d for opcode %s overflows 16-bit limit", o, def.Name)
					}
				}
			}
		}
	}
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

func NewOptimizingCompiler() *Compiler {
	c := New()
	c.Optimize = true
	return c
}

func (c *Compiler) Bytecode() *Bytecode {
	instructions := c.currentInstructions()
	if c.Optimize {
		instructions = OptimizeBytecode(instructions)
	}
	return &Bytecode{
		Instructions: instructions,
		Constants:    c.constants,
		SourceMap:    c.scopes[c.scopeIndex].sourceLines,
	}
}
