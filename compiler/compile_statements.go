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
	"bramandscript/stdlib"
	"bramandscript/token"
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
		if node.Pattern != nil {
			return true, c.compileDestructuring(node.Pattern, node.Value, false, node.Token)
		}
		if existing, ok := c.symbolTable.store[node.Name.Value]; ok && existing.IsConst && (existing.File == c.currentFile) {
			return true, c.makeError(node.Token.Line, node.Token.Col, "cannot re-declare constant '%s'", node.Name.Value)
		}
		if fnLit, ok := node.Value.(*ast.FunctionLiteral); ok && fnLit.Name == "" {
			fnLit.Name = node.Name.Value
		}
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
		if node.Pattern != nil {
			return true, c.compileDestructuring(node.Pattern, node.Value, true, node.Token)
		}
		if existing, ok := c.symbolTable.store[node.Name.Value]; ok && (existing.File == c.currentFile) {
			return true, c.makeError(node.Token.Line, node.Token.Col, "cannot re-declare constant '%s'", node.Name.Value)
		}
		if fnLit, ok := node.Value.(*ast.FunctionLiteral); ok && fnLit.Name == "" {
			fnLit.Name = node.Name.Value
		}
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
			return true, c.makeError(node.Token.Line, node.Token.Col, "cannot assign to undefined variable '%s'", node.Name.Value)
		}
		if symbol.IsConst {
			return true, c.makeError(node.Token.Line, node.Token.Col, "cannot reassign to constant variable '%s'", node.Name.Value)
		}

		if node.Operator == "=" {
			err := c.Compile(node.Value)
			if err != nil {
				return true, err
			}
		} else if node.Operator == "??=" {
			c.loadSymbol(symbol)
			c.emit(code.OpDup)
			jumpNotNullPos := c.emit(code.OpJumpNotNull, 9999)
			c.emit(code.OpPop)

			err := c.Compile(node.Value)
			if err != nil {
				return true, err
			}
			if symbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, symbol.Index)
			} else if symbol.Scope == LocalScope {
				c.emit(code.OpSetLocal, symbol.Index)
			} else if symbol.Scope == FreeScope {
				c.emit(code.OpSetFree, symbol.Index)
			}
			jumpEndPos := c.emit(code.OpJump, 9999)

			afterNotNullPos := len(c.currentInstructions())
			c.changeOperand(jumpNotNullPos, afterNotNullPos)
			c.emit(code.OpPop)

			afterAllPos := len(c.currentInstructions())
			c.changeOperand(jumpEndPos, afterAllPos)
			return true, nil
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
		if node.Operator == "=" || node.Operator == "" {
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
		} else if node.Operator == "??=" {
			// Compound nullish assignment on index/dot: target[index] ??= value
			c.iterCount++
			targetTemp := fmt.Sprintf("__cassign_t_%d", c.iterCount)
			idxTemp := fmt.Sprintf("__cassign_i_%d", c.iterCount)

			// 1. Evaluate target once and save in temp
			err := c.Compile(node.Left)
			if err != nil {
				return true, err
			}
			tSym := c.symbolTable.Define(targetTemp, false)
			if tSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, tSym.Index)
			} else {
				c.emit(code.OpSetLocal, tSym.Index)
			}

			// 2. Evaluate index once and save in temp
			err = c.Compile(node.Index)
			if err != nil {
				return true, err
			}
			iSym := c.symbolTable.Define(idxTemp, false)
			if iSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, iSym.Index)
			} else {
				c.emit(code.OpSetLocal, iSym.Index)
			}

			// 3. Load target[index]
			c.loadSymbol(tSym)
			c.loadSymbol(iSym)
			c.emit(code.OpIndex)
			c.emit(code.OpDup)
			jumpNotNullPos := c.emit(code.OpJumpNotNull, 9999)
			c.emit(code.OpPop)

			// 4. If null: evaluate value and set target[index] = value
			c.loadSymbol(tSym)
			c.loadSymbol(iSym)
			err = c.Compile(node.Value)
			if err != nil {
				return true, err
			}
			c.emit(code.OpSetIndex)
			jumpEndPos := c.emit(code.OpJump, 9999)

			// 5. If not null: skip evaluation and assignment
			afterNotNullPos := len(c.currentInstructions())
			c.changeOperand(jumpNotNullPos, afterNotNullPos)
			c.emit(code.OpPop)

			afterAllPos := len(c.currentInstructions())
			c.changeOperand(jumpEndPos, afterAllPos)
		} else {
			// Compound assignment on index/dot: target[index] op= value
			c.iterCount++
			targetTemp := fmt.Sprintf("__cassign_t_%d", c.iterCount)
			idxTemp := fmt.Sprintf("__cassign_i_%d", c.iterCount)

			// 1. Evaluate target once and save in temp
			err := c.Compile(node.Left)
			if err != nil {
				return true, err
			}
			tSym := c.symbolTable.Define(targetTemp, false)
			if tSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, tSym.Index)
			} else {
				c.emit(code.OpSetLocal, tSym.Index)
			}

			// 2. Evaluate index once and save in temp
			err = c.Compile(node.Index)
			if err != nil {
				return true, err
			}
			iSym := c.symbolTable.Define(idxTemp, false)
			if iSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, iSym.Index)
			} else {
				c.emit(code.OpSetLocal, iSym.Index)
			}

			// 3. For OpSetIndex, stack requires: left, index, new_value
			c.loadSymbol(tSym)
			c.loadSymbol(iSym)

			// 4. Compute new_value = target[index] <op> value
			c.loadSymbol(tSym)
			c.loadSymbol(iSym)
			c.emit(code.OpIndex)

			err = c.Compile(node.Value)
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
				return true, fmt.Errorf("unsupported compound operator %s for index assignment", node.Operator)
			}

			// 5. OpSetIndex stores new_value into left[index]
			c.emit(code.OpSetIndex)
		}

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

		// 4. Loop start condition: __iter_len(iter) > idx
		loopStart := len(c.currentInstructions())
		c.loops = append(c.loops, LoopContext{continuePos: -1, breakJumps: []int{}, continueJumps: []int{}})

		lenSym, ok := c.symbolTable.Resolve("__iter_len")
		if !ok {
			return true, c.makeError(node.Token.Line, node.Token.Col, "builtin '__iter_len' not found for loop iteration")
		}
		c.loadSymbol(lenSym)
		c.loadSymbol(iterSym)
		c.emit(code.OpCall, 1)

		c.loadSymbol(idxSym)
		c.emit(code.OpGreaterThan)

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		// 5. Fetch loop variables
		if node.Index != nil {
			// 2 variables: for k, v in iter
			keySym, _ := c.symbolTable.Resolve("__iter_key")
			c.loadSymbol(keySym)
			c.loadSymbol(iterSym)
			c.loadSymbol(idxSym)
			c.emit(code.OpCall, 2)
			if userIdxSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, userIdxSym.Index)
			} else {
				c.emit(code.OpSetLocal, userIdxSym.Index)
			}

			valSym, _ := c.symbolTable.Resolve("__iter_val")
			c.loadSymbol(valSym)
			c.loadSymbol(iterSym)
			c.loadSymbol(idxSym)
			c.emit(code.OpCall, 2)
			if itemSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, itemSym.Index)
			} else {
				c.emit(code.OpSetLocal, itemSym.Index)
			}
		} else {
			// 1 variable: for x in iter
			itemFnSym, _ := c.symbolTable.Resolve("__iter_item")
			c.loadSymbol(itemFnSym)
			c.loadSymbol(iterSym)
			c.loadSymbol(idxSym)
			c.emit(code.OpCall, 2)
			if itemSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, itemSym.Index)
			} else {
				c.emit(code.OpSetLocal, itemSym.Index)
			}
		}

		// 6. Compile body
		err = c.Compile(node.Body)
		if err != nil {
			c.leaveBlockScope()
			c.loops = c.loops[:len(c.loops)-1]
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
		structSym := c.symbolTable.Define(node.Name.Value, false)
		c.enterScope()

		for _, e := range node.Embeds {
			c.symbolTable.Define(e.Value, false)
		}
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

		numPairs := 1 + len(node.Fields) + len(node.Methods) + len(node.Embeds)
		if len(node.Embeds) > 0 {
			numPairs++ // for __embeds__ array
		}

		c.emit(code.OpConstant, c.addConstant(&object.String{Value: "__struct__"}))
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Name.Value}))

		for _, e := range node.Embeds {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: e.Value}))
			sym, _ := c.symbolTable.Resolve(e.Value)
			c.loadSymbol(sym)
		}

		if len(node.Embeds) > 0 {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: "__embeds__"}))
			for _, e := range node.Embeds {
				sym, _ := c.symbolTable.Resolve(e.Value)
				c.loadSymbol(sym)
			}
			c.emit(code.OpArray, len(node.Embeds))
		}

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

		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions, sourceLines := c.leaveScope()

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

		compiledFn := &object.CompiledFunction{
			Name:          node.Name.Value,
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Embeds) + len(node.Fields),
			SourceMap:     sourceLines,
			FreeNames:     freeNames,
		}

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

		if structSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, structSym.Index)
		} else {
			c.emit(code.OpSetLocal, structSym.Index)
		}

	case *ast.EnumStatement:
		c.currentLine = node.Token.Line
		enumSym := c.symbolTable.Define(node.Name.Value, false)

		currentVal := int64(0)
		numPairs := 0

		for _, entry := range node.Entries {
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: entry.Name.Value}))
			if entry.Value != nil {
				if intLit, ok := entry.Value.(*ast.IntegerLiteral); ok {
					currentVal = intLit.Value
					c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: currentVal}))
					currentVal++
				} else {
					err := c.Compile(entry.Value)
					if err != nil {
						return true, err
					}
				}
			} else {
				c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: currentVal}))
				currentVal++
			}
			numPairs++
		}

		c.emit(code.OpConstant, c.addConstant(&object.String{Value: "__enum__"}))
		c.emit(code.OpTrue)
		numPairs++

		c.emit(code.OpConstant, c.addConstant(&object.String{Value: "__name__"}))
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Name.Value}))
		numPairs++

		c.emit(code.OpHash, numPairs*2)

		if enumSym.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, enumSym.Index)
		} else {
			c.emit(code.OpSetLocal, enumSym.Index)
		}

	case *ast.ImportStatement:
		c.currentLine = node.Token.Line
		filePath, err := c.resolveModulePath(node.Path)
		if err != nil {
			return true, c.makeError(node.Token.Line, node.Token.Col, "cannot import module '%s': %s", node.Path, err)
		}

		if c.importingModules[filePath] {
			return true, c.makeError(node.Token.Line, node.Token.Col, "circular import detected: module '%s' is already being imported", node.Path)
		}

		if !c.importedModules[filePath] {
			c.importingModules[filePath] = true
			defer func() {
				delete(c.importingModules, filePath)
			}()
			c.importedModules[filePath] = true

			var data []byte
			var err error
			if strings.HasPrefix(filePath, "embed:stdlib/") {
				data, err = stdlib.ReadFile(filePath)
			} else {
				data, err = os.ReadFile(filePath)
			}
			if err != nil {
				return true, c.makeError(node.Token.Line, node.Token.Col, "cannot import module '%s': %s", node.Path, err)
			}

			l := lexer.New(string(data))
			p := parser.New(l)
			moduleProgram := p.ParseProgram()
			if len(p.Errors()) > 0 {
				return true, c.makeError(node.Token.Line, node.Token.Col, "syntax error in module '%s': %s", node.Path, p.Errors()[0])
			}

			prevBaseDir := c.BaseDir
			if strings.HasPrefix(filePath, "embed:stdlib/") {
				c.BaseDir = filepath.ToSlash(filepath.Dir(filePath))
			} else {
				c.BaseDir = filepath.Dir(filePath)
			}

			prevFile := c.currentFile
			prevSource := c.currentSource
			c.SetCurrentFile(filePath, string(data))

			restoreFileCtx := func() {
				c.BaseDir = prevBaseDir
				c.SetCurrentFile(prevFile, prevSource)
			}

			exportedNames := []string{}
			for _, stmt := range moduleProgram.Statements {
				switch s := stmt.(type) {
				case *ast.LetStatement:
					if s.Name != nil {
						exportedNames = append(exportedNames, s.Name.Value)
					} else if s.Pattern != nil {
						switch pat := s.Pattern.(type) {
						case *ast.ArrayPattern:
							for _, el := range pat.Elements {
								if rp, ok := el.(*ast.RestPattern); ok && rp.Target != nil {
									exportedNames = append(exportedNames, rp.Target.Value)
								} else if ident, ok := el.(*ast.Identifier); ok {
									exportedNames = append(exportedNames, ident.Value)
								}
							}
						case *ast.HashPattern:
							for _, key := range pat.Keys {
								exportedNames = append(exportedNames, key.Value)
							}
						}
					}
				case *ast.ConstStatement:
					if s.Name != nil {
						exportedNames = append(exportedNames, s.Name.Value)
					} else if s.Pattern != nil {
						switch pat := s.Pattern.(type) {
						case *ast.ArrayPattern:
							for _, el := range pat.Elements {
								if rp, ok := el.(*ast.RestPattern); ok && rp.Target != nil {
									exportedNames = append(exportedNames, rp.Target.Value)
								} else if ident, ok := el.(*ast.Identifier); ok {
									exportedNames = append(exportedNames, ident.Value)
								}
							}
						case *ast.HashPattern:
							for _, key := range pat.Keys {
								exportedNames = append(exportedNames, key.Value)
							}
						}
					}
				case *ast.ExpressionStatement:
					if fnLit, ok := s.Expression.(*ast.FunctionLiteral); ok && fnLit.Name != "" {
						exportedNames = append(exportedNames, fnLit.Name)
					}
				case *ast.StructStatement:
					exportedNames = append(exportedNames, s.Name.Value)
				case *ast.EnumStatement:
					exportedNames = append(exportedNames, s.Name.Value)
				case *ast.ImportStatement:
					if s.Alias != "" {
						exportedNames = append(exportedNames, s.Alias)
					}
				}
				err := c.Compile(stmt)
				if err != nil {
					restoreFileCtx()
					return true, err
				}
			}
			restoreFileCtx()
			c.moduleExports[filePath] = exportedNames
		}

		if node.Alias != "" {
			exportedNames := c.moduleExports[filePath]
			for _, name := range exportedNames {
				c.emit(code.OpConstant, c.addConstant(&object.String{Value: name}))
				sym, ok := c.symbolTable.Resolve(name)
				if !ok {
					c.emit(code.OpNull)
				} else {
					c.loadSymbol(sym)
				}
			}
			c.emit(code.OpHash, len(exportedNames)*2)

			aliasSym := c.symbolTable.Define(node.Alias, false)
			if aliasSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, aliasSym.Index)
			} else {
				c.emit(code.OpSetLocal, aliasSym.Index)
			}
		}

	case *ast.WhileStatement:
		loopStart := len(c.currentInstructions())
		c.loops = append(c.loops, LoopContext{continuePos: loopStart, breakJumps: []int{}, continueJumps: []int{}})

		err := c.Compile(node.Condition)
		if err != nil {
			c.loops = c.loops[:len(c.loops)-1]
			return true, err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Body)
		if err != nil {
			c.loops = c.loops[:len(c.loops)-1]
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
			return true, c.makeError(node.Token.Line, node.Token.Col, "'break' statement outside of loop")
		}
		breakPos := c.emit(code.OpJump, 9999)
		c.loops[len(c.loops)-1].breakJumps = append(c.loops[len(c.loops)-1].breakJumps, breakPos)

	case *ast.ContinueStatement:
		if len(c.loops) == 0 {
			return true, c.makeError(node.Token.Line, node.Token.Col, "'continue' statement outside of loop")
		}
		curr := &c.loops[len(c.loops)-1]
		if curr.continuePos != -1 {
			c.emit(code.OpJump, curr.continuePos)
		} else {
			pos := c.emit(code.OpJump, 9999)
			curr.continueJumps = append(curr.continueJumps, pos)
		}

	case *ast.ReturnStatement:
		if node.ReturnValue == nil {
			c.emit(code.OpReturn)
		} else {
			err := c.Compile(node.ReturnValue)
			if err != nil {
				return true, err
			}
			c.emit(code.OpReturnValue)
		}

	case *ast.TryCatchStatement:
		c.currentLine = node.Token.Line
		pushHandlerPos := c.emit(code.OpPushHandler, 9999)

		err := c.Compile(node.Block)
		if err != nil {
			return true, err
		}

		c.emit(code.OpPopHandler)
		jumpPastCatchPos := c.emit(code.OpJump, 9999)

		catchTarget := len(c.currentInstructions())
		c.changeOperand(pushHandlerPos, catchTarget)

		if node.CatchBlock != nil {
			if node.CatchVar != nil {
				c.enterBlockScope()
				symbol := c.symbolTable.Define(node.CatchVar.Value, false)
				if symbol.Scope == GlobalScope {
					c.emit(code.OpSetGlobal, symbol.Index)
				} else {
					c.emit(code.OpSetLocal, symbol.Index)
				}
			} else {
				c.emit(code.OpPop)
			}

			for _, s := range node.CatchBlock.Statements {
				err := c.Compile(s)
				if err != nil {
					if node.CatchVar != nil {
						c.leaveBlockScope()
					}
					return true, err
				}
			}
			if node.CatchVar != nil {
				c.leaveBlockScope()
			}
		} else {
			// Try-finally without catch: an exception occurred in try block.
			// Save error, run finally block, and re-throw the error!
			c.enterBlockScope()
			tmpErrSym := c.symbolTable.Define("__finally_err", false)
			if tmpErrSym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, tmpErrSym.Index)
			} else {
				c.emit(code.OpSetLocal, tmpErrSym.Index)
			}

			if node.FinallyBlock != nil {
				err := c.Compile(node.FinallyBlock)
				if err != nil {
					c.leaveBlockScope()
					return true, err
				}
			}

			if tmpErrSym.Scope == GlobalScope {
				c.emit(code.OpGetGlobal, tmpErrSym.Index)
			} else {
				c.emit(code.OpGetLocal, tmpErrSym.Index)
			}
			c.emit(code.OpThrow)
			c.leaveBlockScope()
		}

		afterCatchPos := len(c.currentInstructions())
		c.changeOperand(jumpPastCatchPos, afterCatchPos)

		if node.FinallyBlock != nil {
			err := c.Compile(node.FinallyBlock)
			if err != nil {
				return true, err
			}
		}

	case *ast.ThrowStatement:
		c.currentLine = node.Token.Line
		if node.Value != nil {
			err := c.Compile(node.Value)
			if err != nil {
				return true, err
			}
		} else {
			c.emit(code.OpNull)
		}
		c.emit(code.OpThrow)

	case *ast.DeferStatement:
		c.currentLine = node.Token.Line
		if node.Call == nil {
			return true, c.makeError(node.Token.Line, node.Token.Col, "defer statement requires an expression or function call")
		}
		if call, ok := node.Call.(*ast.CallExpression); ok {
			err := c.Compile(call.Function)
			if err != nil {
				return true, err
			}
			for _, arg := range call.Arguments {
				err := c.Compile(arg)
				if err != nil {
					return true, err
				}
			}
			c.emit(code.OpDefer, len(call.Arguments))
		} else {
			err := c.Compile(node.Call)
			if err != nil {
				return true, err
			}
			c.emit(code.OpDefer, 0)
		}

	default:
		return false, nil
	}

	return true, nil
}

func (c *Compiler) resolveModulePath(importPath string) (string, error) {
	// 0. Standard library resolution ("std/...", "stdlib/...")
	var stdSubpath string
	if strings.HasPrefix(importPath, "std/") {
		stdSubpath = strings.TrimPrefix(importPath, "std/")
	} else if strings.HasPrefix(importPath, "stdlib/") {
		stdSubpath = strings.TrimPrefix(importPath, "stdlib/")
	}

	if stdSubpath != "" {
		stdlibRoots := []string{}
		if env := os.Getenv("BRAMAND_STDLIB"); env != "" {
			stdlibRoots = append(stdlibRoots, env)
		}
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			stdlibRoots = append(stdlibRoots,
				filepath.Join(execDir, "stdlib"),
				filepath.Join(execDir, "..", "stdlib"),
			)
		}
		stdlibRoots = append(stdlibRoots, "stdlib", filepath.Join(".", "stdlib"))
		if c.BaseDir != "" {
			stdlibRoots = append(stdlibRoots,
				filepath.Join(c.BaseDir, "stdlib"),
				filepath.Join(c.BaseDir, "..", "stdlib"),
				filepath.Join(c.BaseDir, "..", "..", "stdlib"),
			)
		}
		if homeDir, err := os.UserHomeDir(); err == nil {
			stdlibRoots = append(stdlibRoots,
				filepath.Join(homeDir, ".bramand", "stdlib"),
				filepath.Join(homeDir, ".bramand_libs", "stdlib"),
			)
		}

		for _, root := range stdlibRoots {
			candidates := []string{
				filepath.Join(root, stdSubpath, "index.bmd"),
				filepath.Join(root, stdSubpath+".bmd"),
				filepath.Join(root, stdSubpath),
			}
			for _, cand := range candidates {
				if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
					if abs, err := filepath.Abs(cand); err == nil {
						return abs, nil
					}
					return cand, nil
				}
			}
		}

		// Fallback to embedded standard library bundled in the binary
		if embeddedPath, ok := stdlib.ResolveEmbeddedPath(stdSubpath); ok {
			return "embed:stdlib/" + embeddedPath, nil
		}
	}

	// If current module being compiled is an embedded stdlib module, resolve relative imports
	normBaseDir := filepath.ToSlash(c.BaseDir)
	if strings.HasPrefix(normBaseDir, "embed:stdlib/") {
		base := strings.TrimPrefix(normBaseDir, "embed:stdlib/")
		rel := filepath.ToSlash(filepath.Clean(filepath.Join(base, importPath)))
		if embeddedPath, ok := stdlib.ResolveEmbeddedPath(rel); ok {
			return "embed:stdlib/" + embeddedPath, nil
		}
	}

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

func (c *Compiler) compileDestructuring(pattern ast.Node, value ast.Expression, isConst bool, tok token.Token) error {
	c.currentLine = tok.Line
	err := c.Compile(value)
	if err != nil {
		return err
	}

	c.destructCount++
	tmpName := fmt.Sprintf("__destruct_tmp_%d", c.destructCount)
	tmpSym := c.symbolTable.Define(tmpName, false)
	if tmpSym.Scope == GlobalScope {
		c.emit(code.OpSetGlobal, tmpSym.Index)
	} else {
		c.emit(code.OpSetLocal, tmpSym.Index)
	}

	switch pat := pattern.(type) {
	case *ast.ArrayPattern:
		for i, el := range pat.Elements {
			if rp, ok := el.(*ast.RestPattern); ok {
				sliceBuiltin, ok := c.symbolTable.Resolve("slice")
				if !ok {
					return fmt.Errorf("undefined builtin 'slice'")
				}
				c.loadSymbol(sliceBuiltin)
				c.loadSymbol(tmpSym)
				c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: int64(i)}))
				c.emit(code.OpNull)
				c.emit(code.OpNull)
				c.emit(code.OpCall, 4)

				targetSym := c.symbolTable.Define(rp.Target.Value, isConst)
				if targetSym.Scope == GlobalScope {
					c.emit(code.OpSetGlobal, targetSym.Index)
				} else {
					c.emit(code.OpSetLocal, targetSym.Index)
				}
			} else if ident, ok := el.(*ast.Identifier); ok {
				c.loadSymbol(tmpSym)
				c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: int64(i)}))
				c.emit(code.OpIndex)

				sym := c.symbolTable.Define(ident.Value, isConst)
				if sym.Scope == GlobalScope {
					c.emit(code.OpSetGlobal, sym.Index)
				} else {
					c.emit(code.OpSetLocal, sym.Index)
				}
			}
		}

	case *ast.HashPattern:
		for _, key := range pat.Keys {
			c.loadSymbol(tmpSym)
			c.emit(code.OpConstant, c.addConstant(&object.String{Value: key.Value}))
			c.emit(code.OpIndex)

			sym := c.symbolTable.Define(key.Value, isConst)
			if sym.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, sym.Index)
			} else {
				c.emit(code.OpSetLocal, sym.Index)
			}
		}
	}

	return nil
}
