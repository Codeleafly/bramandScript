package vm

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
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

	// Day 40 Byte Buffers & Binary I/O (Agent 10)
	RegisterBuiltin("buf_alloc", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("buf_alloc expected 1 or 2 arguments (size, [fill_byte]), got %d", len(args))}
			}
			sizeObj, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("buf_alloc size must be an INTEGER, got %s", args[0].Type())}
			}
			size := sizeObj.Value
			if size < 0 {
				return &object.Error{Message: fmt.Sprintf("buf_alloc size cannot be negative: %d", size)}
			}
			if size > 1024*1024*1024 {
				return &object.Error{Message: "buf_alloc size exceeds 1GB limit"}
			}
			buf := make([]byte, size)
			if len(args) == 2 && args[1] != Null {
				var fill byte
				switch f := args[1].(type) {
				case *object.Integer:
					fill = byte(f.Value & 0xFF)
				case *object.String:
					if len(f.Value) > 0 {
						fill = f.Value[0]
					}
				}
				if fill != 0 {
					for i := range buf {
						buf[i] = fill
					}
				}
			}
			return &object.Bytes{Value: buf}
		},
	})

	RegisterBuiltin("buf_from_string", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("buf_from_string expected 1 or 2 arguments (str, [encoding]), got %d", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("buf_from_string str must be a STRING, got %s", args[0].Type())}
			}
			enc := "utf8"
			if len(args) == 2 && args[1] != Null {
				if e, ok := args[1].(*object.String); ok {
					enc = strings.ToLower(strings.TrimSpace(e.Value))
				}
			}
			rawStr := strObj.Value
			switch enc {
			case "hex", "base16":
				cleanHex := strings.TrimSpace(rawStr)
				cleanHex = strings.TrimPrefix(cleanHex, "0x")
				cleanHex = strings.TrimPrefix(cleanHex, "0X")
				if len(cleanHex)%2 != 0 {
					cleanHex = "0" + cleanHex
				}
				decoded, err := hex.DecodeString(cleanHex)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("hex decode error: %v", err)}
				}
				return &object.Bytes{Value: decoded}
			case "base64":
				cleanB64 := strings.TrimSpace(rawStr)
				decoded, err := base64.StdEncoding.DecodeString(cleanB64)
				if err != nil {
					if dec2, err2 := base64.RawStdEncoding.DecodeString(cleanB64); err2 == nil {
						return &object.Bytes{Value: dec2}
					}
					return &object.Error{Message: fmt.Sprintf("base64 decode error: %v", err)}
				}
				return &object.Bytes{Value: decoded}
			case "base64url":
				cleanB64 := strings.TrimSpace(rawStr)
				decoded, err := base64.URLEncoding.DecodeString(cleanB64)
				if err != nil {
					if dec2, err2 := base64.RawURLEncoding.DecodeString(cleanB64); err2 == nil {
						return &object.Bytes{Value: dec2}
					}
					return &object.Error{Message: fmt.Sprintf("base64url decode error: %v", err)}
				}
				return &object.Bytes{Value: decoded}
			default:
				buf := make([]byte, len(rawStr))
				copy(buf, rawStr)
				return &object.Bytes{Value: buf}
			}
		},
	})

	RegisterBuiltin("buf_from_array", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("buf_from_array expected 1 argument (arr), got %d", len(args))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("buf_from_array expected ARRAY, got %s", args[0].Type())}
			}
			arr.Mu.RLock()
			defer arr.Mu.RUnlock()
			buf := make([]byte, len(arr.Elements))
			for i, el := range arr.Elements {
				switch e := el.(type) {
				case *object.Integer:
					buf[i] = byte(e.Value & 0xFF)
				default:
					return &object.Error{Message: fmt.Sprintf("buf_from_array element at index %d must be INTEGER, got %s", i, el.Type())}
				}
			}
			return &object.Bytes{Value: buf}
		},
	})

	RegisterBuiltin("buf_to_string", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 4 {
				return &object.Error{Message: fmt.Sprintf("buf_to_string expected 1 to 4 arguments (buf, [encoding], [start], [end]), got %d", len(args))}
			}
			b, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			enc := "utf8"
			if len(args) >= 2 && args[1] != Null {
				if e, ok := args[1].(*object.String); ok {
					enc = strings.ToLower(strings.TrimSpace(e.Value))
				}
			}
			b.Mu.RLock()
			defer b.Mu.RUnlock()
			length := int64(len(b.Value))
			s := int64(0)
			e := length
			if len(args) >= 3 && args[2] != Null {
				if startObj, ok := args[2].(*object.Integer); ok {
					s = startObj.Value
					if s < 0 {
						s = length + s
						if s < 0 {
							s = 0
						}
					}
					if s > length {
						s = length
					}
				}
			}
			if len(args) >= 4 && args[3] != Null {
				if endObj, ok := args[3].(*object.Integer); ok {
					e = endObj.Value
					if e < 0 {
						e = length + e
						if e < 0 {
							e = 0
						}
					}
					if e > length {
						e = length
					}
					if e < s {
						e = s
					}
				}
			}
			slice := b.Value[s:e]
			switch enc {
			case "hex", "base16":
				return &object.String{Value: hex.EncodeToString(slice)}
			case "base64":
				return &object.String{Value: base64.StdEncoding.EncodeToString(slice)}
			case "base64url":
				return &object.String{Value: base64.URLEncoding.EncodeToString(slice)}
			default:
				return &object.String{Value: string(slice)}
			}
		},
	})

	RegisterBuiltin("buf_copy", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: fmt.Sprintf("buf_copy expected at least 2 arguments, got %d", len(args))}
			}
			target, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("target: %v", err)}
			}
			var source *object.Bytes
			tStart := int64(0)
			sStart := int64(0)
			var sEnd int64 = -1

			if len(args) == 2 {
				src, err := extractBytesObj(args[1])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("source: %v", err)}
				}
				source = src
			} else {
				if off, ok := args[1].(*object.Integer); ok {
					tStart = off.Value
				}
				src, err := extractBytesObj(args[2])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("source: %v", err)}
				}
				source = src
				if len(args) >= 4 && args[3] != Null {
					if sOff, ok := args[3].(*object.Integer); ok {
						sStart = sOff.Value
					}
				}
				if len(args) >= 5 && args[4] != Null {
					if eOff, ok := args[4].(*object.Integer); ok {
						sEnd = eOff.Value
					}
				}
			}

			target.Mu.Lock()
			defer target.Mu.Unlock()
			source.Mu.RLock()
			defer source.Mu.RUnlock()

			sLen := int64(len(source.Value))
			tLen := int64(len(target.Value))

			if sEnd < 0 || sEnd > sLen {
				sEnd = sLen
			}
			if sStart < 0 {
				sStart = 0
			}
			if sStart > sLen {
				sStart = sLen
			}
			if sEnd < sStart {
				sEnd = sStart
			}

			if tStart < 0 {
				tStart = 0
			}
			if tStart > tLen {
				tStart = tLen
			}

			toCopy := sEnd - sStart
			avail := tLen - tStart
			if toCopy > avail {
				toCopy = avail
			}
			if toCopy <= 0 {
				return &object.Integer{Value: 0}
			}
			n := copy(target.Value[tStart:], source.Value[sStart:sStart+toCopy])
			return &object.Integer{Value: int64(n)}
		},
	})

	RegisterBuiltin("buf_fill", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 4 {
				return &object.Error{Message: fmt.Sprintf("buf_fill expected 2 to 4 arguments (buf, value, [start], [end]), got %d", len(args))}
			}
			b, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			var fillVal byte
			switch v := args[1].(type) {
			case *object.Integer:
				fillVal = byte(v.Value & 0xFF)
			case *object.String:
				if len(v.Value) > 0 {
					fillVal = v.Value[0]
				}
			default:
				return &object.Error{Message: fmt.Sprintf("buf_fill value must be INTEGER or STRING, got %s", args[1].Type())}
			}

			b.Mu.Lock()
			defer b.Mu.Unlock()

			length := int64(len(b.Value))
			s := int64(0)
			e := length

			if len(args) >= 3 && args[2] != Null {
				if sObj, ok := args[2].(*object.Integer); ok {
					s = sObj.Value
					if s < 0 {
						s = 0
					}
					if s > length {
						s = length
					}
				}
			}
			if len(args) >= 4 && args[3] != Null {
				if eObj, ok := args[3].(*object.Integer); ok {
					e = eObj.Value
					if e < s {
						e = s
					}
					if e > length {
						e = length
					}
				}
			}
			for i := s; i < e; i++ {
				b.Value[i] = fillVal
			}
			return b
		},
	})

	RegisterBuiltin("buf_read_u8", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("buf_read_u8 expected 2 arguments (buf, offset), got %d", len(args))}
			}
			b, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			off, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("buf_read_u8 offset must be an INTEGER, got %s", args[1].Type())}
			}
			b.Mu.RLock()
			defer b.Mu.RUnlock()
			if off.Value < 0 || off.Value >= int64(len(b.Value)) {
				return &object.Error{Message: fmt.Sprintf("buf_read_u8 offset %d out of bounds (length %d)", off.Value, len(b.Value))}
			}
			return &object.Integer{Value: int64(b.Value[off.Value])}
		},
	})

	RegisterBuiltin("buf_write_u8", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Message: fmt.Sprintf("buf_write_u8 expected 3 arguments (buf, val, offset), got %d", len(args))}
			}
			b, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			v1, ok1 := args[1].(*object.Integer)
			v2, ok2 := args[2].(*object.Integer)
			if !ok1 || !ok2 {
				return &object.Error{Message: "buf_write_u8 arguments must be integers"}
			}
			val := v1.Value
			offset := v2.Value
			b.Mu.Lock()
			defer b.Mu.Unlock()
			l := int64(len(b.Value))
			if (offset < 0 || offset >= l) && (val >= 0 && val < l) && (offset >= 0 && offset <= 255) {
				val, offset = offset, val
			}
			if offset < 0 || offset >= l {
				return &object.Error{Message: fmt.Sprintf("buf_write_u8 offset %d out of bounds (length %d)", offset, l)}
			}
			b.Value[offset] = byte(val & 0xFF)
			return b
		},
	})

	RegisterBuiltin("buf_read_u16_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readUint16(args, binary.BigEndian, "buf_read_u16_be")
		},
	})
	RegisterBuiltin("buf_read_u16_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readUint16(args, binary.LittleEndian, "buf_read_u16_le")
		},
	})
	RegisterBuiltin("buf_write_u16_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeUint16(args, binary.BigEndian, "buf_write_u16_be")
		},
	})
	RegisterBuiltin("buf_write_u16_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeUint16(args, binary.LittleEndian, "buf_write_u16_le")
		},
	})

	RegisterBuiltin("buf_read_u32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readUint32(args, binary.BigEndian, "buf_read_u32_be")
		},
	})
	RegisterBuiltin("buf_read_u32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readUint32(args, binary.LittleEndian, "buf_read_u32_le")
		},
	})
	RegisterBuiltin("buf_write_u32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeUint32(args, binary.BigEndian, "buf_write_u32_be")
		},
	})
	RegisterBuiltin("buf_write_u32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeUint32(args, binary.LittleEndian, "buf_write_u32_le")
		},
	})

	RegisterBuiltin("buf_read_i32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readInt32(args, binary.BigEndian, "buf_read_i32_be")
		},
	})
	RegisterBuiltin("buf_read_i32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readInt32(args, binary.LittleEndian, "buf_read_i32_le")
		},
	})
	RegisterBuiltin("buf_write_i32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeInt32(args, binary.BigEndian, "buf_write_i32_be")
		},
	})
	RegisterBuiltin("buf_write_i32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeInt32(args, binary.LittleEndian, "buf_write_i32_le")
		},
	})

	RegisterBuiltin("buf_read_f32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readFloat32(args, binary.BigEndian, "buf_read_f32_be")
		},
	})
	RegisterBuiltin("buf_read_f32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readFloat32(args, binary.LittleEndian, "buf_read_f32_le")
		},
	})
	RegisterBuiltin("buf_write_f32_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeFloat32(args, binary.BigEndian, "buf_write_f32_be")
		},
	})
	RegisterBuiltin("buf_write_f32_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeFloat32(args, binary.LittleEndian, "buf_write_f32_le")
		},
	})

	RegisterBuiltin("buf_read_f64_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readFloat64(args, binary.BigEndian, "buf_read_f64_be")
		},
	})
	RegisterBuiltin("buf_read_f64_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readFloat64(args, binary.LittleEndian, "buf_read_f64_le")
		},
	})
	RegisterBuiltin("buf_write_f64_be", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeFloat64(args, binary.BigEndian, "buf_write_f64_be")
		},
	})
	RegisterBuiltin("buf_write_f64_le", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return writeFloat64(args, binary.LittleEndian, "buf_write_f64_le")
		},
	})

	RegisterBuiltin("buf_compare", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("buf_compare expected 2 arguments, got %d", len(args))}
			}
			b1, err1 := extractBytesObj(args[0])
			if err1 != nil {
				return &object.Error{Message: fmt.Sprintf("buf1: %v", err1)}
			}
			b2, err2 := extractBytesObj(args[1])
			if err2 != nil {
				return &object.Error{Message: fmt.Sprintf("buf2: %v", err2)}
			}
			b1.Mu.RLock()
			defer b1.Mu.RUnlock()
			b2.Mu.RLock()
			defer b2.Mu.RUnlock()
			res := bytes.Compare(b1.Value, b2.Value)
			return &object.Integer{Value: int64(res)}
		},
	})

	RegisterBuiltin("buf_concat", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 0 {
				return &object.Bytes{Value: []byte{}}
			}
			var bufList []*object.Bytes
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					arr.Mu.RLock()
					for _, el := range arr.Elements {
						b, err := extractBytesObj(el)
						if err == nil {
							bufList = append(bufList, b)
						}
					}
					arr.Mu.RUnlock()
				} else {
					b, err := extractBytesObj(args[0])
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					bufList = append(bufList, b)
				}
			} else {
				for _, arg := range args {
					b, err := extractBytesObj(arg)
					if err == nil {
						bufList = append(bufList, b)
					}
				}
			}

			totalLen := 0
			for _, b := range bufList {
				b.Mu.RLock()
				totalLen += len(b.Value)
				b.Mu.RUnlock()
			}
			res := make([]byte, totalLen)
			offset := 0
			for _, b := range bufList {
				b.Mu.RLock()
				copy(res[offset:], b.Value)
				offset += len(b.Value)
				b.Mu.RUnlock()
			}
			return &object.Bytes{Value: res}
		},
	})

	RegisterBuiltin("buf_index_of", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Message: fmt.Sprintf("buf_index_of expected 2 or 3 arguments (buf, needle, [offset]), got %d", len(args))}
			}
			b, err := extractBytesObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			var needle []byte
			switch n := args[1].(type) {
			case *object.Integer:
				needle = []byte{byte(n.Value & 0xFF)}
			case *object.String:
				needle = []byte(n.Value)
			case *object.Bytes:
				n.Mu.RLock()
				needle = make([]byte, len(n.Value))
				copy(needle, n.Value)
				n.Mu.RUnlock()
			case *object.Hash:
				nb, err := extractBytesObj(n)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("needle: %v", err)}
				}
				nb.Mu.RLock()
				needle = make([]byte, len(nb.Value))
				copy(needle, nb.Value)
				nb.Mu.RUnlock()
			default:
				return &object.Error{Message: fmt.Sprintf("unsupported needle type: %s", args[1].Type())}
			}

			offset := int64(0)
			if len(args) == 3 && args[2] != Null {
				if off, ok := args[2].(*object.Integer); ok {
					offset = off.Value
					if offset < 0 {
						offset = 0
					}
				}
			}

			b.Mu.RLock()
			defer b.Mu.RUnlock()
			l := int64(len(b.Value))
			if offset >= l {
				if len(needle) == 0 && offset == l {
					return &object.Integer{Value: offset}
				}
				return &object.Integer{Value: -1}
			}
			if len(needle) == 0 {
				return &object.Integer{Value: offset}
			}

			idx := bytes.Index(b.Value[offset:], needle)
			if idx == -1 {
				return &object.Integer{Value: -1}
			}
			return &object.Integer{Value: offset + int64(idx)}
		},
	})
}

func extractBytesObj(obj object.Object) (*object.Bytes, error) {
	if obj == nil {
		return nil, fmt.Errorf("expected byte buffer, got null")
	}
	switch v := obj.(type) {
	case *object.Bytes:
		return v, nil
	case *object.Hash:
		v.Mu.RLock()
		defer v.Mu.RUnlock()
		for _, key := range []string{"_raw", "raw", "buf", "bytes"} {
			hk := (&object.String{Value: key}).HashKey()
			if pair, ok := v.Pairs[hk]; ok {
				if b, ok := pair.Value.(*object.Bytes); ok {
					return b, nil
				}
			}
		}
		return nil, fmt.Errorf("hash does not wrap a byte buffer")
	case *object.String:
		return &object.Bytes{Value: []byte(v.Value)}, nil
	default:
		return nil, fmt.Errorf("expected BYTES or Buffer object, got %s", obj.Type())
	}
}

func readUint16(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("%s expected 2 arguments (buf, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	off, ok := args[1].(*object.Integer)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER, got %s", name, args[1].Type())}
	}
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	offset := off.Value
	if offset < 0 || offset+2 > int64(len(b.Value)) {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, len(b.Value))}
	}
	val := order.Uint16(b.Value[offset:])
	return &object.Integer{Value: int64(val)}
}

func writeUint16(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 3 {
		return &object.Error{Message: fmt.Sprintf("%s expected 3 arguments (buf, val, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	v1, ok1 := args[1].(*object.Integer)
	v2, ok2 := args[2].(*object.Integer)
	if !ok1 || !ok2 {
		return &object.Error{Message: fmt.Sprintf("%s arguments must be integers", name)}
	}
	val := v1.Value
	offset := v2.Value
	b.Mu.Lock()
	defer b.Mu.Unlock()
	l := int64(len(b.Value))
	if (offset < 0 || offset+2 > l) && (val >= 0 && val+2 <= l) {
		val, offset = offset, val
	}
	if offset < 0 || offset+2 > l {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, l)}
	}
	order.PutUint16(b.Value[offset:], uint16(val))
	return b
}

func readUint32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("%s expected 2 arguments (buf, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	off, ok := args[1].(*object.Integer)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER, got %s", name, args[1].Type())}
	}
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	offset := off.Value
	if offset < 0 || offset+4 > int64(len(b.Value)) {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, len(b.Value))}
	}
	val := order.Uint32(b.Value[offset:])
	return &object.Integer{Value: int64(val)}
}

func writeUint32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 3 {
		return &object.Error{Message: fmt.Sprintf("%s expected 3 arguments (buf, val, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	v1, ok1 := args[1].(*object.Integer)
	v2, ok2 := args[2].(*object.Integer)
	if !ok1 || !ok2 {
		return &object.Error{Message: fmt.Sprintf("%s arguments must be integers", name)}
	}
	val := v1.Value
	offset := v2.Value
	b.Mu.Lock()
	defer b.Mu.Unlock()
	l := int64(len(b.Value))
	if (offset < 0 || offset+4 > l) && (val >= 0 && val+4 <= l) {
		val, offset = offset, val
	}
	if offset < 0 || offset+4 > l {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, l)}
	}
	order.PutUint32(b.Value[offset:], uint32(val))
	return b
}

func readInt32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("%s expected 2 arguments (buf, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	off, ok := args[1].(*object.Integer)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER, got %s", name, args[1].Type())}
	}
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	offset := off.Value
	if offset < 0 || offset+4 > int64(len(b.Value)) {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, len(b.Value))}
	}
	val := int32(order.Uint32(b.Value[offset:]))
	return &object.Integer{Value: int64(val)}
}

func writeInt32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 3 {
		return &object.Error{Message: fmt.Sprintf("%s expected 3 arguments (buf, val, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	v1, ok1 := args[1].(*object.Integer)
	v2, ok2 := args[2].(*object.Integer)
	if !ok1 || !ok2 {
		return &object.Error{Message: fmt.Sprintf("%s arguments must be integers", name)}
	}
	val := v1.Value
	offset := v2.Value
	b.Mu.Lock()
	defer b.Mu.Unlock()
	l := int64(len(b.Value))
	if (offset < 0 || offset+4 > l) && (val >= 0 && val+4 <= l) {
		val, offset = offset, val
	}
	if offset < 0 || offset+4 > l {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, l)}
	}
	order.PutUint32(b.Value[offset:], uint32(int32(val)))
	return b
}

func readFloat32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("%s expected 2 arguments (buf, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	off, ok := args[1].(*object.Integer)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER, got %s", name, args[1].Type())}
	}
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	offset := off.Value
	if offset < 0 || offset+4 > int64(len(b.Value)) {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, len(b.Value))}
	}
	bits := order.Uint32(b.Value[offset:])
	f := math.Float32frombits(bits)
	return &object.Float{Value: float64(f)}
}

func writeFloat32(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 3 {
		return &object.Error{Message: fmt.Sprintf("%s expected 3 arguments (buf, val, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	var fVal float32
	var offset int64
	if f1, ok := args[1].(*object.Float); ok {
		fVal = float32(f1.Value)
		if off, ok := args[2].(*object.Integer); ok {
			offset = off.Value
		} else {
			return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER", name)}
		}
	} else if f2, ok := args[2].(*object.Float); ok {
		fVal = float32(f2.Value)
		if off, ok := args[1].(*object.Integer); ok {
			offset = off.Value
		} else {
			return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER", name)}
		}
	} else {
		v1, ok1 := args[1].(*object.Integer)
		v2, ok2 := args[2].(*object.Integer)
		if !ok1 || !ok2 {
			return &object.Error{Message: fmt.Sprintf("%s expected numeric value and integer offset", name)}
		}
		fVal = float32(v1.Value)
		offset = v2.Value
		b.Mu.RLock()
		l := int64(len(b.Value))
		b.Mu.RUnlock()
		if (offset < 0 || offset+4 > l) && (int64(fVal) >= 0 && int64(fVal)+4 <= l) {
			fVal, offset = float32(offset), int64(fVal)
		}
	}

	b.Mu.Lock()
	defer b.Mu.Unlock()
	l := int64(len(b.Value))
	if offset < 0 || offset+4 > l {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, l)}
	}
	bits := math.Float32bits(fVal)
	order.PutUint32(b.Value[offset:], bits)
	return b
}

func readFloat64(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("%s expected 2 arguments (buf, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	off, ok := args[1].(*object.Integer)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER, got %s", name, args[1].Type())}
	}
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	offset := off.Value
	if offset < 0 || offset+8 > int64(len(b.Value)) {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, len(b.Value))}
	}
	bits := order.Uint64(b.Value[offset:])
	f := math.Float64frombits(bits)
	return &object.Float{Value: f}
}

func writeFloat64(args []object.Object, order binary.ByteOrder, name string) object.Object {
	if len(args) != 3 {
		return &object.Error{Message: fmt.Sprintf("%s expected 3 arguments (buf, val, offset), got %d", name, len(args))}
	}
	b, err := extractBytesObj(args[0])
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	var fVal float64
	var offset int64
	if f1, ok := args[1].(*object.Float); ok {
		fVal = f1.Value
		if off, ok := args[2].(*object.Integer); ok {
			offset = off.Value
		} else {
			return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER", name)}
		}
	} else if f2, ok := args[2].(*object.Float); ok {
		fVal = f2.Value
		if off, ok := args[1].(*object.Integer); ok {
			offset = off.Value
		} else {
			return &object.Error{Message: fmt.Sprintf("%s offset must be an INTEGER", name)}
		}
	} else {
		v1, ok1 := args[1].(*object.Integer)
		v2, ok2 := args[2].(*object.Integer)
		if !ok1 || !ok2 {
			return &object.Error{Message: fmt.Sprintf("%s expected numeric value and integer offset", name)}
		}
		fVal = float64(v1.Value)
		offset = v2.Value
		b.Mu.RLock()
		l := int64(len(b.Value))
		b.Mu.RUnlock()
		if (offset < 0 || offset+8 > l) && (int64(fVal) >= 0 && int64(fVal)+8 <= l) {
			fVal, offset = float64(offset), int64(fVal)
		}
	}

	b.Mu.Lock()
	defer b.Mu.Unlock()
	l := int64(len(b.Value))
	if offset < 0 || offset+8 > l {
		return &object.Error{Message: fmt.Sprintf("%s offset %d out of bounds (length %d)", name, offset, l)}
	}
	bits := math.Float64bits(fVal)
	order.PutUint64(b.Value[offset:], bits)
	return b
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
