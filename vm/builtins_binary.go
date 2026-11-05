package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"unicode"

	"bramandscript/object"
)

// Helper: parse struct format item
type structField struct {
	count int
	typeChar rune
}

func parseStructFormat(fmtStr string) (binary.ByteOrder, []structField, error) {
	runes := []rune(fmtStr)
	order := binary.ByteOrder(binary.BigEndian)
	idx := 0

	if len(runes) > 0 {
		switch runes[0] {
		case '>':
			order = binary.BigEndian
			idx++
		case '<':
			order = binary.LittleEndian
			idx++
		case '=', '@':
			order = binary.NativeEndian
			idx++
		case '!':
			order = binary.BigEndian
			idx++
		}
	}

	var fields []structField
	for idx < len(runes) {
		r := runes[idx]
		if unicode.IsSpace(r) {
			idx++
			continue
		}
		if unicode.IsDigit(r) {
			numStart := idx
			for idx < len(runes) && unicode.IsDigit(runes[idx]) {
				idx++
			}
			n, _ := strconv.Atoi(string(runes[numStart:idx]))
			if idx >= len(runes) {
				return nil, nil, fmt.Errorf("format ends with count without type")
			}
			tChar := runes[idx]
			idx++
			fields = append(fields, structField{count: n, typeChar: tChar})
		} else {
			fields = append(fields, structField{count: 1, typeChar: r})
			idx++
		}
	}
	return order, fields, nil
}

func calcStructSize(fmtStr string) (int, error) {
	_, fields, err := parseStructFormat(fmtStr)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, f := range fields {
		switch f.typeChar {
		case 'b', 'B', 'c', '?':
			total += f.count * 1
		case 'h', 'H':
			total += f.count * 2
		case 'i', 'I', 'l', 'L', 'f':
			total += f.count * 4
		case 'q', 'Q', 'd':
			total += f.count * 8
		case 's':
			total += f.count // string of length f.count
		default:
			return 0, fmt.Errorf("unsupported format character '%c'", f.typeChar)
		}
	}
	return total, nil
}

func init() {
	RegisterBuiltin("struct_calcsize", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			fmtStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: "struct_calcsize requires STRING format"}
			}
			sz, err := calcStructSize(fmtStr.Value)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			return &object.Integer{Value: int64(sz)}
		},
	})

	RegisterBuiltin("struct_pack", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "struct_pack requires at least 2 arguments: format, values"}
			}
			fmtStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: "first argument to struct_pack must be STRING format"}
			}
			order, fields, err := parseStructFormat(fmtStr.Value)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			// Gather value objects
			var valObjs []object.Object
			if len(args) == 2 {
				if arr, isArr := args[1].(*object.Array); isArr {
					arr.Mu.RLock()
					valObjs = append(valObjs, arr.Elements...)
					arr.Mu.RUnlock()
				} else {
					valObjs = append(valObjs, args[1])
				}
			} else {
				valObjs = append(valObjs, args[1:]...)
			}

			var buf bytes.Buffer
			valIdx := 0

			for _, f := range fields {
				if f.typeChar == 's' {
					if valIdx >= len(valObjs) {
						return &object.Error{Message: "insufficient arguments for struct_pack format"}
					}
					var strBytes []byte
					switch v := valObjs[valIdx].(type) {
					case *object.String:
						strBytes = []byte(v.Value)
					case *object.Bytes:
						v.Mu.RLock()
						strBytes = make([]byte, len(v.Value))
						copy(strBytes, v.Value)
						v.Mu.RUnlock()
					default:
						strBytes = []byte(valObjs[valIdx].Inspect())
					}
					valIdx++
					// Pad or truncate to f.count
					out := make([]byte, f.count)
					copy(out, strBytes)
					buf.Write(out)
					continue
				}

				reps := f.count
				for r := 0; r < reps; r++ {
					if valIdx >= len(valObjs) {
						return &object.Error{Message: "insufficient arguments for struct_pack format"}
					}
					curVal := valObjs[valIdx]
					valIdx++

					switch f.typeChar {
					case 'b':
						iVal := toInt64(curVal)
						buf.WriteByte(byte(int8(iVal)))
					case 'B':
						iVal := toInt64(curVal)
						buf.WriteByte(byte(uint8(iVal)))
					case 'c':
						sVal := curVal.Inspect()
						if strObj, isStr := curVal.(*object.String); isStr && len(strObj.Value) > 0 {
							sVal = strObj.Value
						}
						if len(sVal) > 0 {
							buf.WriteByte(sVal[0])
						} else {
							buf.WriteByte(0)
						}
					case '?':
						bVal := isTruthy(curVal)
						if bVal {
							buf.WriteByte(1)
						} else {
							buf.WriteByte(0)
						}
					case 'h':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, int16(iVal))
					case 'H':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, uint16(iVal))
					case 'i', 'l':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, int32(iVal))
					case 'I', 'L':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, uint32(iVal))
					case 'q':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, int64(iVal))
					case 'Q':
						iVal := toInt64(curVal)
						_ = binary.Write(&buf, order, uint64(iVal))
					case 'f':
						fVal, _ := toFloat(curVal)
						_ = binary.Write(&buf, order, float32(fVal))
					case 'd':
						fVal, _ := toFloat(curVal)
						_ = binary.Write(&buf, order, fVal)
					default:
						return &object.Error{Message: fmt.Sprintf("unsupported format char '%c'", f.typeChar)}
					}
				}
			}

			return object.NewBytes(buf.Bytes())
		},
	})

	RegisterBuiltin("struct_unpack", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("struct_unpack requires 2 arguments: format, data. got=%d", len(args))}
			}
			fmtStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: "first argument to struct_unpack must be STRING format"}
			}
			data, err := getBytesFromObj(args[1])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			order, fields, err := parseStructFormat(fmtStr.Value)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			reader := bytes.NewReader(data)
			var results []object.Object

			for _, f := range fields {
				if f.typeChar == 's' {
					strBytes := make([]byte, f.count)
					_, err := reader.Read(strBytes)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("struct_unpack read error: %v", err)}
					}
					// Trim trailing null bytes
					trimmed := bytes.TrimRight(strBytes, "\x00")
					results = append(results, &object.String{Value: string(trimmed)})
					continue
				}

				reps := f.count
				for r := 0; r < reps; r++ {
					switch f.typeChar {
					case 'b':
						var val int8
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'B':
						var val uint8
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'c':
						var val byte
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.String{Value: string([]byte{val})})
					case '?':
						var val byte
						_ = binary.Read(reader, order, &val)
						if val != 0 {
							results = append(results, True)
						} else {
							results = append(results, False)
						}
					case 'h':
						var val int16
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'H':
						var val uint16
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'i', 'l':
						var val int32
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'I', 'L':
						var val uint32
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'q':
						var val int64
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: val})
					case 'Q':
						var val uint64
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Integer{Value: int64(val)})
					case 'f':
						var val float32
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Float{Value: float64(val)})
					case 'd':
						var val float64
						_ = binary.Read(reader, order, &val)
						results = append(results, &object.Float{Value: val})
					default:
						return &object.Error{Message: fmt.Sprintf("unsupported format character '%c'", f.typeChar)}
					}
				}
			}

			return &object.Array{Elements: results}
		},
	})

	// MessagePack encode/decode builtins
	RegisterBuiltin("msgpack_encode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			var buf bytes.Buffer
			err := encodeMsgPack(args[0], &buf)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("msgpack encode error: %v", err)}
			}
			return object.NewBytes(buf.Bytes())
		},
	})

	RegisterBuiltin("msgpack_decode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			reader := bytes.NewReader(data)
			obj, err := decodeMsgPack(reader)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("msgpack decode error: %v", err)}
			}
			return obj
		},
	})
}

func toInt64(obj object.Object) int64 {
	switch v := obj.(type) {
	case *object.Integer:
		return v.Value
	case *object.Float:
		return int64(v.Value)
	default:
		return 0
	}
}

// MsgPack Serialization Implementation
func encodeMsgPack(obj object.Object, buf *bytes.Buffer) error {
	if obj == nil || obj == Null {
		buf.WriteByte(0xc0)
		return nil
	}

	switch v := obj.(type) {
	case *object.Boolean:
		if v.Value {
			buf.WriteByte(0xc3)
		} else {
			buf.WriteByte(0xc2)
		}
		return nil

	case *object.Integer:
		val := v.Value
		if val >= 0 && val <= 127 {
			buf.WriteByte(byte(val))
		} else if val >= -32 && val < 0 {
			buf.WriteByte(byte(val))
		} else if val >= math.MinInt8 && val <= math.MaxInt8 {
			buf.WriteByte(0xd0)
			buf.WriteByte(byte(int8(val)))
		} else if val >= math.MinInt16 && val <= math.MaxInt16 {
			buf.WriteByte(0xd1)
			_ = binary.Write(buf, binary.BigEndian, int16(val))
		} else if val >= math.MinInt32 && val <= math.MaxInt32 {
			buf.WriteByte(0xd2)
			_ = binary.Write(buf, binary.BigEndian, int32(val))
		} else {
			buf.WriteByte(0xd3)
			_ = binary.Write(buf, binary.BigEndian, val)
		}
		return nil

	case *object.Float:
		buf.WriteByte(0xcb)
		_ = binary.Write(buf, binary.BigEndian, v.Value)
		return nil

	case *object.String:
		b := []byte(v.Value)
		n := len(b)
		if n <= 31 {
			buf.WriteByte(byte(0xa0 | n))
		} else if n <= 255 {
			buf.WriteByte(0xd9)
			buf.WriteByte(byte(n))
		} else if n <= 65535 {
			buf.WriteByte(0xda)
			_ = binary.Write(buf, binary.BigEndian, uint16(n))
		} else {
			buf.WriteByte(0xdb)
			_ = binary.Write(buf, binary.BigEndian, uint32(n))
		}
		buf.Write(b)
		return nil

	case *object.Bytes:
		v.Mu.RLock()
		b := make([]byte, len(v.Value))
		copy(b, v.Value)
		v.Mu.RUnlock()
		n := len(b)
		if n <= 255 {
			buf.WriteByte(0xc4)
			buf.WriteByte(byte(n))
		} else if n <= 65535 {
			buf.WriteByte(0xc5)
			_ = binary.Write(buf, binary.BigEndian, uint16(n))
		} else {
			buf.WriteByte(0xc6)
			_ = binary.Write(buf, binary.BigEndian, uint32(n))
		}
		buf.Write(b)
		return nil

	case *object.Array:
		v.Mu.RLock()
		n := len(v.Elements)
		if n <= 15 {
			buf.WriteByte(byte(0x90 | n))
		} else if n <= 65535 {
			buf.WriteByte(0xdc)
			_ = binary.Write(buf, binary.BigEndian, uint16(n))
		} else {
			buf.WriteByte(0xdd)
			_ = binary.Write(buf, binary.BigEndian, uint32(n))
		}
		for _, el := range v.Elements {
			if err := encodeMsgPack(el, buf); err != nil {
				v.Mu.RUnlock()
				return err
			}
		}
		v.Mu.RUnlock()
		return nil

	case *object.Hash:
		v.Mu.RLock()
		n := len(v.Pairs)
		if n <= 15 {
			buf.WriteByte(byte(0x80 | n))
		} else if n <= 65535 {
			buf.WriteByte(0xde)
			_ = binary.Write(buf, binary.BigEndian, uint16(n))
		} else {
			buf.WriteByte(0xdf)
			_ = binary.Write(buf, binary.BigEndian, uint32(n))
		}
		for _, pair := range v.Pairs {
			if err := encodeMsgPack(pair.Key, buf); err != nil {
				v.Mu.RUnlock()
				return err
			}
			if err := encodeMsgPack(pair.Value, buf); err != nil {
				v.Mu.RUnlock()
				return err
			}
		}
		v.Mu.RUnlock()
		return nil

	default:
		// Fall back to inspect string
		s := obj.Inspect()
		b := []byte(s)
		buf.WriteByte(0xd9)
		buf.WriteByte(byte(len(b)))
		buf.Write(b)
		return nil
	}
}

func decodeMsgPack(r *bytes.Reader) (object.Object, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	// Positive fixint: 0x00 - 0x7f
	if b <= 0x7f {
		return &object.Integer{Value: int64(b)}, nil
	}
	// Negative fixint: 0xe0 - 0xff
	if b >= 0xe0 {
		return &object.Integer{Value: int64(int8(b))}, nil
	}
	// Fixstr: 0xa0 - 0xbf
	if b >= 0xa0 && b <= 0xbf {
		length := int(b & 0x1f)
		strBytes := make([]byte, length)
		_, err := r.Read(strBytes)
		if err != nil {
			return nil, err
		}
		return &object.String{Value: string(strBytes)}, nil
	}
	// Fixarray: 0x90 - 0x9f
	if b >= 0x90 && b <= 0x9f {
		length := int(b & 0x0f)
		elements := make([]object.Object, length)
		for i := 0; i < length; i++ {
			el, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			elements[i] = el
		}
		return &object.Array{Elements: elements}, nil
	}
	// Fixmap: 0x80 - 0x8f
	if b >= 0x80 && b <= 0x8f {
		length := int(b & 0x0f)
		pairs := make(map[object.HashKey]object.HashPair)
		for i := 0; i < length; i++ {
			k, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			v, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			hk, ok := k.(object.Hashable)
			if ok {
				pairs[hk.HashKey()] = object.HashPair{Key: k, Value: v}
			}
		}
		return &object.Hash{Pairs: pairs}, nil
	}

	switch b {
	case 0xc0:
		return Null, nil
	case 0xc2:
		return False, nil
	case 0xc3:
		return True, nil
	case 0xc4: // bin 8
		var length uint8
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return object.NewBytes(buf), nil
	case 0xc5: // bin 16
		var length uint16
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return object.NewBytes(buf), nil
	case 0xc6: // bin 32
		var length uint32
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return object.NewBytes(buf), nil
	case 0xcb: // float 64
		var val float64
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Float{Value: val}, nil
	case 0xcc: // uint 8
		var val uint8
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xcd: // uint 16
		var val uint16
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xce: // uint 32
		var val uint32
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xcf: // uint 64
		var val uint64
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xd0: // int 8
		var val int8
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xd1: // int 16
		var val int16
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xd2: // int 32
		var val int32
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: int64(val)}, nil
	case 0xd3: // int 64
		var val int64
		_ = binary.Read(r, binary.BigEndian, &val)
		return &object.Integer{Value: val}, nil
	case 0xd9: // str 8
		var length uint8
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return &object.String{Value: string(buf)}, nil
	case 0xda: // str 16
		var length uint16
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return &object.String{Value: string(buf)}, nil
	case 0xdb: // str 32
		var length uint32
		_ = binary.Read(r, binary.BigEndian, &length)
		buf := make([]byte, length)
		_, _ = r.Read(buf)
		return &object.String{Value: string(buf)}, nil
	case 0xdc: // array 16
		var length uint16
		_ = binary.Read(r, binary.BigEndian, &length)
		elements := make([]object.Object, length)
		for i := 0; i < int(length); i++ {
			el, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			elements[i] = el
		}
		return &object.Array{Elements: elements}, nil
	case 0xde: // map 16
		var length uint16
		_ = binary.Read(r, binary.BigEndian, &length)
		pairs := make(map[object.HashKey]object.HashPair)
		for i := 0; i < int(length); i++ {
			k, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			v, err := decodeMsgPack(r)
			if err != nil {
				return nil, err
			}
			hk, ok := k.(object.Hashable)
			if ok {
				pairs[hk.HashKey()] = object.HashPair{Key: k, Value: v}
			}
		}
		return &object.Hash{Pairs: pairs}, nil
	default:
		return nil, fmt.Errorf("unknown msgpack format byte: 0x%02x", b)
	}
}
