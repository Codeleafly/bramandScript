package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte

type Opcode byte

const (
	OpConstant Opcode = iota
	OpPop

	// Arithmetic
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow

	// Booleans & Null
	OpTrue
	OpFalse
	OpNull

	// Comparisons
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpGreaterThanEqual

	// Logical / Unary
	OpMinus
	OpBang
	OpAnd
	OpOr

	// Jumps
	OpJumpNotTruthy
	OpJump

	// Variables
	OpSetGlobal
	OpGetGlobal
	OpSetLocal
	OpGetLocal

	// Data Structures
	OpArray
	OpHash
	OpIndex
	OpSetIndex

	// Functions & Builtins
	OpCall
	OpReturnValue
	OpReturn
	OpGetBuiltin
	OpClosure
	OpGetFree
	OpSetFree

	// Concurrency & Channels
	OpSpawn
	OpSend
	OpReceive

	// Bitwise Operators
	OpBitAnd
	OpBitOr
	OpBitXor
	OpShiftLeft
	OpShiftRight
	OpBitNot

	// Stack & Control Flow
	OpDup
	OpJumpTruthy
	OpCurrentClosure
	OpGetLocalCell
	OpGetFreeCell

	// Exception Handling
	OpPushHandler
	OpPopHandler
	OpThrow

	// Enterprise Null-Coalescing & Optional Jumps
	OpJumpNotNull
	OpJumpNull

	// Enterprise Defer & Spread Opcodes
	OpDefer
	OpSpread

	// Centralized Package & Module Engine Opcodes
	OpModule
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant:         {"OpConstant", []int{8}},
	OpPop:              {"OpPop", []int{}},
	OpAdd:              {"OpAdd", []int{}},
	OpSub:              {"OpSub", []int{}},
	OpMul:              {"OpMul", []int{}},
	OpDiv:              {"OpDiv", []int{}},
	OpMod:              {"OpMod", []int{}},
	OpPow:              {"OpPow", []int{}},
	OpTrue:             {"OpTrue", []int{}},
	OpFalse:            {"OpFalse", []int{}},
	OpNull:             {"OpNull", []int{}},
	OpEqual:            {"OpEqual", []int{}},
	OpNotEqual:         {"OpNotEqual", []int{}},
	OpGreaterThan:      {"OpGreaterThan", []int{}},
	OpGreaterThanEqual: {"OpGreaterThanEqual", []int{}},
	OpMinus:            {"OpMinus", []int{}},
	OpBang:             {"OpBang", []int{}},
	OpAnd:              {"OpAnd", []int{}},
	OpOr:               {"OpOr", []int{}},
	OpJumpNotTruthy:    {"OpJumpNotTruthy", []int{8}},
	OpJump:             {"OpJump", []int{8}},
	OpSetGlobal:        {"OpSetGlobal", []int{8}},
	OpGetGlobal:        {"OpGetGlobal", []int{8}},
	OpSetLocal:         {"OpSetLocal", []int{2}},
	OpGetLocal:         {"OpGetLocal", []int{2}},
	OpArray:            {"OpArray", []int{2}},
	OpHash:             {"OpHash", []int{2}},
	OpIndex:            {"OpIndex", []int{}},
	OpSetIndex:         {"OpSetIndex", []int{}},
	OpCall:             {"OpCall", []int{1}},
	OpReturnValue:      {"OpReturnValue", []int{}},
	OpReturn:           {"OpReturn", []int{}},
	OpGetBuiltin:       {"OpGetBuiltin", []int{1}},
	OpClosure:          {"OpClosure", []int{8, 1}},
	OpGetFree:          {"OpGetFree", []int{1}},
	OpSetFree:          {"OpSetFree", []int{1}},
	OpSpawn:            {"OpSpawn", []int{1}},
	OpSend:             {"OpSend", []int{}},
	OpReceive:          {"OpReceive", []int{}},
	OpBitAnd:           {"OpBitAnd", []int{}},
	OpBitOr:            {"OpBitOr", []int{}},
	OpBitXor:           {"OpBitXor", []int{}},
	OpShiftLeft:        {"OpShiftLeft", []int{}},
	OpShiftRight:       {"OpShiftRight", []int{}},
	OpBitNot:           {"OpBitNot", []int{}},
	OpDup:              {"OpDup", []int{}},
	OpJumpTruthy:       {"OpJumpTruthy", []int{8}},
	OpCurrentClosure:   {"OpCurrentClosure", []int{}},
	OpGetLocalCell:     {"OpGetLocalCell", []int{2}},
	OpGetFreeCell:      {"OpGetFreeCell", []int{1}},
	OpPushHandler:      {"OpPushHandler", []int{8}},
	OpPopHandler:       {"OpPopHandler", []int{}},
	OpThrow:            {"OpThrow", []int{}},
	OpJumpNotNull:      {"OpJumpNotNull", []int{8}},
	OpJumpNull:         {"OpJumpNull", []int{8}},
	OpDefer:            {"OpDefer", []int{2}},
	OpSpread:           {"OpSpread", []int{}},
	OpModule:           {"OpModule", []int{8}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 8:
			binary.BigEndian.PutUint64(instruction[offset:], uint64(o))
		case 4:
			binary.BigEndian.PutUint32(instruction[offset:], uint32(o))
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}

	return instruction
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		if offset+width > len(ins) {
			return operands, offset
		}
		switch width {
		case 8:
			operands[i] = int(binary.BigEndian.Uint64(ins[offset:]))
		case 4:
			operands[i] = int(binary.BigEndian.Uint32(ins[offset:]))
		case 2:
			operands[i] = int(binary.BigEndian.Uint16(ins[offset:]))
		case 1:
			operands[i] = int(ins[offset])
		}
		offset += width
	}

	return operands, offset
}

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			i++
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])
		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))
		i += 1 + read
	}

	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}

	switch operandCount {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])
	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}
