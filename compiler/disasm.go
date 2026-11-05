package compiler

import (
	"bramandscript/code"
	"bramandscript/object"
	"bytes"
	"fmt"
	"strings"
)

// FormatBytecode renders a complete disassembled representation of Bytecode,
// including instructions, constant values, and nested functions.
func FormatBytecode(bc *Bytecode) string {
	var out bytes.Buffer
	out.WriteString("\n✦ BramandScript Bytecode Disassembly\n")
	out.WriteString("────────────────────────────────────────────────────────────────────────────\n")
	out.WriteString(FormatInstructions(bc.Instructions, bc.Constants, ""))
	out.WriteString("────────────────────────────────────────────────────────────────────────────\n")
	out.WriteString("✦ Constant Pool:\n")
	if len(bc.Constants) == 0 {
		out.WriteString("  (empty)\n")
	} else {
		for i, c := range bc.Constants {
			out.WriteString(fmt.Sprintf("  [%03d] %-18s %s\n", i, c.Type(), c.Inspect()))
			if fn, ok := c.(*object.CompiledFunction); ok {
				out.WriteString(fmt.Sprintf("        Parameters: %d, Locals: %d\n", fn.NumParameters, fn.NumLocals))
				out.WriteString("        Instructions:\n")
				out.WriteString(FormatInstructions(fn.Instructions, bc.Constants, "        "))
			}
		}
	}
	out.WriteString("────────────────────────────────────────────────────────────────────────────\n\n")
	return out.String()
}

// FormatInstructions disassembles bytecode instructions line-by-line with annotations.
func FormatInstructions(ins code.Instructions, constants []object.Object, indent string) string {
	var out bytes.Buffer
	i := 0
	for i < len(ins) {
		def, err := code.Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "%s%04d ERROR: %s\n", indent, i, err)
			i++
			continue
		}

		operands, read := code.ReadOperands(def, ins[i+1:])
		extra := ""
		if def.Name == "OpConstant" && len(operands) > 0 {
			cIdx := operands[0]
			if cIdx < len(constants) {
				extra = fmt.Sprintf(" ; constant: %s", constants[cIdx].Inspect())
			}
		}

		var operandStrs []string
		for _, op := range operands {
			operandStrs = append(operandStrs, fmt.Sprintf("%d", op))
		}
		opStr := strings.Join(operandStrs, " ")

		fmt.Fprintf(&out, "%s%04d %-22s %-12s%s\n", indent, i, def.Name, opStr, extra)
		i += 1 + read
	}
	return out.String()
}
