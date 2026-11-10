package vm

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"bramandscript/object"
)

func parseNetAddr(arg object.Object) string {
	if sObj, ok := arg.(*object.String); ok {
		return sObj.Value
	}
	if iObj, ok := arg.(*object.Integer); ok {
		return fmt.Sprintf("127.0.0.1:%d", iObj.Value)
	}
	return arg.Inspect()
}

func extractNetConn(arg object.Object) (object.NetConn, error) {
	if c, ok := arg.(object.NetConn); ok {
		return c, nil
	}
	if h, ok := arg.(*object.Hash); ok {
		rawKey := (&object.String{Value: "raw"}).HashKey()
		if pair, found := h.Pairs[rawKey]; found {
			if c, ok := pair.Value.(object.NetConn); ok {
				return c, nil
			}
		}
	}
	return nil, fmt.Errorf("argument must be TcpConn or TlsConn, got %s", arg.Type())
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"BramandScript"},
			CommonName:   "localhost",
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func dialTLS(addrStr, hostName string, timeoutMs int64, insecure bool) (*object.TlsConn, error) {
	timeout := time.Duration(timeoutMs) * time.Millisecond

	host := addrStr
	if strings.Contains(addrStr, ":") {
		h, _, err := net.SplitHostPort(addrStr)
		if err == nil {
			host = h
		}
	} else {
		addrStr = net.JoinHostPort(addrStr, "443")
	}

	if hostName == "" {
		hostName = host
	}

	tlsConfig := &tls.Config{
		ServerName:         hostName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	}

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addrStr, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &object.TlsConn{
		Conn:     conn,
		Reader:   bufio.NewReader(conn),
		Timeout:  timeout,
		HostName: hostName,
	}, nil
}

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"net_listen_tcp": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("net_listen_tcp requires 1 argument (addr). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpServer{Listener: l}
			},
		},
		"tcp_listen": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_listen requires 1 argument (addr). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpServer{Listener: l}
			},
		},
		"tcp_accept": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_accept requires 1 argument (server). got=%d", len(args))}
				}
				srv, ok := args[0].(*object.TcpServer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to tcp_accept must be TcpServer, got %s", args[0].Type())}
				}
				conn, err := srv.Accept()
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return conn
			},
		},
		"net_dial_tcp": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("net_dial_tcp requires 1 or 2 arguments (addr, [timeout_ms]). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				timeout := 5 * time.Second
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						timeout = time.Duration(iObj.Value) * time.Millisecond
					}
				}
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpConn{
					Conn:   conn,
					Reader: bufio.NewReader(conn),
				}
			},
		},
		"tcp_dial": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("tcp_dial requires 1 or 2 arguments (addr, [timeout_ms]). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				timeout := 5 * time.Second
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						timeout = time.Duration(iObj.Value) * time.Millisecond
					}
				}
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpConn{
					Conn:   conn,
					Reader: bufio.NewReader(conn),
				}
			},
		},
		"tcp_connect": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("tcp_connect requires 1 or 2 arguments (addr, [timeout_ms]). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				timeout := 5 * time.Second
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						timeout = time.Duration(iObj.Value) * time.Millisecond
					}
				}
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpConn{
					Conn:   conn,
					Reader: bufio.NewReader(conn),
				}
			},
		},
		"tcp_read": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("tcp_read requires 1 or 2 arguments (conn, [max_bytes]). got=%d", len(args))}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				maxBytes := 4096
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						maxBytes = int(iObj.Value)
					}
				}
				data, readErr := conn.Read(maxBytes)
				if readErr != nil {
					if readErr == io.EOF {
						return Null
					}
					return &object.Error{Message: readErr.Error()}
				}
				return &object.String{Value: string(data)}
			},
		},
		"tcp_read_line": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_read_line requires 1 argument (conn). got=%d", len(args))}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				line, readErr := conn.ReadLine()
				if readErr != nil {
					if readErr == io.EOF && len(line) == 0 {
						return Null
					}
					return &object.Error{Message: readErr.Error()}
				}
				return &object.String{Value: line}
			},
		},
		"tcp_write": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("tcp_write requires 2 arguments (conn, data). got=%d", len(args))}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				var data []byte
				if strObj, ok := args[1].(*object.String); ok {
					data = []byte(strObj.Value)
				} else if bytesObj, ok := args[1].(*object.Bytes); ok {
					data = bytesObj.Value
				} else {
					data = []byte(args[1].Inspect())
				}
				n, writeErr := conn.Write(data)
				if writeErr != nil {
					return &object.Error{Message: writeErr.Error()}
				}
				return &object.Integer{Value: int64(n)}
			},
		},
		"tcp_close": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_close requires 1 argument (conn/server). got=%d", len(args))}
				}
				if srv, ok := args[0].(*object.TcpServer); ok {
					if err := srv.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				if err := conn.Close(); err != nil {
					return &object.Error{Message: err.Error()}
				}
				return Null
			},
		},
		"tcp_set_timeout": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("tcp_set_timeout requires 2 arguments (conn, ms). got=%d", len(args))}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				if iObj, ok := args[1].(*object.Integer); ok {
					conn.SetTimeout(iObj.Value)
					return Null
				}
				return &object.Error{Message: "timeout must be an integer (ms)"}
			},
		},
		"tcp_remote_addr": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_remote_addr requires 1 argument (conn). got=%d", len(args))}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.String{Value: conn.RemoteAddr()}
			},
		},
		"tcp_local_addr": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tcp_local_addr requires 1 argument (conn/server). got=%d", len(args))}
				}
				if srv, ok := args[0].(*object.TcpServer); ok {
					return &object.String{Value: srv.Addr()}
				}
				conn, err := extractNetConn(args[0])
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.String{Value: conn.LocalAddr()}
			},
		},
		"net_listen_udp": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("net_listen_udp requires 1 argument (addr). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				pc, err := net.ListenPacket("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{PacketConn: pc}
			},
		},
		"udp_listen": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("udp_listen requires 1 argument (addr). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				pc, err := net.ListenPacket("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{PacketConn: pc}
			},
		},
		"udp_connect": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := parseNetAddr(args[0])
				conn, err := net.Dial("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{Conn: conn}
			},
		},
		"udp_send_to": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 3 {
					return &object.Error{Message: fmt.Sprintf("udp_send_to requires 3 arguments (socket, data, addr). got=%d", len(args))}
				}
				u, ok := args[0].(*object.UdpSocket)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument must be UdpSocket, got %s", args[0].Type())}
				}
				var data []byte
				if strObj, ok := args[1].(*object.String); ok {
					data = []byte(strObj.Value)
				} else if bytesObj, ok := args[1].(*object.Bytes); ok {
					data = bytesObj.Value
				} else {
					data = []byte(args[1].Inspect())
				}
				addrStr := args[2].Inspect()
				if sObj, ok := args[2].(*object.String); ok {
					addrStr = sObj.Value
				}
				n, err := u.SendTo(data, addrStr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Integer{Value: int64(n)}
			},
		},
		"udp_recv_from": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("udp_recv_from requires 1 or 2 arguments (socket, [max_bytes]). got=%d", len(args))}
				}
				u, ok := args[0].(*object.UdpSocket)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument must be UdpSocket, got %s", args[0].Type())}
				}
				maxBytes := 4096
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						maxBytes = int(iObj.Value)
					}
				}
				data, addr, err := u.RecvFrom(maxBytes)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				pairs := make(map[object.HashKey]object.HashPair)
				dataKey := &object.String{Value: "data"}
				addrKey := &object.String{Value: "addr"}
				pairs[dataKey.HashKey()] = object.HashPair{Key: dataKey, Value: &object.String{Value: string(data)}}
				pairs[addrKey.HashKey()] = object.HashPair{Key: addrKey, Value: &object.String{Value: addr}}
				return &object.Hash{Pairs: pairs}
			},
		},
		"udp_close": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("udp_close requires 1 argument (socket). got=%d", len(args))}
				}
				u, ok := args[0].(*object.UdpSocket)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to udp_close must be UdpSocket, got %s", args[0].Type())}
				}
				if err := u.Close(); err != nil {
					return &object.Error{Message: err.Error()}
				}
				return Null
			},
		},
		"udp_set_timeout": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("udp_set_timeout requires 2 arguments (socket, ms). got=%d", len(args))}
				}
				u, ok := args[0].(*object.UdpSocket)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument must be UdpSocket, got %s", args[0].Type())}
				}
				if iObj, ok := args[1].(*object.Integer); ok {
					u.SetTimeout(iObj.Value)
					return Null
				}
				return &object.Error{Message: "timeout must be an integer (ms)"}
			},
		},
		"tls_dial": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) < 1 || len(args) > 4 {
					return &object.Error{Message: fmt.Sprintf("tls_dial requires 1 to 4 arguments (addr, [host_name], [timeout_ms], [opts]). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				hostName := ""
				if len(args) >= 2 && args[1] != Null {
					if sObj, ok := args[1].(*object.String); ok {
						hostName = sObj.Value
					} else {
						hostName = args[1].Inspect()
					}
				}
				var timeoutMs int64 = 10000
				if len(args) >= 3 && args[2] != Null {
					if iObj, ok := args[2].(*object.Integer); ok && iObj.Value > 0 {
						timeoutMs = iObj.Value
					}
				}
				insecure := false
				if len(args) >= 4 {
					if bObj, ok := args[3].(*object.Boolean); ok {
						insecure = bObj.Value
					} else if hObj, ok := args[3].(*object.Hash); ok {
						for _, pair := range hObj.Pairs {
							if pair.Key.Inspect() == "insecure_skip_verify" {
								if b, ok := pair.Value.(*object.Boolean); ok {
									insecure = b.Value
								}
							}
						}
					}
				}
				conn, err := dialTLS(addr, hostName, timeoutMs, insecure)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return conn
			},
		},
		"tls_listen": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("tls_listen requires 1 argument (addr). got=%d", len(args))}
				}
				addr := parseNetAddr(args[0])
				cert, err := generateSelfSignedCert()
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("generating cert failed: %s", err)}
				}
				tlsConf := &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}
				l, err := tls.Listen("tcp", addr, tlsConf)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpServer{Listener: l}
			},
		},
		"http_get": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				var headers object.Object
				if len(args) == 2 {
					headers = args[1]
				}
				return executeHttpRequest("GET", urlStr, "", headers, 0)
			},
		},
		"http_post": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				bodyStr := ""
				if sObj, ok := args[1].(*object.String); ok {
					bodyStr = sObj.Value
				} else {
					jsonData, _ := json.Marshal(bramandObjectToNative(args[1]))
					bodyStr = string(jsonData)
				}
				var headers object.Object
				if len(args) == 3 {
					headers = args[2]
				}
				return executeHttpRequest("POST", urlStr, bodyStr, headers, 0)
			},
		},
		"http_put": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				bodyStr := ""
				if sObj, ok := args[1].(*object.String); ok {
					bodyStr = sObj.Value
				} else {
					jsonData, _ := json.Marshal(bramandObjectToNative(args[1]))
					bodyStr = string(jsonData)
				}
				var headers object.Object
				if len(args) == 3 {
					headers = args[2]
				}
				return executeHttpRequest("PUT", urlStr, bodyStr, headers, 0)
			},
		},
		"http_delete": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				var headers object.Object
				if len(args) == 2 {
					headers = args[1]
				}
				return executeHttpRequest("DELETE", urlStr, "", headers, 0)
			},
		},
		"http_request": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				opts, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: "argument to http_request must be Hash"}
				}
				method := "GET"
				urlStr := ""
				bodyStr := ""
				var headers object.Object
				var timeoutMs int64

				for _, p := range opts.Pairs {
					k := p.Key.Inspect()
					switch k {
					case "method":
						method = strings.ToUpper(p.Value.Inspect())
						if s, ok := p.Value.(*object.String); ok {
							method = strings.ToUpper(s.Value)
						}
					case "url":
						urlStr = p.Value.Inspect()
						if s, ok := p.Value.(*object.String); ok {
							urlStr = s.Value
						}
					case "body":
						if s, ok := p.Value.(*object.String); ok {
							bodyStr = s.Value
						} else {
							b, _ := json.Marshal(bramandObjectToNative(p.Value))
							bodyStr = string(b)
						}
					case "headers":
						headers = p.Value
					case "timeout":
						if i, ok := p.Value.(*object.Integer); ok {
							timeoutMs = i.Value
						}
					}
				}
				if urlStr == "" {
					return &object.Error{Message: "http_request requires 'url' field"}
				}
				return executeHttpRequest(method, urlStr, bodyStr, headers, timeoutMs)
			},
		},
		"http_server": {
			Fn: func(args ...object.Object) object.Object {
				return object.NewHttpServer()
			},
		},
		"ws_connect": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				u := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					u = sObj.Value
				}
				conn, err := object.WsConnect(u)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return conn
			},
		},
		"ws_server": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error(), Fatal: true}
				}
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
				path := "/"
				if len(args) == 2 {
					if pObj, ok := args[1].(*object.String); ok {
						path = pObj.Value
					}
				}
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.WsServer{Listener: l, Path: path}
			},
		},
	})
}

func executeHttpRequest(method, urlStr, bodyStr string, headersObj object.Object, timeoutMs int64) object.Object {
	var bodyReader io.Reader
	if len(bodyStr) > 0 {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("http request error: %s", err)}
	}

	if hObj, ok := headersObj.(*object.Hash); ok {
		for _, pair := range hObj.Pairs {
			req.Header.Set(pair.Key.Inspect(), pair.Value.Inspect())
		}
	}

	timeout := 15 * time.Second
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("http request failed: %s", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("reading response body failed: %s", err)}
	}
	respBody := string(respBytes)

	pairs := make(map[object.HashKey]object.HashPair)
	setStr := func(k, v string) {
		kObj := &object.String{Value: k}
		pairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
	}
	statusKey := &object.String{Value: "status"}
	pairs[statusKey.HashKey()] = object.HashPair{Key: statusKey, Value: &object.Integer{Value: int64(resp.StatusCode)}}
	statusCodeKey := &object.String{Value: "status_code"}
	pairs[statusCodeKey.HashKey()] = object.HashPair{Key: statusCodeKey, Value: &object.Integer{Value: int64(resp.StatusCode)}}
	setStr("status_text", resp.Status)
	setStr("body", respBody)

	hdrPairs := make(map[object.HashKey]object.HashPair)
	for hk, hvs := range resp.Header {
		if len(hvs) > 0 {
			kObj := &object.String{Value: hk}
			hdrPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: hvs[0]}}
		}
	}
	hdrKey := &object.String{Value: "headers"}
	pairs[hdrKey.HashKey()] = object.HashPair{Key: hdrKey, Value: &object.Hash{Pairs: hdrPairs}}

	jsonKey := &object.String{Value: "json"}
	pairs[jsonKey.HashKey()] = object.HashPair{
		Key: jsonKey,
		Value: &object.Builtin{
			Fn: func(a ...object.Object) object.Object {
				var v interface{}
				if err := json.Unmarshal([]byte(respBody), &v); err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid JSON: %s", err)}
				}
				return nativeToBramandObject(v)
			},
		},
	}

	return &object.Hash{Pairs: pairs}
}
