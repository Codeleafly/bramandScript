package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
)

type ObjectType string

const (
	INTEGER_OBJ              = "INTEGER"
	FLOAT_OBJ                = "FLOAT"
	BOOLEAN_OBJ              = "BOOLEAN"
	STRING_OBJ               = "STRING"
	NULL_OBJ                 = "NULL"
	RETURN_VALUE_OBJ         = "RETURN_VALUE"
	ERROR_OBJ                = "ERROR"
	COMPILED_FUNCTION_OBJ    = "COMPILED_FUNCTION"
	CLOSURE_OBJ              = "CLOSURE"
	BUILTIN_OBJ              = "BUILTIN"
	ARRAY_OBJ                = "ARRAY"
	HASH_OBJ                 = "HASH"
	CHANNEL_OBJ              = "CHANNEL"
	MUTEX_OBJ                = "MUTEX"
	WAIT_GROUP_OBJ           = "WAIT_GROUP"
	BIG_INT_OBJ              = "BIG_INT"
	BIG_FLOAT_OBJ            = "BIG_FLOAT"
	VECTOR_OBJ               = "VECTOR"
	MATRIX_OBJ               = "MATRIX"
	TCP_SERVER_OBJ           = "TCP_SERVER"
	TCP_CONN_OBJ             = "TCP_CONN"
	UDP_SOCKET_OBJ           = "UDP_SOCKET"
	HTTP_SERVER_OBJ          = "HTTP_SERVER"
	HTTP_RESPONSE_WRITER_OBJ = "HTTP_RESPONSE_WRITER"
	WS_CONN_OBJ              = "WS_CONN"
	WS_SERVER_OBJ            = "WS_SERVER"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Hashable interface {
	HashKey() HashKey
}

type HashKey struct {
	Type  ObjectType
	Value uint64
}

// ---------------- Integer ----------------
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

// ---------------- Float ----------------
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

// ---------------- Boolean ----------------
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) HashKey() HashKey {
	var val uint64
	if b.Value {
		val = 1
	} else {
		val = 0
	}
	return HashKey{Type: b.Type(), Value: val}
}

// ---------------- String ----------------
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: h.Sum64()}
}

// ---------------- Null ----------------
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// ---------------- ReturnValue ----------------
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// ---------------- Error ----------------
type Error struct {
	Message string
	Fatal   bool
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "Runtime Error: " + e.Message }

// ---------------- CompiledFunction ----------------
type CompiledFunction struct {
	Instructions  []byte
	NumLocals     int
	NumParameters int
	SourceMap     []int
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", cf)
}

// ---------------- Closure ----------------
type Closure struct {
	Fn   *CompiledFunction
	Free []Object
}

func (c *Closure) Type() ObjectType { return CLOSURE_OBJ }
func (c *Closure) Inspect() string {
	return fmt.Sprintf("Closure[%p]", c)
}

// ---------------- Builtin ----------------
type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// ---------------- Array ----------------
type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	var elements []string
	for _, e := range a.Elements {
		elements = append(elements, e.Inspect())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// ---------------- Hash ----------------
type HashPair struct {
	Key   Object
	Value Object
}

type Hash struct {
	Pairs map[HashKey]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	var out bytes.Buffer
	var pairs []string
	for _, pair := range h.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", pair.Key.Inspect(), pair.Value.Inspect()))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// ---------------- Channel ----------------
type Channel struct {
	Ch       chan Object
	Capacity int
	closed   bool
	mu       sync.Mutex
}

func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string {
	return fmt.Sprintf("chan(%d)", c.Capacity)
}

func (c *Channel) Send(val Object) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("send on closed channel")
	}
	ch := c.Ch
	c.mu.Unlock()

	ch <- val
	return nil
}

func (c *Channel) Recv() (Object, bool) {
	val, ok := <-c.Ch
	if !ok {
		return &Null{}, false
	}
	return val, true
}

func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("close of closed channel")
	}
	c.closed = true
	close(c.Ch)
	return nil
}

// ---------------- Mutex ----------------
type Mutex struct {
	mu sync.Mutex
}

func (m *Mutex) Type() ObjectType { return MUTEX_OBJ }
func (m *Mutex) Inspect() string  { return "mutex" }

func (m *Mutex) Lock() {
	m.mu.Lock()
}

func (m *Mutex) Unlock() {
	m.mu.Unlock()
}

func (m *Mutex) TryLock() bool {
	return m.mu.TryLock()
}

// ---------------- WaitGroup ----------------
type WaitGroup struct {
	wg sync.WaitGroup
}

func (w *WaitGroup) Type() ObjectType { return WAIT_GROUP_OBJ }
func (w *WaitGroup) Inspect() string  { return "wait_group" }

func (w *WaitGroup) Add(delta int) {
	w.wg.Add(delta)
}

func (w *WaitGroup) Done() {
	w.wg.Done()
}

func (w *WaitGroup) Wait() {
	w.wg.Wait()
}
