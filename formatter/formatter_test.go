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
