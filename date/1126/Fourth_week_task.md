# BramandScript — Month 1126: Fourth Week Task (Days 22 to 28)

## Day 22

- [x] 64-Bit Opcode Operands for Large Programs: Rewrite OpConstant, OpJump, OpJumpNotTruthy, OpJumpTruthy, OpSetGlobal, OpGetGlobal, OpClosure, and OpPushHandler with 8-byte uint64 operands to support unbounded 64-bit bytecode programs (Includes: code/code.go, compiler/compiler.go, vm/vm.go, compiler/optimizer.go, unit tests)
- [x] Permission Guard System with CLI Flags: Implement granular CLI permission guards (--allow-all, --allow-run, --allow-write, --allow-net) protecting exec, write_file, append_file, delete_file, and tcp_listen with PermissionDenied errors. (Includes: vm/permissions.go, vm/builtins_io.go, vm/builtins_net.go, main.go, tests)
- [x] Dynamic Stack Allocation and Configurable MaxFrames: Convert static MaxFrames=1024 into configurable and dynamically expandable call frames slice with VM options and CLI flag. (Includes: vm/vm.go, vm/frame.go, main.go, tests)
- [x] Exception Handling Grammar and Runtime (try-catch): Implement AST, Lexer, Parser, Compiler, and VM execution for try { } catch err { } exception handling statements with error capture. (Includes: token/token.go, lexer/lexer.go, ast/ast.go, parser/statements.go, compiler/compile_statements.go, vm/vm.go, tests)
## Day 23

## Day 24

## Day 25

## Day 26

## Day 27

## Day 28

