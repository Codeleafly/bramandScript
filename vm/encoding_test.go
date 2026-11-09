package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"testing"
)

func runEncodingSource(t *testing.T, input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	c := compiler.New()
	err := c.Compile(prog)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	v := New(c.Bytecode())
	err = v.Run()
	if err != nil {
		t.Fatalf("vm error: %v", err)
	}
	return v.LastPoppedStackElem()
}

func TestHexAndUrlEncoding(t *testing.T) {
	input := `
	let h = hex_encode("hello world");
	let d = hex_decode(h);
	let u = url_encode("name=John Doe&city=New York");
	let du = url_decode(u);
	let q = url_parse_query("key=val&num=123");
	let bq = url_build_query({"a": "1", "b": "2"});

	[h, d, du, q["key"], q["num"]]
	`
	res := runEncodingSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "68656c6c6f20776f726c64" {
		t.Errorf("hex_encode failed: got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "hello world" {
		t.Errorf("hex_decode failed: got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.String).Value != "name=John Doe&city=New York" {
		t.Errorf("url decode failed: got %v", arr.Elements[2])
	}
	if arr.Elements[3].(*object.String).Value != "val" {
		t.Errorf("url_parse_query key failed: got %v", arr.Elements[3])
	}
	if arr.Elements[4].(*object.String).Value != "123" {
		t.Errorf("url_parse_query num failed: got %v", arr.Elements[4])
	}
}

func TestCsvEncoding(t *testing.T) {
	input := `
	let csv_raw = "id,name,role\n1,Alice,Engineer\n2,Bob,Architect";
	let parsed = csv_parse_records(csv_raw, ",", true);
	let row1 = parsed[0];
	let row2 = parsed[1];

	let back = csv_format_records(parsed, ",", ["id", "name", "role"]);

	[len(parsed), row1["name"], row2["role"], back]
	`
	res := runEncodingSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.Integer).Value != 2 {
		t.Errorf("expected 2 parsed records, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "Alice" {
		t.Errorf("expected Alice, got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.String).Value != "Architect" {
		t.Errorf("expected Architect, got %v", arr.Elements[2])
	}
}

func TestEncodingStdModule(t *testing.T) {
	input := `
	import "std/encoding" as enc;

	let h = enc.hex_encode("BramandScript");
	let orig = enc.hex_decode(h);
	let b64 = enc.base64_encode("Universal Encoding");
	let b64_dec = enc.base64_decode(b64);
	let chunked = enc.base64_chunk("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 10);

	[orig, b64_dec, len(chunked) > 0]
	`
	res := runEncodingSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "BramandScript" {
		t.Errorf("hex roundtrip failed: got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "Universal Encoding" {
		t.Errorf("base64 roundtrip failed: got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected chunked true, got %v", arr.Elements[2])
	}
}
