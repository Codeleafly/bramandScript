package formatter

import (
	"strings"
	"testing"
)

func TestFormatBasic(t *testing.T) {
	input := `
fn add(a,b){
let x=a+b;
return x;
}
`
	expected := `fn add(a, b) {
    let x = a + b;
    return x;
}
`
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}

	if strings.TrimSpace(formatted) != strings.TrimSpace(expected) {
		t.Errorf("Format mismatch.\nGot:\n%s\nWant:\n%s", formatted, expected)
	}
}

func TestFormatConditionalsAndComments(t *testing.T) {
	input := `
// This is a test function
fn check(x){
if(x>10){
// Greater branch
return true;
}else{
return false;
}
}
`
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}

	if !strings.Contains(formatted, "    if (x > 10) {") {
		t.Errorf("Expected formatted if statement with spacing, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "// Greater branch") {
		t.Errorf("Expected comments preserved, got:\n%s", formatted)
	}
}

func TestFormatParenBracketSpaces(t *testing.T) {
	input := `println( add( x, y ) )`
	expected := `println(add(x, y))`
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}
	if strings.TrimSpace(formatted) != strings.TrimSpace(expected) {
		t.Errorf("Format mismatch. Got %q, want %q", formatted, expected)
	}
}

func TestFormatCompoundAndBitwiseOperators(t *testing.T) {
	input := "let a = 1<<3;\nlet b = 16>>2;\nlet c = 2**8;\nlet d = [1, 2]|>sum();\nlet e = a&b|c^d;"
	expected := "let a = 1 << 3;\nlet b = 16 >> 2;\nlet c = 2 ** 8;\nlet d = [1, 2] |> sum();\nlet e = a & b | c ^ d;"
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}
	if strings.TrimSpace(formatted) != strings.TrimSpace(expected) {
		t.Errorf("Format mismatch.\nGot:\n%s\nWant:\n%s", formatted, expected)
	}
}

func TestFormatEscapedBackslashInStrings(t *testing.T) {
	input := `let SPINNER = ["-", "\\", "|", "/"];
let other = "normal";`
	expected := `let SPINNER = ["-", "\\", "|", "/"];
let other = "normal";
`
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}
	if formatted != expected {
		t.Errorf("Format mismatch.\nGot:\n%s\nWant:\n%s", formatted, expected)
	}
}

func TestFormatEnterpriseOperators(t *testing.T) {
	input := `a??=10;
let x = a??b;
let y = cond?1:2;
let z = obj?.prop;
`
	expected := `a ??= 10;
let x = a ?? b;
let y = cond ? 1 : 2;
let z = obj?.prop;
`
	formatted, err := Format(input)
	if err != nil {
		t.Fatalf("Format error: %s", err)
	}
	if formatted != expected {
		t.Errorf("Format mismatch.\nGot:\n%s\nWant:\n%s", formatted, expected)
	}
}
