package object

import (
	"fmt"
	"sync"
)

type Future struct {
	Done   chan struct{}
	Value  Object
	Err    error
	Status string // "pending", "resolved", "rejected"
	Mu     sync.RWMutex
}

func (f *Future) Type() ObjectType { return FUTURE_OBJ }

func (f *Future) Inspect() string {
	f.Mu.RLock()
	defer f.Mu.RUnlock()
	if f.Status == "resolved" {
		if f.Value == nil {
			return "<future resolved: null>"
		}
		return fmt.Sprintf("<future resolved: %s>", f.Value.Inspect())
	} else if f.Status == "rejected" {
		return fmt.Sprintf("<future rejected: %v>", f.Err)
	}
	return "<future pending>"
}

func NewFuture() *Future {
	return &Future{
		Done:   make(chan struct{}),
		Status: "pending",
		Value:  &Null{},
	}
}

func (f *Future) Resolve(val Object) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.Status == "pending" {
		if val == nil {
			val = &Null{}
		}
		f.Value = val
		f.Status = "resolved"
		close(f.Done)
	}
}

func (f *Future) Reject(err error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.Status == "pending" {
		f.Err = err
		f.Status = "rejected"
		close(f.Done)
	}
}

func (f *Future) Cancel(reason string) {
	if reason == "" {
		reason = "future cancelled"
	}
	f.Reject(fmt.Errorf("%s", reason))
}

func (f *Future) IsDone() bool {
	f.Mu.RLock()
	defer f.Mu.RUnlock()
	return f.Status != "pending"
}

func (f *Future) Result() (Object, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()
	if f.Status == "pending" {
		return nil, fmt.Errorf("future is still pending")
	}
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Value, nil
}

func (f *Future) Await() (Object, error) {
	<-f.Done
	f.Mu.RLock()
	defer f.Mu.RUnlock()
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Value, nil
}
