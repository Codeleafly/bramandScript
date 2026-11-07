package vm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"bramandscript/object"
)

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"tcp_listen": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error()}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpServer{Listener: l}
			},
		},
		"tcp_connect": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				}
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
		"udp_listen": {
			Fn: func(args ...object.Object) object.Object {
				if err := CheckPermission(PermNet); err != nil {
					return &object.Error{Message: err.Error()}
				}
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
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
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				}
				conn, err := net.Dial("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{Conn: conn}
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
					return &object.Error{Message: err.Error()}
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
