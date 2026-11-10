# BramandScript — Month 1126: Sixth Week Task (Days 36 to 42)

## Day 36

- [x] Pure BramandScript HTTP Wire Parser (std/http/parser): Implement HTTP/1.1 request and response wire parser and serializer with RFC 7230 compliance (Includes: parse_request, parse_response, format_request, format_response)
- [x] Pure BramandScript HTTP Server (std/http/server): Implement self-hosted HTTP/1.1 server over raw TCP sockets with Request and Response abstractions (Includes: http.create_server, req.query, req.json, res.send, res.json, res.status)
- [x] Pure BramandScript HTTP Client (std/http/client): Implement self-hosted HTTP/1.1 client over raw TCP sockets with GET, POST, and generic request (Includes: http.get, http.post, http.request, connection close)
- [x] Master Standard Library Integration and Router Interop (std/http): Integrate HTTP module into standard library with router and middleware re-exports (Includes: stdlib/bmd_std/http/index.bmd, router interop, middleware pipeline)
- [x] Automated Pure BMD HTTP Test Suite (tests/36_pure_bmd_http_test.bmd): Comprehensive integration test suite validating pure BramandScript HTTP client and server (Includes: GET/POST requests, JSON payloads, 404 handling, header case-insensitivity)
## Day 37

- [x] Pure BramandScript RFC 6455 Handshake (std/ws/handshake): Implement WebSocket handshake key derivation and HTTP 101 upgrade protocol formatting — Implement RFC 6455 WebSocket handshake accept key derivation with SHA-1 and Base64, server HTTP 101 response, and client upgrade request (Includes: compute_accept_key, build_server_handshake, build_client_handshake, parse_handshake)
- [x] Pure BramandScript RFC 6455 Frame Codec (std/ws/frame): Implement binary frame encoder and decoder with FIN bit, opcodes, masking, and variable payload lengths — Implement RFC 6455 frame serializer and parser supporting 7-bit, 16-bit, and 64-bit payload lengths with client XOR masking (Includes: encode_frame, decode_frame, text, binary, ping, pong, close)
- [x] Pure BramandScript WebSocket Connection (std/ws/connection): Implement WebSocket connection abstraction wrapping raw TCP socket with event emitters and control frame handlers — Implement WebSocket connection abstraction over raw duplex TCP socket with send, ping, pong, close methods and asynchronous reader loop (Includes: WebSocket class, send, ping, pong, close, on message, reader loop)
- [x] Pure BramandScript WebSocket Server & Client (std/ws/server & client): Implement self-hosted WebSocket server and client with connection lifecycle, broadcast, and URL dialer — Implement pure BramandScript WebSocket server and client with handshake negotiation, connection lifecycle, and multi-client broadcast (Includes: ws.create_server, ws.Server, ws.connect, ws.Client, broadcast)
- [x] Master Standard Library Integration and Test Suite (std/ws & tests/37_pure_bmd_ws_test.bmd): Integrate WebSocket module into stdlib with RoomHub interop and comprehensive test suite — Integrate WebSocket module into standard library with RoomHub re-exports and comprehensive integration test suite for handshake, framing, and duplex communication (Includes: stdlib/bmd_std/ws/index.bmd, RFC 6455 test vectors, duplex client-server echo)
## Day 38

- [x] Pure BramandScript RFC 1035 DNS Packet Codec (std/dns/packet): Implement RFC 1035 DNS packet encoder and decoder with 12-byte header, QName label compression, A and CNAME record parsing (Includes: encode_query, decode_response, label compression pointers 0xC0, A and CNAME parsing, TTL decoding)
- [x] Pure BramandScript DNS Resolver and In-Memory Cache (std/dns/resolver): Implement UDP DNS resolver over raw sockets with in-memory TTL caching, query timeouts, and convenience lookup helpers (Includes: dns.resolve, dns.lookup, net.create_udp_socket, query datagram transmission, TTL cache expiration)
- [x] Pure BramandScript HTTPS Client over TLS Sockets (std/https/client): Implement self-hosted HTTPS client over raw TLS sockets with SNI verification, URL parsing, and HTTP parser integration (Includes: https.get, https.post, https.request, URL parsing with port 443 default, TLS dial with SNI, structured response)
- [x] Pure BramandScript HTTPS Server and Standard Library Exports (std/https/server & index): Implement self-hosted HTTPS server wrapping TLS listeners and dispatching to HTTP/1.1 request pipeline with stdlib exports (Includes: https.create_server, tls_listen, TLS listener dispatch, stdlib/bmd_std/dns/index.bmd, stdlib/bmd_std/https/index.bmd)
- [x] Automated Pure BMD DNS & HTTPS Test Suite (tests/38_pure_bmd_dns_https_test.bmd): Comprehensive integration test suite validating RFC 1035 DNS codec, mock UDP DNS server, resolver cache, and HTTPS requests (Includes: RFC 1035 QNAME encoding/decoding, pointer compression, mock UDP DNS server, cache eviction, HTTPS TLS request)
## Day 39

## Day 40

## Day 41

## Day 42

