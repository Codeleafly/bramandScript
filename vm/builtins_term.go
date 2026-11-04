package vm

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"bramandscript/object"
)

var (
	termChan          = make(chan byte, 512)
	termReaderStarted = false
	termReaderLock    sync.Mutex
	isRawMode         = false
	termRawLock       sync.Mutex
	signalOnce        sync.Once
)

// IsTermReaderActive returns whether the terminal background reader has started.
func IsTermReaderActive() bool {
	termReaderLock.Lock()
	defer termReaderLock.Unlock()
	return termReaderStarted
}

// StartTermReader starts the background goroutine reading raw bytes from stdin.
func StartTermReader() {
	termReaderLock.Lock()
	defer termReaderLock.Unlock()
	if termReaderStarted {
		return
	}
	termReaderStarted = true

	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			// Non-blocking send with drop-oldest strategy to prevent buffer stall
			select {
			case termChan <- buf[0]:
			default:
				select {
				case <-termChan:
				default:
				}
				select {
				case termChan <- buf[0]:
				default:
				}
			}
		}
	}()
}

// ReadLineFromTermChan reads a line from the termChan buffer until newline.
func ReadLineFromTermChan() string {
	var sb strings.Builder
	for {
		b := <-termChan
		if b == '\n' {
			break
		}
		if b == '\r' {
			// Check if CRLF
			select {
			case next := <-termChan:
				if next != '\n' {
					sb.WriteByte(next)
				}
			case <-time.After(10 * time.Millisecond):
			}
			break
		}
		sb.WriteByte(b)
	}
	return sb.String()
}

// SetTermRaw toggles the terminal between raw and cooked/sane mode.
func SetTermRaw(raw bool) error {
	termRawLock.Lock()
	defer termRawLock.Unlock()

	if raw {
		if isRawMode {
			return nil
		}
		// Put terminal into raw mode: disable icanon (line buffering) and echo
		cmd := exec.Command("stty", "-icanon", "-echo")
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			cmdRaw := exec.Command("stty", "raw", "-echo")
			cmdRaw.Stdin = os.Stdin
			_ = cmdRaw.Run()
		}
		isRawMode = true
		StartTermReader()
		setupSignalRestore()
	} else {
		if !isRawMode {
			return nil
		}
		cmd := exec.Command("stty", "sane")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
		isRawMode = false
	}
	return nil
}

// RestoreTerm ensures terminal sane mode is restored.
func RestoreTerm() {
	termRawLock.Lock()
	defer termRawLock.Unlock()
	if isRawMode {
		cmd := exec.Command("stty", "sane")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
		isRawMode = false
	}
}

func setupSignalRestore() {
	signalOnce.Do(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			RestoreTerm()
			os.Exit(0)
		}()
	})
}

// ReadKeyWithTimeout reads a key event with a timeout in milliseconds.
// If timeoutMs <= 0, it performs a non-blocking check.
func ReadKeyWithTimeout(timeoutMs int64) string {
	StartTermReader()

	var firstByte byte
	if timeoutMs <= 0 {
		select {
		case b := <-termChan:
			firstByte = b
		default:
			return ""
		}
	} else {
		select {
		case b := <-termChan:
			firstByte = b
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			return ""
		}
	}

	// Handle Escape sequences (arrow keys, function keys)
	if firstByte == 0x1b {
		select {
		case b2 := <-termChan:
			if b2 == '[' {
				select {
				case b3 := <-termChan:
					switch b3 {
					case 'A':
						return "up"
					case 'B':
						return "down"
					case 'C':
						return "right"
					case 'D':
						return "left"
					case 'H':
						return "home"
					case 'F':
						return "end"
					case '1', '2', '3', '4', '5', '6':
						// E.g. \x1b[3~ (delete), \x1b[5~ (pageup), \x1b[6~ (pagedown)
						select {
						case b4 := <-termChan:
							if b4 == '~' {
								switch b3 {
								case '2':
									return "insert"
								case '3':
									return "delete"
								case '5':
									return "pageup"
								case '6':
									return "pagedown"
								}
							}
							return fmt.Sprintf("esc[%c%c", b3, b4)
						case <-time.After(15 * time.Millisecond):
							return fmt.Sprintf("esc[%c", b3)
						}
					default:
						return fmt.Sprintf("esc[%c", b3)
					}
				case <-time.After(15 * time.Millisecond):
					return "esc["
				}
			} else if b2 == 'O' {
				select {
				case b3 := <-termChan:
					switch b3 {
					case 'P':
						return "f1"
					case 'Q':
						return "f2"
					case 'R':
						return "f3"
					case 'S':
						return "f4"
					default:
						return fmt.Sprintf("escO%c", b3)
					}
				case <-time.After(15 * time.Millisecond):
					return "escO"
				}
			}
			return fmt.Sprintf("esc%c", b2)
		case <-time.After(15 * time.Millisecond):
			return "esc"
		}
	}

	// Special ASCII control characters
	switch firstByte {
	case 0x03:
		return "ctrl_c"
	case 0x04:
		return "ctrl_d"
	case '\r', '\n':
		return "enter"
	case ' ':
		return "space"
	case '\t':
		return "tab"
	case 0x7f, 0x08:
		return "backspace"
	default:
		return string([]byte{firstByte})
	}
}

func init() {
	RegisterBuiltin("term_raw", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			raw := true
			if len(args) > 0 {
				if b, ok := args[0].(*object.Boolean); ok {
					raw = b.Value
				} else if i, ok := args[0].(*object.Integer); ok {
					raw = i.Value != 0
				}
			}
			if err := SetTermRaw(raw); err != nil {
				return &object.Error{Message: fmt.Sprintf("term_raw error: %s", err)}
			}
			return True
		},
	})

	RegisterBuiltin("read_key", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var timeoutMs int64 = 0
			if len(args) > 0 {
				if i, ok := args[0].(*object.Integer); ok {
					timeoutMs = i.Value
				} else if f, ok := args[0].(*object.Float); ok {
					timeoutMs = int64(f.Value)
				}
			}
			key := ReadKeyWithTimeout(timeoutMs)
			return &object.String{Value: key}
		},
	})

	RegisterBuiltin("poll_key", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			key := ReadKeyWithTimeout(0)
			return &object.String{Value: key}
		},
	})
}
