package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"testing"
	"time"
)

func runActorSource(t *testing.T, input string) object.Object {
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

func TestActorAskAndSend(t *testing.T) {
	input := `
	let counter = actor(fn(msg, count) {
		if msg == "inc" {
			return count + 1;
		}
		if msg == "get" {
			return count;
		}
		return count;
	}, 0);

	counter.send("inc");
	counter.send("inc");
	sleep(50);
	let val = counter.ask("get", 1000);
	counter.stop();

	val
	`
	res := runActorSource(t, input)
	intVal, ok := res.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s (%+v)", res.Type(), res)
	}
	if intVal.Value != 2 {
		t.Errorf("expected counter value 2, got %d", intVal.Value)
	}
}

func TestActorKeyword(t *testing.T) {
	input := `
	let echo = actor fn(msg) {
		return "echo: " + str(msg);
	};

	let reply = echo.ask("hello", 1000);
	echo.stop();

	reply
	`
	res := runActorSource(t, input)
	strVal, ok := res.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %s (%+v)", res.Type(), res)
	}
	if strVal.Value != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %s", strVal.Value)
	}
}

func TestActorRegistry(t *testing.T) {
	input := `
	let worker = actor fn(msg) { return "ok"; };
	actor_register("worker_1", worker);

	let found = actor_whereis("worker_1");
	let before_unreg = (found != null && found.name() == "worker_1");

	actor_unregister("worker_1");
	let after_unreg = (actor_whereis("worker_1") == null);
	worker.stop();

	[before_unreg, after_unreg]
	`
	res := runActorSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0] != True {
		t.Errorf("expected before_unreg true, got %v", arr.Elements[0])
	}
	if arr.Elements[1] != True {
		t.Errorf("expected after_unreg true, got %v", arr.Elements[1])
	}
}

func TestActorSupervisionTree(t *testing.T) {
	input := `
	import "std/actor" as act_mod;

	let sup = act_mod.supervisor({
		"strategy": "one_for_one",
		"max_restarts": 5,
		"time_window": 3000,
		"children": [
			{
				"id": "c1",
				"start": fn() {
					return actor fn(msg) {
						if msg == "crash" {
							panic("deliberate child panic");
						}
						return "c1_alive";
					};
				}
			}
		]
	});

	let child1 = sup.get_child("c1");
	let r1 = child1.ask("ping", 1000);

	// Send crashing message
	child1.send("crash");
	sleep(60);

	// Let supervisor recover child
	sup.check_and_recover();
	sleep(40);

	let child1_new = sup.get_child("c1");
	let r2 = child1_new.ask("ping", 1000);
	sup.stop();

	[r1, r2]
	`
	res := runActorSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "c1_alive" {
		t.Errorf("expected c1_alive before crash, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "c1_alive" {
		t.Errorf("expected c1_alive after recovery, got %v", arr.Elements[1])
	}
	_ = time.Second
}
