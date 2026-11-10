package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"bramandscript/object"
)

func toFloat(obj object.Object) (float64, bool) {
	switch o := obj.(type) {
	case *object.Float:
		return o.Value, true
	case *object.Integer:
		return float64(o.Value), true
	default:
		return 0, false
	}
}

func nativeToBramandObject(val interface{}) object.Object {
	if val == nil {
		return Null
	}
	switch v := val.(type) {
	case bool:
		return nativeBoolToBooleanObject(v)
	case float64:
		if math.Floor(v) == v && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -9223372036854775808 && v <= 9223372036854775807 {
			return &object.Integer{Value: int64(v)}
		}
		return &object.Float{Value: v}
	case int64:
		return &object.Integer{Value: v}
	case int:
		return &object.Integer{Value: int64(v)}
	case string:
		return &object.String{Value: v}
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, el := range v {
			elements[i] = nativeToBramandObject(el)
		}
		return &object.Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[object.HashKey]object.HashPair)
		for k, val := range v {
			keyObj := &object.String{Value: k}
			valObj := nativeToBramandObject(val)
			pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: valObj}
		}
		return &object.Hash{Pairs: pairs}
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}

func bramandObjectToNative(obj object.Object) interface{} {
	switch o := obj.(type) {
	case *object.Integer:
		return o.Value
	case *object.Float:
		return o.Value
	case *object.Boolean:
		return o.Value
	case *object.String:
		return o.Value
	case *object.Null:
		return nil
	case *object.Array:
		o.Mu.RLock()
		defer o.Mu.RUnlock()
		res := make([]interface{}, len(o.Elements))
		for i, el := range o.Elements {
			res[i] = bramandObjectToNative(el)
		}
		return res
	case *object.Hash:
		o.Mu.RLock()
		defer o.Mu.RUnlock()
		res := make(map[string]interface{})
		for _, pair := range o.Pairs {
			res[pair.Key.Inspect()] = bramandObjectToNative(pair.Value)
		}
		return res
	case *object.Vector:
		res := make([]float64, len(o.Elements))
		copy(res, o.Elements)
		return res
	case *object.Matrix:
		return o.Data
	case *object.BigInt:
		return o.Value.String()
	case *object.BigFloat:
		return o.Value.Text('g', -1)
	default:
		return o.Inspect()
	}
}

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"type": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: string(args[0].Type())}
			},
		},
		"int": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					return &object.Integer{Value: int64(arg.Value)}
				case *object.String:
					s := strings.TrimSpace(arg.Value)
					base := 10
					clean := s
					if strings.HasPrefix(clean, "+") || strings.HasPrefix(clean, "-") {
						clean = clean[1:]
					}
					if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") ||
						strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") ||
						strings.HasPrefix(clean, "0o") || strings.HasPrefix(clean, "0O") {
						base = 0
					}
					v, err := strconv.ParseInt(s, base, 64)
					if err != nil {
						bi := new(big.Int)
						if bi, ok := bi.SetString(s, base); ok {
							return &object.BigInt{Value: bi}
						}
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to integer", arg.Value)}
					}
					return &object.Integer{Value: v}
				case *object.Boolean:
					if arg.Value {
						return &object.Integer{Value: 1}
					}
					return &object.Integer{Value: 0}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to integer", arg.Type())}
				}
			},
		},
		"float": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Float:
					return arg
				case *object.Integer:
					return &object.Float{Value: float64(arg.Value)}
				case *object.String:
					v, err := strconv.ParseFloat(arg.Value, 64)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to float", arg.Value)}
					}
					return &object.Float{Value: v}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to float", arg.Type())}
				}
			},
		},
		"str": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				if b, ok := args[0].(*object.Bytes); ok {
					b.Mu.RLock()
					defer b.Mu.RUnlock()
					return &object.String{Value: string(b.Value)}
				}
				return &object.String{Value: args[0].Inspect()}
			},
		},
		"bool": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return nativeBoolToBooleanObject(isTruthy(args[0]))
			},
		},
		"bytes": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) == 0 {
					return &object.Bytes{Value: []byte{}}
				}
				if len(args) > 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to bytes. got=%d, want 0 or 1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Integer:
					if arg.Value < 0 {
						return &object.Error{Message: fmt.Sprintf("bytes length cannot be negative: %d", arg.Value)}
					}
					if arg.Value > 1024*1024*1024 {
						return &object.Error{Message: "bytes buffer size exceeds 1GB limit"}
					}
					return &object.Bytes{Value: make([]byte, arg.Value)}
				case *object.String:
					buf := make([]byte, len(arg.Value))
					copy(buf, arg.Value)
					return &object.Bytes{Value: buf}
				case *object.Array:
					arg.Mu.RLock()
					defer arg.Mu.RUnlock()
					buf := make([]byte, len(arg.Elements))
					for i, elem := range arg.Elements {
						switch e := elem.(type) {
						case *object.Integer:
							buf[i] = byte(e.Value)
						default:
							return &object.Error{Message: fmt.Sprintf("bytes array element at index %d must be INTEGER, got %s", i, elem.Type())}
						}
					}
					return &object.Bytes{Value: buf}
				case *object.Bytes:
					arg.Mu.RLock()
					defer arg.Mu.RUnlock()
					buf := make([]byte, len(arg.Value))
					copy(buf, arg.Value)
					return &object.Bytes{Value: buf}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot construct bytes from %s", arg.Type())}
				}
			},
		},
		"read_u8": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("read_u8 expected 2 arguments (bytes, offset), got %d", len(args))}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("read_u8 first argument must be BYTES, got %s", args[0].Type())}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("read_u8 offset must be an INTEGER, got %s", args[1].Type())}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value >= int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u8 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				return &object.Integer{Value: int64(b.Value[offset.Value])}
			},
		},
		"write_u8": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 3 {
					return &object.Error{Message: fmt.Sprintf("write_u8 expected 3 arguments (bytes, offset, value), got %d", len(args))}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("write_u8 first argument must be BYTES, got %s", args[0].Type())}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("write_u8 offset must be an INTEGER, got %s", args[1].Type())}
				}
				val, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("write_u8 value must be an INTEGER, got %s", args[2].Type())}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value >= int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u8 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				b.Value[offset.Value] = byte(val.Value)
				return b
			},
		},
		"read_u16": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("read_u16 expected 2 or 3 arguments (bytes, offset, [endian]), got %d", len(args))}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("read_u16 first argument must be BYTES, got %s", args[0].Type())}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 3 {
					if e, ok := args[2].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value+2 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u16 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				var val uint16
				if endian == "le" {
					val = binary.LittleEndian.Uint16(b.Value[offset.Value:])
				} else {
					val = binary.BigEndian.Uint16(b.Value[offset.Value:])
				}
				return &object.Integer{Value: int64(val)}
			},
		},
		"write_u16": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 3 || len(args) > 4 {
					return &object.Error{Message: "write_u16 expected 3 or 4 arguments (bytes, offset, value, [endian])"}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: "write_u16 first argument must be BYTES"}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 4 {
					if e, ok := args[3].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value+2 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u16 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				if endian == "le" {
					binary.LittleEndian.PutUint16(b.Value[offset.Value:], uint16(val.Value))
				} else {
					binary.BigEndian.PutUint16(b.Value[offset.Value:], uint16(val.Value))
				}
				return b
			},
		},
		"read_u32": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: "read_u32 expected 2 or 3 arguments (bytes, offset, [endian])"}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: "read_u32 first argument must be BYTES"}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 3 {
					if e, ok := args[2].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value+4 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u32 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				var val uint32
				if endian == "le" {
					val = binary.LittleEndian.Uint32(b.Value[offset.Value:])
				} else {
					val = binary.BigEndian.Uint32(b.Value[offset.Value:])
				}
				return &object.Integer{Value: int64(val)}
			},
		},
		"write_u32": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 3 || len(args) > 4 {
					return &object.Error{Message: "write_u32 expected 3 or 4 arguments (bytes, offset, value, [endian])"}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: "write_u32 first argument must be BYTES"}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 4 {
					if e, ok := args[3].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value+4 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u32 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				if endian == "le" {
					binary.LittleEndian.PutUint32(b.Value[offset.Value:], uint32(val.Value))
				} else {
					binary.BigEndian.PutUint32(b.Value[offset.Value:], uint32(val.Value))
				}
				return b
			},
		},
		"read_u64": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: "read_u64 expected 2 or 3 arguments (bytes, offset, [endian])"}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: "read_u64 first argument must be BYTES"}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 3 {
					if e, ok := args[2].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value+8 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u64 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				var val uint64
				if endian == "le" {
					val = binary.LittleEndian.Uint64(b.Value[offset.Value:])
				} else {
					val = binary.BigEndian.Uint64(b.Value[offset.Value:])
				}
				return &object.Integer{Value: int64(val)}
			},
		},
		"write_u64": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 3 || len(args) > 4 {
					return &object.Error{Message: "write_u64 expected 3 or 4 arguments (bytes, offset, value, [endian])"}
				}
				b, ok := args[0].(*object.Bytes)
				if !ok {
					return &object.Error{Message: "write_u64 first argument must be BYTES"}
				}
				offset, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[2].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 4 {
					if e, ok := args[3].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value+8 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u64 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				if endian == "le" {
					binary.LittleEndian.PutUint64(b.Value[offset.Value:], uint64(val.Value))
				} else {
					binary.BigEndian.PutUint64(b.Value[offset.Value:], uint64(val.Value))
				}
				return b
			},
		},
		"ord": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to ord. got=%d, want=1", len(args))}
				}
				str, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("ord() expected STRING, got %s", args[0].Type())}
				}
				runes := []rune(str.Value)
				if len(runes) == 0 {
					return &object.Error{Message: "ord() expected a non-empty string"}
				}
				return &object.Integer{Value: int64(runes[0])}
			},
		},
		"chr": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to chr. got=%d, want=1", len(args))}
				}
				code, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("chr() expected INTEGER, got %s", args[0].Type())}
				}
				if code.Value < 0 || code.Value > 0x10FFFF || (code.Value >= 0xD800 && code.Value <= 0xDFFF) {
					return &object.Error{Message: fmt.Sprintf("chr() code point %d out of valid Unicode range (0..0x10FFFF, excluding surrogates)", code.Value)}
				}
				return &object.String{Value: string(rune(code.Value))}
			},
		},
	})
}
