package vm

import (
	"bramandscript/code"
	"bramandscript/object"
)

type ExceptionHandler struct {
	CatchIP int
	SP      int
}

type DeferredCall struct {
	Callee object.Object
	Args   []object.Object
}

type Frame struct {
	cl          *object.Closure
	ip          int
	basePointer int
	handlers    []ExceptionHandler
	defers      []DeferredCall
	contract    *object.PackageContract
}

func NewFrame(cl *object.Closure, basePointer int) *Frame {
	var contract *object.PackageContract
	if cl != nil && cl.Fn != nil {
		contract = cl.Fn.Contract
	}
	return &Frame{
		cl:          cl,
		ip:          -1,
		basePointer: basePointer,
		contract:    contract,
	}
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Fn.Instructions
}

func (f *Frame) PushHandler(catchIP, sp int) {
	f.handlers = append(f.handlers, ExceptionHandler{CatchIP: catchIP, SP: sp})
}

func (f *Frame) PopHandler() (ExceptionHandler, bool) {
	if len(f.handlers) == 0 {
		return ExceptionHandler{}, false
	}
	h := f.handlers[len(f.handlers)-1]
	f.handlers = f.handlers[:len(f.handlers)-1]
	return h, true
}

func (f *Frame) PushDefer(d DeferredCall) {
	f.defers = append(f.defers, d)
}

func (f *Frame) PopDefer() (DeferredCall, bool) {
	if len(f.defers) == 0 {
		return DeferredCall{}, false
	}
	d := f.defers[len(f.defers)-1]
	f.defers = f.defers[:len(f.defers)-1]
	return d, true
}
