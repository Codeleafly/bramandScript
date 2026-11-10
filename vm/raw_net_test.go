package vm

import (
	"strings"
	"testing"
	"time"

	"bramandscript/object"
)

func TestRawNetTcpPrimitives(t *testing.T) {
	listenFn := GetBuiltin("net_listen_tcp").Fn
	acceptFn := GetBuiltin("tcp_accept").Fn
	dialFn := GetBuiltin("net_dial_tcp").Fn
	writeFn := GetBuiltin("tcp_write").Fn
	readLineFn := GetBuiltin("tcp_read_line").Fn
	closeFn := GetBuiltin("tcp_close").Fn
	remoteAddrFn := GetBuiltin("tcp_remote_addr").Fn
	localAddrFn := GetBuiltin("tcp_local_addr").Fn

	srvObj := listenFn(&object.String{Value: "127.0.0.1:28201"})
	if srvObj.Type() == object.ERROR_OBJ {
		t.Fatalf("listen failed: %s", srvObj.Inspect())
	}
	defer closeFn(srvObj)

	go func() {
		connObj := acceptFn(srvObj)
		if connObj.Type() == object.ERROR_OBJ {
			return
		}
		defer closeFn(connObj)

		lineObj := readLineFn(connObj)
		if s, ok := lineObj.(*object.String); ok {
			writeFn(connObj, &object.String{Value: "PONG:" + s.Value + "\n"})
		}
	}()

	time.Sleep(20 * time.Millisecond)

	clientObj := dialFn(&object.String{Value: "127.0.0.1:28201"})
	if clientObj.Type() == object.ERROR_OBJ {
		t.Fatalf("dial failed: %s", clientObj.Inspect())
	}
	defer closeFn(clientObj)

	rAddr := remoteAddrFn(clientObj)
	if !strings.Contains(rAddr.Inspect(), "127.0.0.1") {
		t.Errorf("expected remote addr to contain 127.0.0.1, got %s", rAddr.Inspect())
	}

	lAddr := localAddrFn(clientObj)
	if !strings.Contains(lAddr.Inspect(), "127.0.0.1") {
		t.Errorf("expected local addr to contain 127.0.0.1, got %s", lAddr.Inspect())
	}

	writeFn(clientObj, &object.String{Value: "PING\n"})
	resp := readLineFn(clientObj)
	if s, ok := resp.(*object.String); !ok || s.Value != "PONG:PING" {
		t.Errorf("expected PONG:PING, got %s", resp.Inspect())
	}
}

func TestRawNetUdpPrimitives(t *testing.T) {
	listenUdpFn := GetBuiltin("net_listen_udp").Fn
	sendToFn := GetBuiltin("udp_send_to").Fn
	recvFromFn := GetBuiltin("udp_recv_from").Fn
	closeFn := GetBuiltin("udp_close").Fn

	u1 := listenUdpFn(&object.String{Value: "127.0.0.1:28202"})
	if u1.Type() == object.ERROR_OBJ {
		t.Fatalf("udp listen failed: %s", u1.Inspect())
	}
	defer closeFn(u1)

	u2 := listenUdpFn(&object.String{Value: "127.0.0.1:28203"})
	if u2.Type() == object.ERROR_OBJ {
		t.Fatalf("udp listen failed: %s", u2.Inspect())
	}
	defer closeFn(u2)

	sendRes := sendToFn(u2, &object.String{Value: "HELLO_UDP"}, &object.String{Value: "127.0.0.1:28202"})
	if sendRes.Type() == object.ERROR_OBJ {
		t.Fatalf("udp send failed: %s", sendRes.Inspect())
	}

	recvRes := recvFromFn(u1, &object.Integer{Value: 1024})
	if recvRes.Type() == object.ERROR_OBJ {
		t.Fatalf("udp recv failed: %s", recvRes.Inspect())
	}

	h, ok := recvRes.(*object.Hash)
	if !ok {
		t.Fatalf("expected hash, got %s", recvRes.Type())
	}

	dataVal := h.Pairs[(&object.String{Value: "data"}).HashKey()].Value.Inspect()
	if dataVal != "HELLO_UDP" {
		t.Errorf("expected HELLO_UDP, got %s", dataVal)
	}
}

func TestRawNetTlsPrimitives(t *testing.T) {
	listenTlsFn := GetBuiltin("tls_listen").Fn
	dialTlsFn := GetBuiltin("tls_dial").Fn
	acceptFn := GetBuiltin("tcp_accept").Fn
	writeFn := GetBuiltin("tcp_write").Fn
	readLineFn := GetBuiltin("tcp_read_line").Fn
	closeFn := GetBuiltin("tcp_close").Fn

	srvObj := listenTlsFn(&object.String{Value: "127.0.0.1:28204"})
	if srvObj.Type() == object.ERROR_OBJ {
		t.Fatalf("tls_listen failed: %s", srvObj.Inspect())
	}
	defer closeFn(srvObj)

	go func() {
		connObj := acceptFn(srvObj)
		if connObj.Type() == object.ERROR_OBJ {
			return
		}
		defer closeFn(connObj)

		msg := readLineFn(connObj)
		if s, ok := msg.(*object.String); ok {
			writeFn(connObj, &object.String{Value: "SECURE:" + s.Value + "\n"})
		}
	}()

	time.Sleep(30 * time.Millisecond)

	// Test with insecure_skip_verify = true for local self-signed cert
	opts := &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			(&object.String{Value: "insecure_skip_verify"}).HashKey(): {
				Key:   &object.String{Value: "insecure_skip_verify"},
				Value: &object.Boolean{Value: true},
			},
		},
	}
	clientObj := dialTlsFn(&object.String{Value: "127.0.0.1:28204"}, &object.String{Value: "localhost"}, &object.Integer{Value: 5000}, opts)
	if clientObj.Type() == object.ERROR_OBJ {
		t.Fatalf("tls_dial failed: %s", clientObj.Inspect())
	}
	defer closeFn(clientObj)

	writeFn(clientObj, &object.String{Value: "SECRET\n"})
	resp := readLineFn(clientObj)
	if s, ok := resp.(*object.String); !ok || s.Value != "SECURE:SECRET" {
		t.Errorf("expected SECURE:SECRET, got %s", resp.Inspect())
	}
}
