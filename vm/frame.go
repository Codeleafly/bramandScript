package vm

import (
	"bramandscript/code"
	"bramandscript/object"
)

type Frame struct {
	cl          *object.CompiledFunction
	ip          int
	basePointer int
}

func NewFrame(cl *object.CompiledFunction, basePointer int) *Frame {
	return &Frame{
		cl:          cl,
		ip:          -1,
		basePointer: basePointer,
	}
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Instructions
}
