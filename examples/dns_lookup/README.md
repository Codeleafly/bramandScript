# Pure BramandScript DNS Lookup Utility (`examples/dns_lookup/`)

A production-grade, 100% pure BramandScript RFC 1035 UDP DNS resolution tool and packet codec built with `std/dns` and `std/net`.

## Architecture & Features

- **Pure BMD RFC 1035 Packet Codec**:
  - Encodes 12-byte DNS headers, QNAME labels, and question sections into binary datagrams.
  - Decodes answers, CNAME aliases, and handles QNAME pointer compression (`0xC0`).
- **UDP Datagram Communications**: Transmits and receives DNS packets over raw UDP sockets (`net.create_udp_socket`).
- **In-Memory TTL Caching**: High-performance in-memory cache with automatic TTL tracking and eviction to prevent duplicate queries.
- **Convenience API**: `dns.resolve(domain, type, server)` and `dns.lookup(domain, server)`.
- **Automated Verification**: Spawns an internal loopback authoritative DNS server responding with mock A records, tests fresh UDP resolution, cache hit resolution, and clean shutdown.

## Running the Application

```bash
# Run automated DNS lookup demo
./bramand run --allow-all examples/dns_lookup/main.bmd
```
