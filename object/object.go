package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"
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
	BYTES_OBJ                = "BYTES"
	CELL_OBJ                 = "CELL"
	SPREAD_WRAPPER_OBJ       = "SPREAD_WRAPPER"
	PACKAGE_CONTRACT_OBJ     = "PACKAGE_CONTRACT"
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

// ---------------- Bytes ----------------
type Bytes struct {
	Value []byte
	Mu    sync.RWMutex
}

func NewBytes(data []byte) *Bytes {
	return &Bytes{Value: data}
}

func (b *Bytes) Type() ObjectType { return BYTES_OBJ }
func (b *Bytes) Inspect() string {
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	limit := len(b.Value)
	if limit > 16 {
		limit = 16
	}
	hexParts := make([]string, limit)
	for i := 0; i < limit; i++ {
		hexParts[i] = fmt.Sprintf("%02x", b.Value[i])
	}
	if len(b.Value) > 16 {
		return fmt.Sprintf("bytes([%s, ... %d bytes])", strings.Join(hexParts, " "), len(b.Value))
	}
	return fmt.Sprintf("bytes([%s])", strings.Join(hexParts, " "))
}
func (b *Bytes) HashKey() HashKey {
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	h := fnv.New64a()
	h.Write(b.Value)
	return HashKey{Type: b.Type(), Value: h.Sum64()}
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

// ---------------- StackFrameInfo ----------------
type StackFrameInfo struct {
	Function string
	Line     int
	File     string
}

// ---------------- Error ----------------
type Error struct {
	Message    string
	Fatal      bool
	StackTrace []StackFrameInfo
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "Runtime Error: " + e.Message }

func (e *Error) FormatStackTrace() string {
	var sb strings.Builder
	sb.WriteString("Runtime Error: ")
	sb.WriteString(e.Message)
	if len(e.StackTrace) > 0 {
		for _, frame := range e.StackTrace {
			sb.WriteString("\n  at ")
			fn := frame.Function
			if fn == "" {
				fn = "<anonymous>"
			}
			sb.WriteString(fn)
			if frame.Line > 0 {
				sb.WriteString(fmt.Sprintf(" (line %d)", frame.Line))
			}
		}
	}
	return sb.String()
}

// ---------------- CompiledFunction ----------------
type CompiledFunction struct {
	Name          string
	Instructions  []byte
	NumLocals     int
	NumParameters int
	SourceMap     []int
	FreeNames     []string
	Contract      *PackageContract
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", cf)
}

// ---------------- PackageContract ----------------
type PackageContract struct {
	PackageName     string
	Network         bool
	FilesystemRead  bool
	FilesystemWrite bool
	Subprocess      bool
	Declared        bool
}

func (p *PackageContract) Type() ObjectType { return PACKAGE_CONTRACT_OBJ }
func (p *PackageContract) Inspect() string {
	return fmt.Sprintf("PackageContract[%s: net=%t, fs_r=%t, fs_w=%t, subproc=%t]",
		p.PackageName, p.Network, p.FilesystemRead, p.FilesystemWrite, p.Subprocess)
}

// ---------------- Closure ----------------
type Closure struct {
	Fn       *CompiledFunction
	Free     []Object
	Instance *Hash
}

func (c *Closure) Type() ObjectType { return CLOSURE_OBJ }
func (c *Closure) Inspect() string {
	return fmt.Sprintf("Closure[%p]", c)
}

// ---------------- Cell ----------------
type Cell struct {
	Mu    sync.RWMutex
	Value Object
}

func (c *Cell) Type() ObjectType { return CELL_OBJ }
func (c *Cell) Inspect() string {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	if c.Value == nil {
		return "cell(null)"
	}
	return fmt.Sprintf("cell(%s)", c.Value.Inspect())
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
	Mu       sync.RWMutex
	ReadOnly bool
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.inspectDepth(0)
}

func (a *Array) inspectDepth(depth int) string {
	if depth >= 16 {
		return "[...]"
	}
	var out bytes.Buffer
	var elements []string
	for _, e := range a.Elements {
		if e == nil {
			elements = append(elements, "null")
		} else if subArr, ok := e.(*Array); ok {
			elements = append(elements, subArr.inspectDepth(depth+1))
		} else if subHash, ok := e.(*Hash); ok {
			elements = append(elements, subHash.inspectDepth(depth+1))
		} else {
			elements = append(elements, e.Inspect())
		}
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
	Mu       sync.RWMutex
	Pairs    map[HashKey]HashPair
	ReadOnly bool
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	return h.inspectDepth(0)
}

func (h *Hash) inspectDepth(depth int) string {
	if depth >= 16 {
		return "{...}"
	}
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if strPair, ok := h.Pairs[(&String{Value: "__str__"}).HashKey()]; ok {
		if s, ok := strPair.Value.(*String); ok {
			return s.Value
		}
	}

	var out bytes.Buffer
	var pairs []string
	for _, pair := range h.Pairs {
		keyStr := "null"
		if pair.Key != nil {
			keyStr = pair.Key.Inspect()
		}
		var valStr string
		if pair.Value == nil {
			valStr = "null"
		} else if subArr, ok := pair.Value.(*Array); ok {
			valStr = subArr.inspectDepth(depth + 1)
		} else if subHash, ok := pair.Value.(*Hash); ok {
			valStr = subHash.inspectDepth(depth + 1)
		} else {
			valStr = pair.Value.Inspect()
		}
		pairs = append(pairs, fmt.Sprintf("%s: %s", keyStr, valStr))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

func (h *Hash) HashKey() HashKey {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	if hashPair, ok := h.Pairs[(&String{Value: "__hash__"}).HashKey()]; ok {
		if intObj, ok := hashPair.Value.(*Integer); ok {
			return HashKey{Type: h.Type(), Value: uint64(intObj.Value)}
		}
	}
	hasher := fnv.New64a()
	hasher.Write([]byte(fmt.Sprintf("%p", h)))
	return HashKey{Type: h.Type(), Value: hasher.Sum64()}
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

func (c *Channel) Send(val Object) (err error) {
	return c.SendTimeout(val, 0)
}

func (c *Channel) SendTimeout(val Object, timeout time.Duration) (err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("send on closed channel")
	}
	ch := c.Ch
	c.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("send on closed channel: %v", r)
		}
	}()

	if timeout <= 0 {
		ch <- val
		return nil
	}

	select {
	case ch <- val:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("channel send timed out after %v", timeout)
	}
}

func (c *Channel) TrySend(val Object) (sent bool) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	ch := c.Ch
	c.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case ch <- val:
		return true
	default:
		return false
	}
}

func (c *Channel) Recv() (Object, bool) {
	return c.RecvTimeout(0)
}

func (c *Channel) RecvTimeout(timeout time.Duration) (Object, bool) {
	if timeout <= 0 {
		val, ok := <-c.Ch
		if !ok {
			return &Null{}, false
		}
		return val, true
	}
	select {
	case val, ok := <-c.Ch:
		if !ok {
			return &Null{}, false
		}
		return val, true
	case <-time.After(timeout):
		return &Null{}, false
	}
}

func (c *Channel) TryRecv() (Object, bool) {
	select {
	case val, ok := <-c.Ch:
		if !ok {
			return &Null{}, false
		}
		return val, true
	default:
		return &Null{}, false
	}
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
	defer func() {
		_ = recover()
	}()
	w.wg.Done()
}

func (w *WaitGroup) Wait() {
	w.wg.Wait()
}

// ---------------- SpreadWrapper ----------------
type SpreadWrapper struct {
	Value Object
}

func (sw *SpreadWrapper) Type() ObjectType { return SPREAD_WRAPPER_OBJ }
func (sw *SpreadWrapper) Inspect() string {
	if sw.Value != nil {
		return "..." + sw.Value.Inspect()
	}
	return "..."
}
