package compiler

import (
	"encoding/binary"
	"math"

	"bramandscript/ast"
	"bramandscript/code"
)

// FoldConstants performs compile-time evaluation of constant expressions in the AST.
func FoldConstants(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.InfixExpression:
		if n.Left == nil || n.Right == nil {
			return n
		}
		leftExpr, ok := FoldConstants(n.Left).(ast.Expression)
		if !ok {
			return n
		}
		rightExpr, ok := FoldConstants(n.Right).(ast.Expression)
		if !ok {
			return n
		}
		n.Left = leftExpr
		n.Right = rightExpr

		// Integer op Integer
		intLeft, okLeft := leftExpr.(*ast.IntegerLiteral)
		intRight, okRight := rightExpr.(*ast.IntegerLiteral)
		if okLeft && okRight {
			v1 := intLeft.Value
			v2 := intRight.Value
			switch n.Operator {
			case "+":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 + v2}
			case "-":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 - v2}
			case "*":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 * v2}
			case "/":
				if v2 != 0 && !(v1 == math.MinInt64 && v2 == -1) {
					return &ast.IntegerLiteral{Token: n.Token, Value: v1 / v2}
				}
			case "%":
				if v2 != 0 && !(v1 == math.MinInt64 && v2 == -1) {
					return &ast.IntegerLiteral{Token: n.Token, Value: v1 % v2}
				}
			case "**":
				if v2 >= 0 && v2 <= 30 {
					res := int64(1)
					for i := int64(0); i < v2; i++ {
						res = res * v1
					}
					return &ast.IntegerLiteral{Token: n.Token, Value: res}
				}
			case "&":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 & v2}
			case "|":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 | v2}
			case "^":
				return &ast.IntegerLiteral{Token: n.Token, Value: v1 ^ v2}
			case "<<":
				if v2 >= 0 && v2 < 64 {
					return &ast.IntegerLiteral{Token: n.Token, Value: v1 << uint(v2)}
				}
			case ">>":
				if v2 >= 0 && v2 < 64 {
					return &ast.IntegerLiteral{Token: n.Token, Value: v1 >> uint(v2)}
				}
			case "==":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 == v2}
			case "!=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 != v2}
			case "<":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 < v2}
			case ">":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 > v2}
			case "<=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 <= v2}
			case ">=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 >= v2}
			}
		}

		// Float op Float
		floatLeft, okFLeft := leftExpr.(*ast.FloatLiteral)
		floatRight, okFRight := rightExpr.(*ast.FloatLiteral)
		if okFLeft && okFRight {
			v1 := floatLeft.Value
			v2 := floatRight.Value
			switch n.Operator {
			case "+":
				return &ast.FloatLiteral{Token: n.Token, Value: v1 + v2}
			case "-":
				return &ast.FloatLiteral{Token: n.Token, Value: v1 - v2}
			case "*":
				return &ast.FloatLiteral{Token: n.Token, Value: v1 * v2}
			case "/":
				if v2 != 0.0 {
					return &ast.FloatLiteral{Token: n.Token, Value: v1 / v2}
				}
			case "**":
				return &ast.FloatLiteral{Token: n.Token, Value: math.Pow(v1, v2)}
			case "==":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 == v2}
			case "!=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 != v2}
			case "<":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 < v2}
			case ">":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 > v2}
			case "<=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 <= v2}
			case ">=":
				return &ast.BooleanLiteral{Token: n.Token, Value: v1 >= v2}
			}
		}

		// String concatenation
		strLeft, okSLeft := leftExpr.(*ast.StringLiteral)
		strRight, okSRight := rightExpr.(*ast.StringLiteral)
		if okSLeft && okSRight {
			switch n.Operator {
			case "+":
				return &ast.StringLiteral{Token: n.Token, Value: strLeft.Value + strRight.Value}
			case "==":
				return &ast.BooleanLiteral{Token: n.Token, Value: strLeft.Value == strRight.Value}
			case "!=":
				return &ast.BooleanLiteral{Token: n.Token, Value: strLeft.Value != strRight.Value}
			}
		}

		// Boolean equality
		boolLeft, okBLeft := leftExpr.(*ast.BooleanLiteral)
		boolRight, okBRight := rightExpr.(*ast.BooleanLiteral)
		if okBLeft && okBRight {
			switch n.Operator {
			case "==":
				return &ast.BooleanLiteral{Token: n.Token, Value: boolLeft.Value == boolRight.Value}
			case "!=":
				return &ast.BooleanLiteral{Token: n.Token, Value: boolLeft.Value != boolRight.Value}
			}
		}

		return n

	case *ast.PrefixExpression:
		if n.Right == nil {
			return n
		}
		rightExpr, ok := FoldConstants(n.Right).(ast.Expression)
		if !ok {
			return n
		}
		n.Right = rightExpr

		switch n.Operator {
		case "!":
			if b, ok := rightExpr.(*ast.BooleanLiteral); ok {
				return &ast.BooleanLiteral{Token: n.Token, Value: !b.Value}
			}
		case "-":
			if i, ok := rightExpr.(*ast.IntegerLiteral); ok {
				return &ast.IntegerLiteral{Token: n.Token, Value: -i.Value}
			}
			if f, ok := rightExpr.(*ast.FloatLiteral); ok {
				return &ast.FloatLiteral{Token: n.Token, Value: -f.Value}
			}
		case "~":
			if i, ok := rightExpr.(*ast.IntegerLiteral); ok {
				return &ast.IntegerLiteral{Token: n.Token, Value: ^i.Value}
			}
		}
		return n

	default:
		return node
	}
}

// OptimizeBytecode performs a peephole optimization pass over bytecode instructions.
func OptimizeBytecode(ins code.Instructions) code.Instructions {
	if len(ins) < 2 {
		return ins
	}
	out := make(code.Instructions, 0, len(ins))
	offsetMap := make([]int, len(ins)+1)
	i := 0
	for i < len(ins) {
		op := code.Opcode(ins[i])
		def, err := code.Lookup(ins[i])
		if err != nil {
			offsetMap[i] = len(out)
			out = append(out, ins[i])
			i++
			continue
		}
		insLen := 1
		for _, w := range def.OperandWidths {
			insLen += w
		}
		if i+insLen > len(ins) {
			for k := i; k < len(ins); k++ {
				offsetMap[k] = len(out)
				out = append(out, ins[k])
			}
			break
		}

		// Peephole: OpDup followed immediately by OpPop -> cancel both out
		if op == code.OpDup && i+insLen < len(ins) && code.Opcode(ins[i+insLen]) == code.OpPop {
			for k := 0; k < insLen+1; k++ {
				offsetMap[i+k] = len(out)
			}
			i += insLen + 1
			continue
		}

		for k := 0; k < insLen; k++ {
			offsetMap[i+k] = len(out) + k
		}
		out = append(out, ins[i:i+insLen]...)
		i += insLen
	}
	offsetMap[len(ins)] = len(out)

	// Pass 2: Re-target all jumps according to offsetMap
	j := 0
	for j < len(out) {
		op := code.Opcode(out[j])
		def, err := code.Lookup(out[j])
		if err != nil {
			j++
			continue
		}
		insLen := 1
		for _, w := range def.OperandWidths {
			insLen += w
		}
		if op == code.OpJump || op == code.OpJumpNotTruthy || op == code.OpJumpTruthy || op == code.OpPushHandler {
			if j+9 <= len(out) {
				oldTarget := int(binary.BigEndian.Uint64(out[j+1:]))
				if oldTarget <= len(ins) {
					newTarget := offsetMap[oldTarget]
					binary.BigEndian.PutUint64(out[j+1:], uint64(newTarget))
				}
			}
		}
		j += insLen
	}
	return out
}
