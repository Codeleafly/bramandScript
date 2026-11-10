package vm

import (
	"math"
	"testing"

	"bramandscript/object"
)

func TestBufferBuiltins(t *testing.T) {
	// 1. buf_alloc
	allocFn := GetBuiltin("buf_alloc")
	bObj := allocFn.Fn(&object.Integer{Value: 16}, &object.Integer{Value: 0xAA})
	b, ok := bObj.(*object.Bytes)
	if !ok {
		t.Fatalf("expected *object.Bytes, got %T", bObj)
	}
	if len(b.Value) != 16 {
		t.Fatalf("expected len 16, got %d", len(b.Value))
	}
	for i, val := range b.Value {
		if val != 0xAA {
			t.Fatalf("byte %d expected 0xAA, got %x", i, val)
		}
	}

	// 2. buf_from_string (utf8, hex, base64)
	fromStrFn := GetBuiltin("buf_from_string")
	bUtf8 := fromStrFn.Fn(&object.String{Value: "Hello Bramand"})
	if string(bUtf8.(*object.Bytes).Value) != "Hello Bramand" {
		t.Fatalf("utf8 mismatch: %s", string(bUtf8.(*object.Bytes).Value))
	}

	bHex := fromStrFn.Fn(&object.String{Value: "deadbeef"}, &object.String{Value: "hex"})
	hBytes := bHex.(*object.Bytes).Value
	if len(hBytes) != 4 || hBytes[0] != 0xDE || hBytes[1] != 0xAD || hBytes[2] != 0xBE || hBytes[3] != 0xEF {
		t.Fatalf("hex mismatch: %x", hBytes)
	}

	bB64 := fromStrFn.Fn(&object.String{Value: "SGVsbG8="}, &object.String{Value: "base64"})
	if string(bB64.(*object.Bytes).Value) != "Hello" {
		t.Fatalf("base64 mismatch: %s", string(bB64.(*object.Bytes).Value))
	}

	// 3. buf_from_array
	fromArrFn := GetBuiltin("buf_from_array")
	bArr := fromArrFn.Fn(&object.Array{Elements: []object.Object{
		&object.Integer{Value: 1},
		&object.Integer{Value: 2},
		&object.Integer{Value: 3},
	}})
	arrBytes := bArr.(*object.Bytes).Value
	if len(arrBytes) != 3 || arrBytes[0] != 1 || arrBytes[1] != 2 || arrBytes[2] != 3 {
		t.Fatalf("arr mismatch: %v", arrBytes)
	}

	// 4. buf_to_string
	toStrFn := GetBuiltin("buf_to_string")
	sUtf8 := toStrFn.Fn(bHex, &object.String{Value: "hex"})
	if sUtf8.(*object.String).Value != "deadbeef" {
		t.Fatalf("to_string hex mismatch: %s", sUtf8.(*object.String).Value)
	}

	sB64 := toStrFn.Fn(bB64, &object.String{Value: "base64"})
	if sB64.(*object.String).Value != "SGVsbG8=" {
		t.Fatalf("to_string base64 mismatch: %s", sB64.(*object.String).Value)
	}

	// 5. buf_copy
	copyFn := GetBuiltin("buf_copy")
	target := allocFn.Fn(&object.Integer{Value: 8}, &object.Integer{Value: 0}).(*object.Bytes)
	n := copyFn.Fn(target, &object.Integer{Value: 2}, bHex, &object.Integer{Value: 0}, &object.Integer{Value: 4})
	if n.(*object.Integer).Value != 4 {
		t.Fatalf("copy returned %d want 4", n.(*object.Integer).Value)
	}
	if target.Value[2] != 0xDE || target.Value[5] != 0xEF {
		t.Fatalf("copy content mismatch: %x", target.Value)
	}

	// 6. buf_fill
	fillFn := GetBuiltin("buf_fill")
	fillFn.Fn(target, &object.Integer{Value: 0xFF}, &object.Integer{Value: 0}, &object.Integer{Value: 2})
	if target.Value[0] != 0xFF || target.Value[1] != 0xFF {
		t.Fatalf("fill mismatch: %x", target.Value)
	}

	// 7. Numeric codecs
	buf := allocFn.Fn(&object.Integer{Value: 32}, &object.Integer{Value: 0}).(*object.Bytes)

	// u8
	wU8 := GetBuiltin("buf_write_u8")
	rU8 := GetBuiltin("buf_read_u8")
	wU8.Fn(buf, &object.Integer{Value: 250}, &object.Integer{Value: 0})
	if rU8.Fn(buf, &object.Integer{Value: 0}).(*object.Integer).Value != 250 {
		t.Fatalf("u8 mismatch")
	}

	// u16 BE/LE
	wU16BE := GetBuiltin("buf_write_u16_be")
	rU16BE := GetBuiltin("buf_read_u16_be")
	wU16LE := GetBuiltin("buf_write_u16_le")
	rU16LE := GetBuiltin("buf_read_u16_le")

	wU16BE.Fn(buf, &object.Integer{Value: 0x1234}, &object.Integer{Value: 1})
	if rU16BE.Fn(buf, &object.Integer{Value: 1}).(*object.Integer).Value != 0x1234 {
		t.Fatalf("u16 be mismatch")
	}

	wU16LE.Fn(buf, &object.Integer{Value: 0xABCD}, &object.Integer{Value: 3})
	if rU16LE.Fn(buf, &object.Integer{Value: 3}).(*object.Integer).Value != 0xABCD {
		t.Fatalf("u16 le mismatch")
	}

	// u32 BE/LE
	wU32BE := GetBuiltin("buf_write_u32_be")
	rU32BE := GetBuiltin("buf_read_u32_be")
	wU32LE := GetBuiltin("buf_write_u32_le")
	rU32LE := GetBuiltin("buf_read_u32_le")

	wU32BE.Fn(buf, &object.Integer{Value: 0x11223344}, &object.Integer{Value: 5})
	if rU32BE.Fn(buf, &object.Integer{Value: 5}).(*object.Integer).Value != 0x11223344 {
		t.Fatalf("u32 be mismatch")
	}

	wU32LE.Fn(buf, &object.Integer{Value: 0xAABBCCDD}, &object.Integer{Value: 9})
	if uint32(rU32LE.Fn(buf, &object.Integer{Value: 9}).(*object.Integer).Value) != 0xAABBCCDD {
		t.Fatalf("u32 le mismatch")
	}

	// i32 BE/LE
	wI32BE := GetBuiltin("buf_write_i32_be")
	rI32BE := GetBuiltin("buf_read_i32_be")
	wI32LE := GetBuiltin("buf_write_i32_le")
	rI32LE := GetBuiltin("buf_read_i32_le")

	wI32BE.Fn(buf, &object.Integer{Value: -42}, &object.Integer{Value: 13})
	if rI32BE.Fn(buf, &object.Integer{Value: 13}).(*object.Integer).Value != -42 {
		t.Fatalf("i32 be mismatch")
	}

	wI32LE.Fn(buf, &object.Integer{Value: -123456}, &object.Integer{Value: 17})
	if rI32LE.Fn(buf, &object.Integer{Value: 17}).(*object.Integer).Value != -123456 {
		t.Fatalf("i32 le mismatch")
	}

	// f32 BE/LE
	wF32BE := GetBuiltin("buf_write_f32_be")
	rF32BE := GetBuiltin("buf_read_f32_be")
	wF32BE.Fn(buf, &object.Float{Value: 1.5}, &object.Integer{Value: 21})
	f32Val := rF32BE.Fn(buf, &object.Integer{Value: 21}).(*object.Float).Value
	if math.Abs(f32Val-1.5) > 1e-6 {
		t.Fatalf("f32 be mismatch: %f", f32Val)
	}

	// f64 BE/LE
	wF64BE := GetBuiltin("buf_write_f64_be")
	rF64BE := GetBuiltin("buf_read_f64_be")
	wF64BE.Fn(buf, &object.Float{Value: 3.1415926535}, &object.Integer{Value: 24})
	f64Val := rF64BE.Fn(buf, &object.Integer{Value: 24}).(*object.Float).Value
	if math.Abs(f64Val-3.1415926535) > 1e-9 {
		t.Fatalf("f64 be mismatch: %f", f64Val)
	}

	// 8. buf_compare
	cmpFn := GetBuiltin("buf_compare")
	b1 := fromStrFn.Fn(&object.String{Value: "abc"}).(*object.Bytes)
	b2 := fromStrFn.Fn(&object.String{Value: "abd"}).(*object.Bytes)
	b3 := fromStrFn.Fn(&object.String{Value: "abc"}).(*object.Bytes)
	if cmpFn.Fn(b1, b2).(*object.Integer).Value != -1 {
		t.Fatalf("compare b1, b2 want -1")
	}
	if cmpFn.Fn(b2, b1).(*object.Integer).Value != 1 {
		t.Fatalf("compare b2, b1 want 1")
	}
	if cmpFn.Fn(b1, b3).(*object.Integer).Value != 0 {
		t.Fatalf("compare b1, b3 want 0")
	}

	// 9. buf_concat
	concatFn := GetBuiltin("buf_concat")
	cBuf := concatFn.Fn(&object.Array{Elements: []object.Object{b1, b2}}).(*object.Bytes)
	if string(cBuf.Value) != "abcabd" {
		t.Fatalf("concat want 'abcabd', got %s", string(cBuf.Value))
	}

	// 10. buf_index_of
	idxFn := GetBuiltin("buf_index_of")
	haystack := fromStrFn.Fn(&object.String{Value: "hello world bramand"}).(*object.Bytes)
	idx1 := idxFn.Fn(haystack, &object.String{Value: "world"}).(*object.Integer).Value
	if idx1 != 6 {
		t.Fatalf("index_of 'world' want 6 got %d", idx1)
	}
	idx2 := idxFn.Fn(haystack, &object.String{Value: "missing"}).(*object.Integer).Value
	if idx2 != -1 {
		t.Fatalf("index_of 'missing' want -1 got %d", idx2)
	}
}
