package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"os"
	"strings"
	"testing"
)

func runFsPathSource(t *testing.T, input string) object.Object {
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

func TestPathBuiltins(t *testing.T) {
	input := `
	let j = path_join("a", "b", "c.txt");
	let c = path_clean("a/b/../b/./c.txt");
	let b = path_base("/dir/test.txt");
	let d = path_dir("/dir/test.txt");
	let ext = path_ext("archive.tar.gz");
	let is_abs = path_is_abs("/usr/local/bin");
	let spl = path_split("path/to/file.txt");

	[j, c, b, d, ext, is_abs, spl[0], spl[1]]
	`
	res := runFsPathSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "a/b/c.txt" {
		t.Errorf("path_join failed: got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "a/b/c.txt" {
		t.Errorf("path_clean failed: got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.String).Value != "test.txt" {
		t.Errorf("path_base failed: got %v", arr.Elements[2])
	}
	if arr.Elements[3].(*object.String).Value != "/dir" {
		t.Errorf("path_dir failed: got %v", arr.Elements[3])
	}
	if arr.Elements[4].(*object.String).Value != ".gz" {
		t.Errorf("path_ext failed: got %v", arr.Elements[4])
	}
	if arr.Elements[5] != True {
		t.Errorf("path_is_abs failed: got %v", arr.Elements[5])
	}
	if arr.Elements[6].(*object.String).Value != "path/to/" {
		t.Errorf("path_split dir failed: got %v", arr.Elements[6])
	}
	if arr.Elements[7].(*object.String).Value != "file.txt" {
		t.Errorf("path_split file failed: got %v", arr.Elements[7])
	}
}

func TestFsBuiltins(t *testing.T) {
	input := `
	let tmp = fs_temp_file("test_", ".txt");
	write_file(tmp["path"], "Hello Enterprise FS");
	let exists_before = file_exists(tmp["path"]);
	let st = fs_stat(tmp["path"]);
	let tree = fs_tree(".", 1);
	delete_file(tmp["path"]);
	let exists_after = file_exists(tmp["path"]);

	[exists_before, st["size"] > 0, tree["path"] == ".", exists_after]
	`
	res := runFsPathSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0] != True {
		t.Errorf("expected exists_before true, got %v", arr.Elements[0])
	}
	if arr.Elements[1] != True {
		t.Errorf("expected size > 0, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected tree path ., got %v", arr.Elements[2])
	}
	if arr.Elements[3] != False {
		t.Errorf("expected exists_after false, got %v", arr.Elements[3])
	}
}

func TestFsAndPathStdModules(t *testing.T) {
	input := `
	import "std/fs" as fs;
	import "std/path" as path;

	let p = path.join("dir", "sub", "item.bmd");
	let b = path.base(p);
	let ext = path.ext(p);
	let tmp = fs.temp_file("unit_", ".tmp");
	fs.write_text(tmp["path"], "sample");
	let content = fs.read_text(tmp["path"]);
	fs.remove(tmp["path"]);

	[p, b, ext, content]
	`
	res := runFsPathSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "dir/sub/item.bmd" {
		t.Errorf("path.join failed: got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "item.bmd" {
		t.Errorf("path.base failed: got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.String).Value != ".bmd" {
		t.Errorf("path.ext failed: got %v", arr.Elements[2])
	}
	if arr.Elements[3].(*object.String).Value != "sample" {
		t.Errorf("fs read/write failed: got %v", arr.Elements[3])
	}
	_ = os.Remove("unit_.tmp")
	_ = strings.Contains("", "")
}
