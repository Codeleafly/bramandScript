package vm

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"bramandscript/object"
)

func (vm *VM) executeNetMethod(left object.Object, index object.Object) (bool, error) {
	switch {
	case left.Type() == object.TCP_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.TcpServer)
		prop := index.(*object.String).Value
		switch prop {
		case "addr":
			return true, vm.push(&object.String{Value: s.Addr()})
		case "accept":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					conn, err := s.Accept()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return conn
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on TcpServer", prop)
		}

	case left.Type() == object.TCP_CONN_OBJ && index.Type() == object.STRING_OBJ:
		c := left.(*object.TcpConn)
		prop := index.(*object.String).Value
		switch prop {
		case "local_addr":
			return true, vm.push(&object.String{Value: c.LocalAddr()})
		case "remote_addr":
			return true, vm.push(&object.String{Value: c.RemoteAddr()})
		case "write", "send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "write requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := c.Write(data)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "read", "recv":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
						}
					}
					data, err := c.Read(maxBytes)
					if err != nil {
						if err == io.EOF {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: string(data)}
				},
			})
		case "read_line":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					line, err := c.ReadLine()
					if err != nil {
						if err == io.EOF && len(line) == 0 {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: line}
				},
			})
		case "set_timeout":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "set_timeout requires 1 argument (ms)"}
					}
					if intObj, ok := args[0].(*object.Integer); ok {
						c.SetTimeout(intObj.Value)
					}
					return Null
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := c.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on TcpConn", prop)
		}

	case left.Type() == object.UDP_SOCKET_OBJ && index.Type() == object.STRING_OBJ:
		u := left.(*object.UdpSocket)
		prop := index.(*object.String).Value
		switch prop {
		case "local_addr":
			return true, vm.push(&object.String{Value: u.LocalAddr()})
		case "send_to":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "send_to requires 2 arguments (data, addr)"}
					}
					addrObj, ok := args[1].(*object.String)
					if !ok {
						return &object.Error{Message: "address must be string"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := u.SendTo(data, addrObj.Value)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "recv_from":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
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
			})
		case "write", "send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "write requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := u.Write(data)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "read", "recv":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
						}
					}
					data, err := u.Read(maxBytes)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: string(data)}
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := u.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on UdpSocket", prop)
		}

	case left.Type() == object.WS_CONN_OBJ && index.Type() == object.STRING_OBJ:
		w := left.(*object.WsConn)
		prop := index.(*object.String).Value
		switch prop {
		case "is_open":
			return true, vm.push(nativeBoolToBooleanObject(w.IsOpen()))
		case "send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "send requires 1 argument"}
					}
					var msg string
					if strObj, ok := args[0].(*object.String); ok {
						msg = strObj.Value
					} else {
						msg = args[0].Inspect()
					}
					if err := w.Send(msg); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "send_binary":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "send_binary requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					if err := w.SendBinary(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "recv":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					msg, err := w.Recv()
					if err != nil {
						if err == io.EOF {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: msg}
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := w.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on WsConn", prop)
		}

	case left.Type() == object.WS_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.WsServer)
		prop := index.(*object.String).Value
		switch prop {
		case "addr":
			return true, vm.push(&object.String{Value: s.Addr()})
		case "path":
			return true, vm.push(&object.String{Value: s.Path})
		case "accept":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					conn, err := s.Accept()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return conn
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on WsServer", prop)
		}

	case left.Type() == object.HTTP_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.HttpServer)
		prop := index.(*object.String).Value
		switch prop {
		case "get", "post", "put", "delete":
			method := strings.ToUpper(prop)
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: fmt.Sprintf("%s route requires 2 arguments (path, handler)", prop)}
					}
					pathObj, ok := args[0].(*object.String)
					if !ok {
						return &object.Error{Message: "route path must be string"}
					}
					s.AddRoute(method, pathObj.Value, args[1])
					return left
				},
			})
		case "use":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "use requires 1 argument (middleware)"}
					}
					s.Use(args[0])
					return left
				},
			})
		case "static":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "static requires 2 arguments (prefix, dir)"}
					}
					prefixObj, ok1 := args[0].(*object.String)
					dirObj, ok2 := args[1].(*object.String)
					if !ok1 || !ok2 {
						return &object.Error{Message: "static prefix and dir must be strings"}
					}
					s.Static(prefixObj.Value, dirObj.Value)
					return left
				},
			})
		case "listen":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) < 1 {
						return &object.Error{Message: "listen requires at least 1 argument (port or addr)"}
					}
					addr := "127.0.0.1:8080"
					if strObj, ok := args[0].(*object.String); ok {
						addr = strObj.Value
					} else if intObj, ok := args[0].(*object.Integer); ok {
						addr = fmt.Sprintf("127.0.0.1:%d", intObj.Value)
					}
					l, err := net.Listen("tcp", addr)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					s.Listener = l

					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						s.Mu.RLock()
						for prefix, dir := range s.StaticDirs {
							if strings.HasPrefix(r.URL.Path, prefix) {
								s.Mu.RUnlock()
								http.StripPrefix(prefix, http.FileServer(http.Dir(dir))).ServeHTTP(w, r)
								return
							}
						}
						s.Mu.RUnlock()

						bodyBytes, _ := io.ReadAll(r.Body)
						bodyStr := string(bodyBytes)

						reqPairs := make(map[object.HashKey]object.HashPair)
						setStr := func(k, v string) {
							kObj := &object.String{Value: k}
							reqPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
						}
						setStr("method", r.Method)
						setStr("path", r.URL.Path)
						setStr("body", bodyStr)

						qPairs := make(map[object.HashKey]object.HashPair)
						for qk, qvs := range r.URL.Query() {
							if len(qvs) > 0 {
								kObj := &object.String{Value: qk}
								qPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: qvs[0]}}
							}
						}
						qKey := &object.String{Value: "query"}
						reqPairs[qKey.HashKey()] = object.HashPair{Key: qKey, Value: &object.Hash{Pairs: qPairs}}

						hdrPairs := make(map[object.HashKey]object.HashPair)
						for hk, hvs := range r.Header {
							if len(hvs) > 0 {
								kObj := &object.String{Value: hk}
								hdrPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: hvs[0]}}
							}
						}
						hKey := &object.String{Value: "headers"}
						reqPairs[hKey.HashKey()] = object.HashPair{Key: hKey, Value: &object.Hash{Pairs: hdrPairs}}

						jsonKey := &object.String{Value: "json"}
						reqPairs[jsonKey.HashKey()] = object.HashPair{
							Key: jsonKey,
							Value: &object.Builtin{
								Fn: func(a ...object.Object) object.Object {
									var v interface{}
									if err := json.Unmarshal([]byte(bodyStr), &v); err != nil {
										return &object.Error{Message: fmt.Sprintf("invalid JSON: %s", err)}
									}
									return nativeToBramandObject(v)
								},
							},
						}

						reqObj := &object.Hash{Pairs: reqPairs}
						rwObj := &object.HttpResponseWriter{W: w}

						// Middlewares
						s.Mu.RLock()
						mws := make([]object.Object, len(s.Middlewares))
						copy(mws, s.Middlewares)
						s.Mu.RUnlock()

						for _, mw := range mws {
							_, err := ExecuteCallable(mw, reqObj, rwObj)
							if err != nil {
								http.Error(w, err.Error(), 500)
								return
							}
							if rwObj.Written {
								return
							}
						}

						s.Mu.RLock()
						mRoutes := s.Routes[r.Method]
						var routeHandler object.Object
						if mRoutes != nil {
							routeHandler = mRoutes[r.URL.Path]
						}
						s.Mu.RUnlock()

						if routeHandler != nil {
							_, err := ExecuteCallable(routeHandler, reqObj, rwObj)
							if err != nil {
								http.Error(w, err.Error(), 500)
								return
							}
							if !rwObj.Written {
								_ = rwObj.Send(nil)
							}
							return
						}

						http.NotFound(w, r)
					})

					s.Server = &http.Server{Handler: handler}
					go func() {
						_ = s.Server.Serve(l)
					}()

					if len(args) >= 2 {
						_, _ = ExecuteCallable(args[1])
					}
					return left
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on HttpServer", prop)
		}

	case left.Type() == object.HTTP_RESPONSE_WRITER_OBJ && index.Type() == object.STRING_OBJ:
		rw := left.(*object.HttpResponseWriter)
		prop := index.(*object.String).Value
		switch prop {
		case "send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					var data []byte
					if len(args) > 0 {
						if strObj, ok := args[0].(*object.String); ok {
							data = []byte(strObj.Value)
						} else {
							data = []byte(args[0].Inspect())
						}
					}
					if err := rw.Send(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "json":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "json requires 1 argument"}
					}
					rw.SetHeader("Content-Type", "application/json")
					native := bramandObjectToNative(args[0])
					data, err := json.Marshal(native)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					if err := rw.Send(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "status":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "status requires 1 argument"}
					}
					if intObj, ok := args[0].(*object.Integer); ok {
						rw.SendStatus(int(intObj.Value))
					}
					return left
				},
			})
		case "set_header", "setHeader":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "set_header requires 2 arguments (key, value)"}
					}
					kObj, ok1 := args[0].(*object.String)
					vObj, ok2 := args[1].(*object.String)
					if !ok1 || !ok2 {
						return &object.Error{Message: "header key and value must be strings"}
					}
					rw.SetHeader(kObj.Value, vObj.Value)
					return left
				},
			})
		case "redirect":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) < 1 {
						return &object.Error{Message: "redirect requires at least 1 argument (url)"}
					}
					urlStr := args[0].Inspect()
					if sObj, ok := args[0].(*object.String); ok {
						urlStr = sObj.Value
					}
					code := 302
					if len(args) >= 2 {
						if intObj, ok := args[1].(*object.Integer); ok {
							code = int(intObj.Value)
						}
					}
					rw.SendStatus(code)
					rw.SetHeader("Location", urlStr)
					_ = rw.Send(nil)
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on HttpResponseWriter", prop)
		}
	}

	return false, nil
}
