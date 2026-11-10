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

	// Day 29 Production-Grade Date, Time & Calendar Engine
	{Name: "datetime_now"},
	{Name: "datetime_now_iso"},
	{Name: "datetime_parse"},
	{Name: "datetime_format"},
	{Name: "datetime_parts"},
	{Name: "datetime_is_leap_year"},
	{Name: "datetime_days_in_month"},
	{Name: "datetime_add"},
	{Name: "datetime_diff"},
	{Name: "datetime_timezone_offset"},

	// Day 29 Enterprise File System & Path Manipulation Engine (std/fs & std/path)
	{Name: "path_join"},
	{Name: "path_clean"},
	{Name: "path_base"},
	{Name: "path_dir"},
	{Name: "path_ext"},
	{Name: "path_abs"},
	{Name: "path_is_abs"},
	{Name: "path_split"},
	{Name: "fs_walk"},
	{Name: "fs_glob"},
	{Name: "fs_temp_file"},
	{Name: "fs_tree"},
	{Name: "fs_mkdir"},
	{Name: "fs_stat"},

	// Day 29 Universal Data Encoding & Serialization Suite (std/encoding)
	{Name: "hex_encode"},
	{Name: "hex_decode"},
	{Name: "url_encode"},
	{Name: "url_decode"},
	{Name: "url_parse_query"},
	{Name: "url_build_query"},
	{Name: "csv_parse_records"},
	{Name: "csv_format_records"},

	// Day 29 & Day 30 Actor Model & Process Registry
	{Name: "actor"},
	{Name: "actor_spawn"},
	{Name: "actor_register"},
	{Name: "actor_whereis"},
	{Name: "actor_unregister"},

	// Day 30 Monadic Safety & Algebraic Data Types (Result & Option)
	{Name: "Ok"},
	{Name: "Err"},
	{Name: "Some"},
	{Name: "None"},
	{Name: "is_ok"},
	{Name: "is_err"},
	{Name: "is_some"},
	{Name: "is_none"},

	// Day 31 Cooperative Multitasking Scheduler & Structured Task Groups
	{Name: "task_group"},

	// Day 32 Native Reactive Event Streams with Backpressure
	{Name: "stream"},

	// Day 33 Relational SQL Engine Execution
	{Name: "sql_query"},

	// Day 34 Local AI Vector Similarity & Metric Engine
	{Name: "dot_product"},
	{Name: "vector_normalize"},

	// Day 35 Benchmarking and Fuzz Testing Framework
	{Name: "bench"},
	{Name: "fuzz"},

	// Raw Networking Engine (Agent 5)
	{Name: "net_listen_tcp"},
	{Name: "tcp_accept"},
	{Name: "net_dial_tcp"},
	{Name: "tcp_dial"},
	{Name: "tcp_read"},
	{Name: "tcp_read_line"},
	{Name: "tcp_write"},
	{Name: "tcp_close"},
	{Name: "tcp_set_timeout"},
	{Name: "tcp_remote_addr"},
	{Name: "tcp_local_addr"},
	{Name: "net_listen_udp"},
	{Name: "udp_send_to"},
	{Name: "udp_recv_from"},
	{Name: "udp_close"},
	{Name: "udp_set_timeout"},
	{Name: "tls_dial"},
	{Name: "tls_listen"},
	{Name: "read_u64"},
	{Name: "write_u64"},
	{Name: "sha1"},
	{Name: "crypto_sha1"},

	// Day 39 OS, Process & IPC Streams (Agent 9)
	{Name: "proc_spawn"},
	{Name: "proc_stdin_write"},
	{Name: "proc_stdin_close"},
	{Name: "proc_stdout_read"},
	{Name: "proc_stdout_read_all"},
	{Name: "proc_stderr_read"},
	{Name: "proc_stderr_read_all"},
	{Name: "proc_wait"},
	{Name: "proc_kill"},
	{Name: "proc_pid"},
	{Name: "os_platform"},
	{Name: "os_arch"},
	{Name: "os_cpus"},
	{Name: "os_memory_info"},
	{Name: "os_env_all"},
	{Name: "os_get_env"},
	{Name: "os_set_env"},
	{Name: "os_cwd"},
	{Name: "os_pid"},

	// Day 40 Byte Buffers & Binary I/O (Agent 10)
	{Name: "buf_alloc"},
	{Name: "buf_from_string"},
	{Name: "buf_from_array"},
	{Name: "buf_to_string"},
	{Name: "buf_copy"},
	{Name: "buf_fill"},
	{Name: "buf_read_u8"},
	{Name: "buf_write_u8"},
	{Name: "buf_read_u16_be"},
	{Name: "buf_read_u16_le"},
	{Name: "buf_write_u16_be"},
	{Name: "buf_write_u16_le"},
	{Name: "buf_read_u32_be"},
	{Name: "buf_read_u32_le"},
	{Name: "buf_write_u32_be"},
	{Name: "buf_write_u32_le"},
	{Name: "buf_read_i32_be"},
	{Name: "buf_read_i32_le"},
	{Name: "buf_write_i32_be"},
	{Name: "buf_write_i32_le"},
	{Name: "buf_read_f32_be"},
	{Name: "buf_read_f32_le"},
	{Name: "buf_write_f32_be"},
	{Name: "buf_write_f32_le"},
	{Name: "buf_read_f64_be"},
	{Name: "buf_read_f64_le"},
	{Name: "buf_write_f64_be"},
	{Name: "buf_write_f64_le"},
	{Name: "buf_compare"},
	{Name: "buf_concat"},
	{Name: "buf_index_of"},
}
