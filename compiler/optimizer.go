package compiler

import (
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
		left := FoldConstants(n.Left).(ast.Expression)
		right := FoldConstants(n.Right).(ast.Expression)
		n.Left = left
		n.Right = right

		// Integer op Integer
		intLeft, okLeft := left.(*ast.IntegerLiteral)
		intRight, okRight := right.(*ast.IntegerLiteral)
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
				if v2 != 0 {
					return &ast.IntegerLiteral{Token: n.Token, Value: v1 / v2}
				}
			case "%":
				if v2 != 0 {
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
		floatLeft, okFLeft := left.(*ast.FloatLiteral)
		floatRight, okFRight := right.(*ast.FloatLiteral)
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
		strLeft, okSLeft := left.(*ast.StringLiteral)
		strRight, okSRight := right.(*ast.StringLiteral)
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
		boolLeft, okBLeft := left.(*ast.BooleanLiteral)
		boolRight, okBRight := right.(*ast.BooleanLiteral)
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
		right := FoldConstants(n.Right).(ast.Expression)
		n.Right = right

		switch n.Operator {
		case "!":
			if b, ok := right.(*ast.BooleanLiteral); ok {
				return &ast.BooleanLiteral{Token: n.Token, Value: !b.Value}
			}
		case "-":
			if i, ok := right.(*ast.IntegerLiteral); ok {
				return &ast.IntegerLiteral{Token: n.Token, Value: -i.Value}
			}
			if f, ok := right.(*ast.FloatLiteral); ok {
				return &ast.FloatLiteral{Token: n.Token, Value: -f.Value}
			}
		case "~":
			if i, ok := right.(*ast.IntegerLiteral); ok {
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
	i := 0
	for i < len(ins) {
		op := code.Opcode(ins[i])
		def, err := code.Lookup(ins[i])
		if err != nil {
			out = append(out, ins[i])
			i++
			continue
		}
		insLen := 1
		for _, w := range def.OperandWidths {
			insLen += w
		}

		// Peephole: OpDup followed immediately by OpPop -> cancel both out
		if op == code.OpDup && i+insLen < len(ins) && code.Opcode(ins[i+insLen]) == code.OpPop {
			i += insLen + 1
			continue
		}

		out = append(out, ins[i:i+insLen]...)
		i += insLen
	}
	return out
}
