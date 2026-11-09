package compiler

type BuiltinDef struct {
	Name string
}

var Builtins = []BuiltinDef{
	{Name: "len"},
	{Name: "print"},
	{Name: "println"},
	{Name: "print_raw"},
	{Name: "type"},
	{Name: "push"},
	{Name: "append"},
	{Name: "now"},
	{Name: "input"},
	{Name: "int"},
	{Name: "float"},
	{Name: "str"},
	{Name: "bool"},
	{Name: "keys"},
	{Name: "values"},
	{Name: "slice"},
	{Name: "pop"},
	{Name: "contains"},
	{Name: "delete"},
	{Name: "exit"},

	// Day 4 Concurrency, Channels, Mutex
	{Name: "chan"},
	{Name: "close"},
	{Name: "mutex"},
	{Name: "wait_group"},
	{Name: "sleep"},

	// Day 4 File I/O
	{Name: "read_file"},
	{Name: "write_file"},
	{Name: "append_file"},
	{Name: "delete_file"},
	{Name: "file_exists"},
	{Name: "list_dir"},
	{Name: "chmod"},
	{Name: "is_dir"},

	// Day 4 Process Execution & Environment
	{Name: "env"},
	{Name: "set_env"},
	{Name: "exec"},

	// Day 4 Standard I/O & Formatting
	{Name: "eprint"},
	{Name: "eprintln"},
	{Name: "color"},
	{Name: "styled"},
	{Name: "read_line"},

	// Day 5 BigInt & BigFloat
	{Name: "big_int"},
	{Name: "bigint"},
	{Name: "big_float"},
	{Name: "bigfloat"},

	// Day 5 Vectors & Matrices
	{Name: "vec"},
	{Name: "vector"},
	{Name: "matrix"},

	// Day 5 Trigonometric, Logarithmic & Math
	{Name: "pi"},
	{Name: "e"},
	{Name: "sin"},
	{Name: "cos"},
	{Name: "tan"},
	{Name: "asin"},
	{Name: "acos"},
	{Name: "atan"},
	{Name: "atan2"},
	{Name: "sinh"},
	{Name: "cosh"},
	{Name: "tanh"},
	{Name: "sqrt"},
	{Name: "cbrt"},
	{Name: "hypot"},
	{Name: "log"},
	{Name: "log10"},
	{Name: "log2"},
	{Name: "exp"},
	{Name: "pow"},
	{Name: "abs"},
	{Name: "floor"},
	{Name: "ceil"},
	{Name: "round"},
	{Name: "trunc"},
	{Name: "deg_to_rad"},
	{Name: "rad_to_deg"},
	{Name: "min"},
	{Name: "max"},

	// Day 5 Statistical standard library
	{Name: "sum"},
	{Name: "mean"},
	{Name: "math_mean"},
	{Name: "median"},
	{Name: "mode"},
	{Name: "variance"},
	{Name: "std_dev"},

	// Day 5 JSON Parser & Serializer
	{Name: "json_parse"},
	{Name: "json_stringify"},

	// Day 5 CSV & Base64
	{Name: "csv_parse"},
	{Name: "csv_stringify"},
	{Name: "base64_encode"},
	{Name: "base64_decode"},
	{Name: "base64_url_encode"},
	{Name: "base64_url_decode"},

	// Day 5 Cryptography & CSPRNG
	{Name: "sha256"},
	{Name: "crypto_sha256"},
	{Name: "sha512"},
	{Name: "md5"},
	{Name: "hmac_sha256"},
	{Name: "crypto_hmac"},
	{Name: "random_bytes"},
	{Name: "random_int"},
	{Name: "uuid"},

	// Day 6 TCP & UDP Sockets
	{Name: "tcp_listen"},
	{Name: "tcp_connect"},
	{Name: "udp_listen"},
	{Name: "udp_connect"},

	// Day 6 HTTP/HTTPS Client
	{Name: "http_get"},
	{Name: "http_post"},
	{Name: "http_put"},
	{Name: "http_delete"},
	{Name: "http_request"},

	// Day 6 HTTP Web Server
	{Name: "http_server"},

	// Day 6 WebSockets
	{Name: "ws_connect"},
	{Name: "ws_server"},

	// Day 6 AI Integrations (LLM streaming, prompt, structured outputs)
	{Name: "ai_prompt"},
	{Name: "ai_stream"},
	{Name: "ai_structured"},

	// Day 6 Vector Similarity & AI Embeddings
	{Name: "cosine_similarity"},
	{Name: "cosine_distance"},
	{Name: "euclidean_distance"},
	{Name: "manhattan_distance"},
	{Name: "vector_search"},

	// Day 7 Testing & Developer Tooling
	{Name: "assert"},
	{Name: "assert_eq"},
	{Name: "assert_ne"},
	{Name: "panic"},
	{Name: "throw"},
	{Name: "test"},
	{Name: "describe"},
	{Name: "time_now"},
	{Name: "time_ms"},
	{Name: "benchmark"},

	// Command-Line Arguments & Program State
	{Name: "args"},
	{Name: "argv"},

	// String & Collection Utilities
	{Name: "substring"},
	{Name: "sort"},
	{Name: "reverse"},
	{Name: "split"},
	{Name: "join"},
	{Name: "trim"},
	{Name: "to_lower"},
	{Name: "to_upper"},
	{Name: "replace"},
	{Name: "time_iso"},
	{Name: "date_format"},

	// Terminal, Keyboard & Raw Input
	{Name: "term_raw"},
	{Name: "term_size"},
	{Name: "read_key"},
	{Name: "poll_key"},

	// Day 8 & Day 9 Features
	{Name: "bytes"},
	{Name: "read_u8"},
	{Name: "write_u8"},
	{Name: "read_u16"},
	{Name: "write_u16"},
	{Name: "read_u32"},
	{Name: "write_u32"},
	{Name: "ord"},
	{Name: "chr"},
	{Name: "eval"},
	{Name: "implements"},

	// Iteration Helpers
	{Name: "__iter_len"},
	{Name: "__iter_key"},
	{Name: "__iter_val"},
	{Name: "__iter_item"},

	// Testing Assertions
	{Name: "assert_true"},
	{Name: "assert_false"},

	// Day 15 Fast Binary Exponentiation & Modular Arithmetic
	{Name: "pow_mod"},

	// Day 20 Compression & Archiving
	{Name: "gzip_compress"},
	{Name: "gzip_decompress"},
	{Name: "zlib_compress"},
	{Name: "zlib_decompress"},
	{Name: "tar_pack"},
	{Name: "tar_unpack"},

	// Day 20 Binary Struct Packing & MsgPack
	{Name: "struct_pack"},
	{Name: "struct_unpack"},
	{Name: "struct_calcsize"},
	{Name: "msgpack_encode"},
	{Name: "msgpack_decode"},

	// Day 20 Modern Cryptography
	{Name: "aes_gcm_encrypt"},
	{Name: "aes_gcm_decrypt"},
	{Name: "ed25519_keypair"},
	{Name: "ed25519_sign"},
	{Name: "ed25519_verify"},
	{Name: "jwt_sign"},
	{Name: "jwt_verify"},

	// Developer Tooling & Reflection
	{Name: "create_session"},
	{Name: "session_vars"},
	{Name: "disasm"},
	{Name: "ensure_newline"},
	{Name: "error"},
	{Name: "yield"},
}
