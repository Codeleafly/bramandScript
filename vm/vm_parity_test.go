package vm

import (
	"os"
	"strings"
	"testing"

	"bramandscript/compiler"
	"bramandscript/object"
)

func TestAllCompilerBuiltinsRegisteredInVM(t *testing.T) {
	for i, def := range compiler.Builtins {
		b := getBuiltinByName(def.Name)
		if b == nil {
			t.Errorf("builtin [%d] %q defined in compiler but not registered in VM", i, def.Name)
		}
	}
	builtinRegistryMu.RLock()
	defer builtinRegistryMu.RUnlock()
	for name := range builtinRegistry {
		found := false
		for _, def := range compiler.Builtins {
			if def.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Note: builtin %q is registered in VM but not listed in compiler.Builtins", name)
		}
	}
}

func TestDynamicCodeEvaluation(t *testing.T) {
	tests := []vmTestCase{
		{`eval("10 + 20")`, 30},
		{`eval("let x = 5; let y = 7; x * y")`, 35},
		{`eval("str(len([1, 2, 3]))")`, "3"},
	}
	runVmTests(t, tests)
}

func TestEvalSessionAndDisasm(t *testing.T) {
	input := `
		let sess = create_session();
		eval("let x = 42;", sess);
		eval("let y = 8;", sess);
		let res = eval("x + y;", sess);
		let v = session_vars(sess);
		let d = disasm("1 + 2");
		[res, v["x"], v["y"], len(d) > 0]
	`
	program := parse(input)
	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compile error: %s", err)
	}
	vm := New(comp.Bytecode())
	err = vm.Run()
	if err != nil {
		t.Fatalf("vm error: %s", err)
	}
	stackElem := vm.LastPoppedStackElem()
	arr, ok := stackElem.(*object.Array)
	if !ok || len(arr.Elements) != 4 {
		t.Fatalf("expected 4 elements, got %s", stackElem.Inspect())
	}
	if arr.Elements[0].Inspect() != "50" {
		t.Errorf("expected 50, got %s", arr.Elements[0].Inspect())
	}
	if arr.Elements[1].Inspect() != "42" {
		t.Errorf("expected 42, got %s", arr.Elements[1].Inspect())
	}
	if arr.Elements[2].Inspect() != "8" {
		t.Errorf("expected 8, got %s", arr.Elements[2].Inspect())
	}
	if arr.Elements[3].Inspect() != "true" {
		t.Errorf("expected true, got %s", arr.Elements[3].Inspect())
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len([1, 2, 3])`, 3},
		{`type("universe")`, "STRING"},
		{`type(42)`, "INTEGER"},
		{`len(push([1, 2], 3))`, 3},
		{`let a = [1, 2]; push(a, 3); len(a)`, 3},
		{`let a = [1, 2]; push(a, 3); a[2]`, 3},
		{`int("42")`, 42},
		{`float("3.14")`, 3.14},
		{`str(100)`, "100"},
		{`bool(1)`, true},
		{`bool(null)`, false},
		{`len([1, 2] + [3, 4, 5])`, 5},
		{`([10, 20] + [30, 40])[2]`, 30},
		{`len(slice([1, 2, 3, 4, 5], 1, 4))`, 3},
	}
	runVmTests(t, tests)
}

func TestModuleImport(t *testing.T) {
	modCode := `
	fn add(a, b) {
		return a + b
	}
	let multiplier = 5
	`
	os.WriteFile("test_calc.bmd", []byte(modCode), 0644)
	defer os.Remove("test_calc.bmd")

	tests := []vmTestCase{
		{
			input: `
			import "test_calc.bmd"
			add(20, 22)
			`,
			expected: 42,
		},
		{
			input: `
			import "test_calc.bmd" as calc
			calc.add(10, calc.multiplier)
			`,
			expected: 15,
		},
	}
	runVmTests(t, tests)
}

func TestFileIO(t *testing.T) {
	testFile := "test_day4_io.txt"
	defer os.Remove(testFile)

	tests := []vmTestCase{
		{
			input: `
			write_file("test_day4_io.txt", "BramandScript Day 4");
			file_exists("test_day4_io.txt");
			`,
			expected: true,
		},
		{
			input: `
			read_file("test_day4_io.txt");
			`,
			expected: "BramandScript Day 4",
		},
		{
			input: `
			append_file("test_day4_io.txt", " Concurrency");
			read_file("test_day4_io.txt");
			`,
			expected: "BramandScript Day 4 Concurrency",
		},
		{
			input: `
			delete_file("test_day4_io.txt");
			file_exists("test_day4_io.txt");
			`,
			expected: false,
		},
		{
			input: `
			is_dir(".");
			`,
			expected: true,
		},
		{
			input: `
			let files = list_dir(".");
			len(files) > 0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestProcessExecutionAndEnv(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			set_env("BMD_DAY4_TEST", "verified");
			env("BMD_DAY4_TEST");
			`,
			expected: "verified",
		},
		{
			input: `
			let res = exec("echo", "bramand");
			res.code;
			`,
			expected: 0,
		},
		{
			input: `
			let res = exec("echo", "test_output");
			res.stdout;
			`,
			expected: "test_output\n",
		},
	}
	runVmTests(t, tests)
}

func TestStandardIOAndFormatting(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let colored = color("green", "OK");
			colored == "\033[32mOK\033[0m";
			`,
			expected: true,
		},
		{
			input: `
			let styledText = styled("HELLO", "bold");
			styledText == "\033[1mHELLO\033[0m";
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestBigIntAndBigFloat(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let a = big_int("1000000000000000000000000000000");
			let b = big_int("2000000000000000000000000000000");
			let c = a + b;
			c.str();
			`,
			expected: "3000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("1000000000000000000000000000000");
			let b = big_int("3000000000000000000000000000000");
			let c = b - a;
			c.str();
			`,
			expected: "2000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("1000000000000000");
			let b = big_int("2000000000000000");
			let c = a * b;
			c.str();
			`,
			expected: "2000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("100000000000000000000");
			let b = big_int("10000000000");
			let c = a / b;
			c.str();
			`,
			expected: "10000000000",
		},
		{
			input: `
			let a = big_int("105");
			let b = big_int("10");
			let c = a % b;
			c.str();
			`,
			expected: "5",
		},
		{
			input: `
			let a = big_int("2");
			let b = big_int("100");
			let c = a ** b;
			c.str();
			`,
			expected: "1267650600228229401496703205376",
		},
		{
			input: `
			let a = big_int("100");
			let b = big_int("200");
			a < b && b > a && a != b && (a == big_int("100"));
			`,
			expected: true,
		},
		{
			input: `
			let a = big_int("50");
			let b = -a;
			b.str();
			`,
			expected: "-50",
		},
		{
			input: `
			let a = big_int("-99");
			a.abs().str();
			`,
			expected: "99",
		},
		{
			input: `
			let a = big_float("1.25");
			let b = big_float("2.5");
			let c = a + b;
			c.str(3);
			`,
			expected: "3.75",
		},
		{
			input: `
			let a = big_float("10.5");
			let b = big_float("2.5");
			let c = a - b;
			c.str(3);
			`,
			expected: "8",
		},
		{
			input: `
			let a = big_float("2.5");
			let b = big_float("4.0");
			let c = a * b;
			c.str(3);
			`,
			expected: "10",
		},
		{
			input: `
			let a = big_float("15.0");
			let b = big_float("2.0");
			let c = a / b;
			c.str(3);
			`,
			expected: "7.5",
		},
		{
			input: `
			let a = big_float("16.0");
			let b = a.sqrt();
			b.str(3);
			`,
			expected: "4",
		},
		{
			input: `
			let a = big_float("1.5");
			let b = big_float("2.5");
			a < b && b > a && a != b && (a == big_float("1.5"));
			`,
			expected: true,
		},
		{
			input: `
			let x = 99999999999999999999999999999 ** 2;
			x.str();
			`,
			expected: "9999999999999999999999999999800000000000000000000000000001",
		},
	}
	runVmTests(t, tests)
}

func TestBigIntPowerOverflowSafety(t *testing.T) {
	input := `let x = 99999999999999999999999999999 ** 99999999999999999999999;`
	program := parse(input)
	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("unexpected compiler error: %s", err)
	}
	vm := New(comp.Bytecode())
	err = vm.Run()
	if err == nil {
		t.Fatalf("expected exponent overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "exponent too large") {
		t.Fatalf("expected 'exponent too large' error, got %s", err)
	}
}

func TestVectorsAndMatrices(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let v = vec([1.0, 2.0, 3.0]);
			v.dim;
			`,
			expected: 3,
		},
		{
			input: `
			let v = vec([10.0, 20.0, 30.0]);
			v[1];
			`,
			expected: 20.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0, 3.0]);
			let v2 = vec([4.0, 5.0, 6.0]);
			v1.dot(v2);
			`,
			expected: 32.0,
		},
		{
			input: `
			let v1 = vec([1.0, 0.0, 0.0]);
			let v2 = vec([0.0, 1.0, 0.0]);
			let v3 = v1.cross(v2);
			v3[0] == 0.0 && v3[1] == 0.0 && v3[2] == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			let v = vec([3.0, 4.0]);
			v.norm();
			`,
			expected: 5.0,
		},
		{
			input: `
			let v = vec([0.0, 5.0]);
			let u = v.normalize();
			u[0] == 0.0 && u[1] == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0]);
			let v2 = vec([3.0, 4.0]);
			let v3 = v1.add(v2);
			v3[0] == 4.0 && v3[1] == 6.0;
			`,
			expected: true,
		},
		{
			input: `
			let v = vec([2.0, 3.0]);
			let scaled = v.scale(3.0);
			scaled[0] == 6.0 && scaled[1] == 9.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.rows == 2 && m.cols == 2;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.get(1, 0);
			`,
			expected: 3.0,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[5.0, 6.0], [7.0, 8.0]]);
			let m3 = m1.add(m2);
			m3.get(0, 0) == 6.0 && m3.get(1, 1) == 12.0;
			`,
			expected: true,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[2.0, 0.0], [1.0, 2.0]]);
			let m3 = m1.mul(m2);
			m3.get(0, 0) == 4.0 && m3.get(0, 1) == 4.0 && m3.get(1, 0) == 10.0 && m3.get(1, 1) == 8.0;
			`,
			expected: true,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[2.0, 0.0], [1.0, 2.0]]);
			let m3 = m1 * m2;
			m3.get(0, 0) == 4.0 && m3.get(0, 1) == 4.0 && m3.get(1, 0) == 10.0 && m3.get(1, 1) == 8.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let scaled = m * 2;
			scaled.get(0, 0) == 2.0 && scaled.get(1, 1) == 8.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let scaled = 3.0 * m;
			scaled.get(0, 0) == 3.0 && scaled.get(1, 1) == 12.0;
			`,
			expected: true,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[2.0, 1.0], [1.0, 2.0]]);
			let sum = m1 + m2;
			sum.get(0, 0) == 3.0 && sum.get(1, 1) == 6.0;
			`,
			expected: true,
		},
		{
			input: `
			let v1 = vec([1.0, 0.0, 0.0]);
			let v2 = vec([0.0, 1.0, 0.0]);
			v1.cross(v2) == [0.0, 0.0, 1.0];
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let v = vec([2.0, 1.0]);
			let res = m.mul_vec(v);
			res[0] == 4.0 && res[1] == 10.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let t = m.transpose();
			t.get(0, 1) == 3.0 && t.get(1, 0) == 2.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.det();
			`,
			expected: -2.0,
		},
		{
			input: `
			let m = matrix([[4.0, 7.0], [2.0, 6.0]]);
			let inv = m.inv();
			let ident = m.mul(inv);
			floor(ident.get(0, 0) + 0.5) == 1.0 && floor(ident.get(1, 1) + 0.5) == 1.0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestScientificMathAndStats(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			floor(pi() * 100);
			`,
			expected: 314,
		},
		{
			input: `
			floor(e() * 100);
			`,
			expected: 271,
		},
		{
			input: `
			sin(0.0) == 0.0 && cos(0.0) == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			sqrt(144.0);
			`,
			expected: 12.0,
		},
		{
			input: `
			cbrt(27.0);
			`,
			expected: 3.0,
		},
		{
			input: `
			hypot(3.0, 4.0);
			`,
			expected: 5.0,
		},
		{
			input: `
			log10(1000.0);
			`,
			expected: 3.0,
		},
		{
			input: `
			log2(32.0);
			`,
			expected: 5.0,
		},
		{
			input: `
			pow(2.0, 8.0);
			`,
			expected: 256.0,
		},
		{
			input: `
			abs(-42.5);
			`,
			expected: 42.5,
		},
		{
			input: `
			floor(3.9) == 3.0 && ceil(3.1) == 4.0 && round(3.5) == 4.0 && trunc(3.7) == 3.0;
			`,
			expected: true,
		},
		{
			input: `
			deg_to_rad(180.0) == pi();
			`,
			expected: true,
		},
		{
			input: `
			rad_to_deg(pi());
			`,
			expected: 180.0,
		},
		{
			input: `
			min([5, 2, 9, 1, 7]);
			`,
			expected: 1,
		},
		{
			input: `
			max([5, 2, 9, 1, 7]);
			`,
			expected: 9,
		},
		{
			input: `
			sum([10, 20, 30, 40]);
			`,
			expected: 100,
		},
		{
			input: `
			mean([10.0, 20.0, 30.0]);
			`,
			expected: 20.0,
		},
		{
			input: `
			median([1.0, 3.0, 2.0, 5.0, 4.0]);
			`,
			expected: 3.0,
		},
		{
			input: `
			median([1.0, 2.0, 3.0, 4.0]);
			`,
			expected: 2.5,
		},
		{
			input: `
			mode([1, 2, 2, 3, 3, 3, 4]);
			`,
			expected: 3,
		},
		{
			input: `
			variance([2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0]);
			`,
			expected: 4.0,
		},
		{
			input: `
			std_dev([2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0]);
			`,
			expected: 2.0,
		},
	}
	runVmTests(t, tests)
}

func TestJSONSerialization(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let raw = "{\"name\": \"bramand\", \"version\": 1, \"active\": true}";
			let obj = json_parse(raw);
			obj["name"] == "bramand" && obj["version"] == 1 && obj["active"] == true;
			`,
			expected: true,
		},
		{
			input: `
			let raw = "[10, 20, 30]";
			let arr = json_parse(raw);
			arr[0] == 10 && arr[1] == 20 && arr[2] == 30;
			`,
			expected: true,
		},
		{
			input: `
			let data = {"title": "Language", "id": 42};
			let str = json_stringify(data);
			let back = json_parse(str);
			back["title"] == "Language" && back["id"] == 42;
			`,
			expected: true,
		},
		{
			input: `
			let data = [1, 2, 3];
			let str = json_stringify(data, 2);
			let back = json_parse(str);
			back[0] == 1 && back[1] == 2 && back[2] == 3;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestCSVAndBase64(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let csvStr = "name,age,city\nAlice,30,Tokyo\nBob,25,Paris";
			let rows = csv_parse(csvStr);
			rows[0][0] == "name" && rows[1][1] == "30" && rows[2][2] == "Paris";
			`,
			expected: true,
		},
		{
			input: `
			let csvStr = "a;b;c\n1;2;3";
			let rows = csv_parse(csvStr, ";");
			rows[0][1] == "b" && rows[1][2] == "3";
			`,
			expected: true,
		},
		{
			input: `
			let rows = [["id", "name"], ["1", "alpha"], ["2", "beta"]];
			let str = csv_stringify(rows);
			let parsed = csv_parse(str);
			parsed[1][1] == "alpha" && parsed[2][1] == "beta";
			`,
			expected: true,
		},
		{
			input: `
			let enc = base64_encode("BramandScript ✦ High Performance");
			let dec = base64_decode(enc);
			dec == "BramandScript ✦ High Performance";
			`,
			expected: true,
		},
		{
			input: `
			let enc = base64_url_encode("hello+world/test?foo=bar");
			let dec = base64_url_decode(enc);
			dec == "hello+world/test?foo=bar";
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestCryptoAndRandom(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			sha256("hello");
			`,
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			input: `
			md5("hello");
			`,
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			input: `
			sha512("hello");
			`,
			expected: "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043",
		},
		{
			input: `
			hmac_sha256("secret", "message");
			`,
			expected: "8b5f48702995c1598c573db1e21866a9b825d4a794d169d7060a03605796360b",
		},
		{
			input: `
			let rnd = random_bytes(16);
			len(rnd);
			`,
			expected: 32,
		},
		{
			input: `
			let n = random_int(10, 20);
			n >= 10 && n <= 20;
			`,
			expected: true,
		},
		{
			input: `
			let u = uuid();
			len(u) == 36;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestVectorSimilarityAndEmbeddings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let v1 = vec([1.0, 0.0]);
			let v2 = vec([0.0, 1.0]);
			cosine_similarity(v1, v2);
			`,
			expected: 0.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0, 3.0]);
			let v2 = vec([1.0, 2.0, 3.0]);
			floor(cosine_similarity(v1, v2) * 100);
			`,
			expected: 100,
		},
		{
			input: `
			let v1 = vec([1.0, 1.0]);
			let v2 = vec([1.0, 1.0]);
			cosine_distance(v1, v2);
			`,
			expected: 0.0,
		},
		{
			input: `
			let v1 = vec([0.0, 0.0]);
			let v2 = vec([3.0, 4.0]);
			euclidean_distance(v1, v2);
			`,
			expected: 5.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0]);
			let v2 = vec([4.0, 6.0]);
			manhattan_distance(v1, v2);
			`,
			expected: 7.0,
		},
		{
			input: `
			let query = vec([1.0, 0.0]);
			let doc1 = {"id": "doc1", "embedding": vec([0.0, 1.0])};
			let doc2 = {"id": "doc2", "embedding": vec([0.9, 0.1])};
			let doc3 = {"id": "doc3", "embedding": vec([-1.0, 0.0])};

			let results = vector_search(query, [doc1, doc2, doc3], 1, "cosine");
			let topMatch = results[0]["item"];
			topMatch["id"];
			`,
			expected: "doc2",
		},
	}
	runVmTests(t, tests)
}

func TestAIIntegrationModule(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = http_server();
			srv.post("/api/mock_stream", fn(req, res) {
				res.set_header("Content-Type", "text/event-stream");
				res.send("data: {\"response\": \"Hello \"}\n\ndata: {\"response\": \"from \"}\n\ndata: {\"response\": \"AI!\"}\n\ndata: [DONE]\n\n");
			});
			srv.listen("127.0.0.1:19085");

			let collected = [];
			let full = ai_stream({"url": "http://127.0.0.1:19085/api/mock_stream", "body": "{}"}, fn(token) {
				collected = push(collected, token);
			});

			srv.close();
			full == "Hello from AI!" && len(collected) == 3 && collected[0] == "Hello ";
			`,
			expected: true,
		},
		{
			input: `
			let srv = http_server();
			srv.post("/api/mock_structured", fn(req, res) {
				res.json({"summary": "High Performance Language", "stars": 5});
			});
			srv.listen("127.0.0.1:19086");

			let schema = {"summary": "string", "stars": 0};
			let structured = ai_structured({"model": "test-model", "prompt": "analyze", "endpoint": "http://127.0.0.1:19086/api/mock_structured"}, schema);

			srv.close();
			structured["summary"] == "High Performance Language" && structured["stars"] == 5;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestAssertionsAndTestingFramework(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			assert(true);
			assert_eq(10 + 20, 30);
			assert_eq([1, 2, 3], [1, 2, 3]);
			assert_eq({"a": 1, "b": 2}, {"a": 1, "b": 2});
			assert_ne(10, 20);
			assert_ne([1, 2], [1, 3]);
			true;
			`,
			expected: true,
		},
		{
			input: `
			let passed = 0;
			describe("Math Suite", fn() {
				test("addition works", fn() {
					assert_eq(5 + 5, 10);
					passed = passed + 1;
				});
				test("subtraction works", fn() {
					assert_eq(10 - 4, 6);
					passed = passed + 1;
				});
			});
			passed == 2;
			`,
			expected: true,
		},
		{
			input: `
			let failedHandled = false;
			test("deliberate failure caught", fn() {
				assert_eq(1, 2, "1 should equal 2");
			});
			true;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestTimeAndBenchmarking(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let t1 = time_ms();
			let now = time_now();
			t1 > 0 && now > 0;
			`,
			expected: true,
		},
		{
			input: `
			let b = benchmark("simple_add", fn() {
				let x = 1 + 1;
			}, 100);
			b["name"] == "simple_add" && b["iterations"] == 100 && b["total_ms"] >= 0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}
