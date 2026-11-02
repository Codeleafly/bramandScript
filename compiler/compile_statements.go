package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bramandscript/ast"
	"bramandscript/code"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
)

func (c *Compiler) compileStatement(node ast.Node) (bool, error) {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return true, err
			}
		}

	case *ast.ExpressionStatement:
		if fnLit, ok := node.Expression.(*ast.FunctionLiteral); ok && fnLit.Name != "" {
			symbol := c.symbolTable.Define(fnLit.Name, false)
			err := c.Compile(node.Expression)
			if err != nil {
				return true, err
			}
			if symbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, symbol.Index)
			} else {
				c.emit(code.OpSetLocal, symbol.Index)
			}
			c.emit(code.OpNull)
			return true, nil
		}
		err := c.Compile(node.Expression)
		if err != nil {
			return true, err
		}
		c.emit(code.OpPop)

	case *ast.BlockStatement:
		c.enterBlockScope()
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				c.leaveBlockScope()
				return true, err
			}
		}
		c.leaveBlockScope()

	case *ast.LetStatement:
		symbol := c.symbolTable.Define(node.Name.Value, false)
		err := c.Compile(node.Value)
		if err != nil {
			return true, err
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
			return true, err
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.ReassignStatement:
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot assign to undefined variable '%s'", node.Name.Value)}
		}
		if symbol.IsConst {
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot reassign to constant variable '%s'", node.Name.Value)}
		}

		if node.Operator == "=" {
			err := c.Compile(node.Value)
			if err != nil {
				return true, err
			}
		} else {
			c.loadSymbol(symbol)
			err := c.Compile(node.Value)
			if err != nil {
				return true, err
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
				return true, fmt.Errorf("unsupported compound operator %s", node.Operator)
			}
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else if symbol.Scope == LocalScope {
			c.emit(code.OpSetLocal, symbol.Index)
		} else if symbol.Scope == FreeScope {
			c.emit(code.OpSetFree, symbol.Index)
		} else {
			return true, fmt.Errorf("cannot assign to %s variable %s", symbol.Scope, node.Name.Value)
		}

	case *ast.IndexAssignStatement:
		c.currentLine = node.Token.Line
		err := c.Compile(node.Left)
		if err != nil {
			return true, err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return true, err
		}
		err = c.Compile(node.Value)
		if err != nil {
			return true, err
		}
		c.emit(code.OpSetIndex)

	case *ast.ForInStatement:
		c.enterBlockScope()
		c.currentLine = node.Token.Line
		c.iterCount++
		iterName := fmt.Sprintf("__iter_%d", c.iterCount)
		idxName := fmt.Sprintf("__idx_%d", c.iterCount)

		// 1. Compile iterable & store in iterName
		err := c.Compile(node.Iterable)
		if err != nil {
			c.leaveBlockScope()
			return true, err
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
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "builtin 'len' not found for loop iteration"}
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
			c.leaveBlockScope()
			return true, err
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
		c.leaveBlockScope()

	case *ast.StructStatement:
		c.currentLine = node.Token.Line
		c.enterScope()

		for _, f := range node.Fields {
			c.symbolTable.Define(f.Value, false)
		}

		for _, m := range node.Methods {
			err := c.Compile(m)
			if err != nil {
				return true, err
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
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot import module '%s': %s", node.Path, err)}
		}

		if !c.importedModules[filePath] {
			c.importedModules[filePath] = true

			data, err := os.ReadFile(filePath)
			if err != nil {
				return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("cannot import module '%s': %s", node.Path, err)}
			}

			l := lexer.New(string(data))
			p := parser.New(l)
			moduleProgram := p.ParseProgram()
			if len(p.Errors()) > 0 {
				return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: fmt.Sprintf("syntax error in module '%s': %s", node.Path, p.Errors()[0])}
			}

			prevBaseDir := c.BaseDir
			c.BaseDir = filepath.Dir(filePath)

			if node.Alias == "" {
				for _, stmt := range moduleProgram.Statements {
					err := c.Compile(stmt)
					if err != nil {
						c.BaseDir = prevBaseDir
						return true, err
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
						return true, err
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

	case *ast.WhileStatement:
		loopStart := len(c.currentInstructions())
		c.loops = append(c.loops, LoopContext{continuePos: loopStart, breakJumps: []int{}, continueJumps: []int{}})

		err := c.Compile(node.Condition)
		if err != nil {
			return true, err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Body)
		if err != nil {
			return true, err
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
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "'break' statement outside of loop"}
		}
		breakPos := c.emit(code.OpJump, 9999)
		c.loops[len(c.loops)-1].breakJumps = append(c.loops[len(c.loops)-1].breakJumps, breakPos)

	case *ast.ContinueStatement:
		if len(c.loops) == 0 {
			return true, &CompilerError{Line: node.Token.Line, Col: node.Token.Col, Message: "'continue' statement outside of loop"}
		}
		curr := &c.loops[len(c.loops)-1]
		if curr.continuePos != -1 {
			c.emit(code.OpJump, curr.continuePos)
		} else {
			pos := c.emit(code.OpJump, 9999)
			curr.continueJumps = append(curr.continueJumps, pos)
		}

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return true, err
		}
		c.emit(code.OpReturnValue)

	default:
		return false, nil
	}

	return true, nil
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
