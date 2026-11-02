package object

import (
	"bufio"
	"bytes"
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TCP Server & Connection
// ============================================================================

type TcpServer struct {
	Listener net.Listener
}

func (s *TcpServer) Type() ObjectType { return TCP_SERVER_OBJ }
func (s *TcpServer) Inspect() string  { return fmt.Sprintf("tcp_server(%s)", s.Addr()) }

func (s *TcpServer) Addr() string {
	if s.Listener != nil {
		return s.Listener.Addr().String()
	}
	return ""
}

func (s *TcpServer) Accept() (*TcpConn, error) {
	conn, err := s.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &TcpConn{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}, nil
}

func (s *TcpServer) Close() error {
	if s.Listener != nil {
		return s.Listener.Close()
	}
	return nil
}

type TcpConn struct {
	Conn   net.Conn
	Reader *bufio.Reader
	mu     sync.Mutex
}

func (c *TcpConn) Type() ObjectType { return TCP_CONN_OBJ }
func (c *TcpConn) Inspect() string {
	if c.Conn != nil {
		return fmt.Sprintf("tcp_conn(%s -> %s)", c.LocalAddr(), c.RemoteAddr())
	}
	return "tcp_conn(closed)"
}

func (c *TcpConn) LocalAddr() string {
	if c.Conn != nil {
		return c.Conn.LocalAddr().String()
	}
	return ""
}

func (c *TcpConn) RemoteAddr() string {
	if c.Conn != nil {
		return c.Conn.RemoteAddr().String()
	}
	return ""
}

func (c *TcpConn) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Conn == nil {
		return 0, fmt.Errorf("connection is closed")
	}
	return c.Conn.Write(data)
}

func (c *TcpConn) Read(maxBytes int) ([]byte, error) {
	if c.Conn == nil {
		return nil, fmt.Errorf("connection is closed")
	}
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	buf := make([]byte, maxBytes)
	n, err := c.Reader.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (c *TcpConn) ReadLine() (string, error) {
	if c.Conn == nil {
		return "", fmt.Errorf("connection is closed")
	}
	line, err := c.Reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (c *TcpConn) SetTimeout(ms int64) {
	if c.Conn != nil {
		if ms > 0 {
			deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
			_ = c.Conn.SetDeadline(deadline)
		} else {
			_ = c.Conn.SetDeadline(time.Time{})
		}
	}
}

func (c *TcpConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Conn != nil {
		err := c.Conn.Close()
		c.Conn = nil
		return err
	}
	return nil
}

// ============================================================================
// UDP Socket
// ============================================================================

type UdpSocket struct {
	PacketConn net.PacketConn
	Conn       net.Conn
	mu         sync.Mutex
}

func (u *UdpSocket) Type() ObjectType { return UDP_SOCKET_OBJ }
func (u *UdpSocket) Inspect() string  { return fmt.Sprintf("udp_socket(%s)", u.LocalAddr()) }

func (u *UdpSocket) LocalAddr() string {
	if u.PacketConn != nil {
		return u.PacketConn.LocalAddr().String()
	}
	if u.Conn != nil {
		return u.Conn.LocalAddr().String()
	}
	return ""
}

func (u *UdpSocket) SendTo(data []byte, addrStr string) (int, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.PacketConn == nil {
		return 0, fmt.Errorf("socket is not listening")
	}
	dstAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return 0, err
	}
	return u.PacketConn.WriteTo(data, dstAddr)
}

func (u *UdpSocket) RecvFrom(maxBytes int) ([]byte, string, error) {
	if u.PacketConn == nil {
		return nil, "", fmt.Errorf("socket is not listening")
	}
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	buf := make([]byte, maxBytes)
	n, addr, err := u.PacketConn.ReadFrom(buf)
	if err != nil {
		return nil, "", err
	}
	return buf[:n], addr.String(), nil
}

func (u *UdpSocket) Write(data []byte) (int, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.Conn == nil {
		return 0, fmt.Errorf("socket is not connected")
	}
	return u.Conn.Write(data)
}

func (u *UdpSocket) Read(maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	buf := make([]byte, maxBytes)
	if u.Conn != nil {
		n, err := u.Conn.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
	if u.PacketConn != nil {
		n, _, err := u.PacketConn.ReadFrom(buf)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
	return nil, fmt.Errorf("socket is not connected")
}

func (u *UdpSocket) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	var err error
	if u.PacketConn != nil {
		err = u.PacketConn.Close()
		u.PacketConn = nil
	}
	if u.Conn != nil {
		err2 := u.Conn.Close()
		u.Conn = nil
		if err == nil {
			err = err2
		}
	}
	return err
}

// ============================================================================
// RFC 6455 WebSocket Implementation (Pure Go Standard Library)
// ============================================================================

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func computeWebSocketAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

type WsConn struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	IsServer bool
	Closed   bool
	mu       sync.Mutex
}

func (w *WsConn) Type() ObjectType { return WS_CONN_OBJ }
func (w *WsConn) Inspect() string {
	if w.IsOpen() {
		return fmt.Sprintf("ws_conn(%s)", w.Conn.RemoteAddr())
	}
	return "ws_conn(closed)"
}

func (w *WsConn) IsOpen() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return !w.Closed && w.Conn != nil
}

func (w *WsConn) Send(msg string) error {
	return w.sendFrame(0x1, []byte(msg)) // 0x1 = Text frame
}

func (w *WsConn) SendBinary(data []byte) error {
	return w.sendFrame(0x2, data) // 0x2 = Binary frame
}

func (w *WsConn) sendFrame(opcode byte, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.Closed || w.Conn == nil {
		return fmt.Errorf("websocket connection is closed")
	}

	var header bytes.Buffer
	// Byte 0: FIN (0x80) | Opcode
	header.WriteByte(0x80 | opcode)

	mask := !w.IsServer // Client MUST mask frames (RFC 6455)
	var maskBit byte
	if mask {
		maskBit = 0x80
	}

	length := len(payload)
	if length < 126 {
		header.WriteByte(maskBit | byte(length))
	} else if length <= 65535 {
		header.WriteByte(maskBit | 126)
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(length))
		header.Write(b)
	} else {
		header.WriteByte(maskBit | 127)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(length))
		header.Write(b)
	}

	if mask {
		var maskKey [4]byte
		_, _ = crand.Read(maskKey[:])
		header.Write(maskKey[:])

		maskedPayload := make([]byte, length)
		for i := 0; i < length; i++ {
			maskedPayload[i] = payload[i] ^ maskKey[i%4]
		}
		header.Write(maskedPayload)
	} else {
		header.Write(payload)
	}

	_, err := w.Conn.Write(header.Bytes())
	return err
}

func (w *WsConn) Recv() (string, error) {
	for {
		w.mu.Lock()
		if w.Closed || w.Conn == nil {
			w.mu.Unlock()
			return "", io.EOF
		}
		reader := w.Reader
		w.mu.Unlock()

		// Read byte 0
		b0, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		opcode := b0 & 0x0F

		// Read byte 1
		b1, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		hasMask := (b1 & 0x80) != 0
		lenByte := b1 & 0x7F

		var payloadLen int64
		if lenByte < 126 {
			payloadLen = int64(lenByte)
		} else if lenByte == 126 {
			var l uint16
			if err := binary.Read(reader, binary.BigEndian, &l); err != nil {
				return "", err
			}
			payloadLen = int64(l)
		} else {
			var l uint64
			if err := binary.Read(reader, binary.BigEndian, &l); err != nil {
				return "", err
			}
			payloadLen = int64(l)
		}

		var maskKey [4]byte
		if hasMask {
			if _, err := io.ReadFull(reader, maskKey[:]); err != nil {
				return "", err
			}
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return "", err
		}

		if hasMask {
			for i := int64(0); i < payloadLen; i++ {
				payload[i] ^= maskKey[i%4]
			}
		}

		switch opcode {
		case 0x1, 0x2: // Text or Binary
			return string(payload), nil
		case 0x8: // Close frame
			_ = w.sendFrame(0x8, payload)
			_ = w.Close()
			return "", io.EOF
		case 0x9: // Ping frame -> auto reply with Pong
			_ = w.sendFrame(0xA, payload)
		case 0xA: // Pong frame -> ignore and continue
			continue
		}
	}
}

func (w *WsConn) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.Closed && w.Conn != nil {
		w.Closed = true
		return w.Conn.Close()
	}
	return nil
}

// ============================================================================
// WebSocket Server & Client Handshake Helpers
// ============================================================================

type WsServer struct {
	Listener net.Listener
	Path     string
}

func (s *WsServer) Type() ObjectType { return WS_SERVER_OBJ }
func (s *WsServer) Inspect() string  { return fmt.Sprintf("ws_server(%s%s)", s.Addr(), s.Path) }

func (s *WsServer) Addr() string {
	if s.Listener != nil {
		return s.Listener.Addr().String()
	}
	return ""
}

func (s *WsServer) Accept() (*WsConn, error) {
	conn, err := s.Listener.Accept()
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	reqLine, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}

	wsKey := headers["sec-websocket-key"]
	if wsKey == "" {
		_ = conn.Close()
		return nil, fmt.Errorf("missing Sec-WebSocket-Key in handshake: %s", reqLine)
	}

	acceptVal := computeWebSocketAccept(wsKey)
	res := fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Accept: %s\r\n\r\n", acceptVal)

	if _, err := conn.Write([]byte(res)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &WsConn{
		Conn:     conn,
		Reader:   reader,
		IsServer: true,
	}, nil
}

func (s *WsServer) Close() error {
	if s.Listener != nil {
		return s.Listener.Close()
	}
	return nil
}

func WsConnect(rawURL string) (*WsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var conn net.Conn
	if u.Scheme == "wss" {
		conn, err = tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = net.Dial("tcp", host)
	}
	if err != nil {
		return nil, err
	}

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	var keyBytes [16]byte
	_, _ = crand.Read(keyBytes[:])
	key := base64.StdEncoding.EncodeToString(keyBytes[:])

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Key: %s\r\n"+
		"Sec-WebSocket-Version: 13\r\n\r\n", path, u.Host, key)

	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !strings.Contains(statusLine, "101") {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %s", statusLine)
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if strings.TrimRight(line, "\r\n") == "" {
			break
		}
	}

	return &WsConn{
		Conn:     conn,
		Reader:   reader,
		IsServer: false,
	}, nil
}

// ============================================================================
// Built-in HTTP Server
// ============================================================================

type HttpServer struct {
	Server      *http.Server
	Listener    net.Listener
	Routes      map[string]map[string]Object // Method -> Path -> Handler
	Middlewares []Object
	StaticDirs  map[string]string // Prefix -> LocalDirectory
	Mu          sync.RWMutex
}

func NewHttpServer() *HttpServer {
	return &HttpServer{
		Routes:     make(map[string]map[string]Object),
		StaticDirs: make(map[string]string),
	}
}

func (s *HttpServer) Type() ObjectType { return HTTP_SERVER_OBJ }
func (s *HttpServer) Inspect() string {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	addr := "not listening"
	if s.Listener != nil {
		addr = s.Listener.Addr().String()
	}
	return fmt.Sprintf("http_server(%s)", addr)
}

func (s *HttpServer) AddRoute(method, path string, handler Object) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	method = strings.ToUpper(method)
	if s.Routes[method] == nil {
		s.Routes[method] = make(map[string]Object)
	}
	s.Routes[method][path] = handler
}

func (s *HttpServer) Use(middleware Object) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Middlewares = append(s.Middlewares, middleware)
}

func (s *HttpServer) Static(prefix, dir string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.StaticDirs[prefix] = dir
}

func (s *HttpServer) Close() error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.Listener != nil {
		err := s.Listener.Close()
		s.Listener = nil
		return err
	}
	return nil
}

// ============================================================================
// HTTP Response Writer Helper Object
// ============================================================================

type HttpResponseWriter struct {
	W       http.ResponseWriter
	Status  int
	Written bool
	mu      sync.Mutex
}

func (rw *HttpResponseWriter) Type() ObjectType { return HTTP_RESPONSE_WRITER_OBJ }
func (rw *HttpResponseWriter) Inspect() string  { return "http_response_writer" }

func (rw *HttpResponseWriter) SendStatus(code int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.Status = code
}

func (rw *HttpResponseWriter) SetHeader(key, val string) {
	rw.W.Header().Set(key, val)
}

func (rw *HttpResponseWriter) Send(body []byte) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.Written {
		return nil
	}
	rw.Written = true
	code := rw.Status
	if code == 0 {
		code = http.StatusOK
	}
	rw.W.WriteHeader(code)
	_, err := rw.W.Write(body)
	return err
}
