package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

type Event struct {
	Ctx     *ConnContext
	Command string
	Args    [][]byte
}
type ConnContext struct {
	Conn net.Conn
	mu   sync.Mutex
}

func (connContext *ConnContext) Write(message []byte) (n int, err error) {
	connContext.mu.Lock()
	defer connContext.mu.Unlock()
	return connContext.Conn.Write(message)
}

type Handler func(Event)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	eventChan := make(chan Event, 100)
	store := NewStore()

	handlers := map[string]Handler{
		"PING": func(e Event) {
			msg := AppendString([]byte{}, "PONG")
			e.Ctx.Write(msg)
		},
		"ECHO": func(e Event) {
			if len(e.Args) > 0 {
				e.Ctx.Write(AppendBulkString([]byte{}, e.Args[0]))
			}
		},
		"SET": func(e Event) {
			if len(e.Args) == 2 {
				key := string(e.Args[0])
				value := string(e.Args[1])
				store.Set(key, value)
				e.Ctx.Write(AppendString([]byte{}, "OK"))
			}
		},
		"GET": func(e Event) {
			if len(e.Args) > 0 {
				key := string(e.Args[0])
				value := store.Get(key)
				e.Ctx.Write(AppendBulkString([]byte{}, []byte(value)))
			}
		},
	}

	go func() {
		for ev := range eventChan {
			if handlers, ok := handlers[ev.Command]; ok {
				handlers(ev)
			} else {
				ev.Ctx.Write(AppendError(nil, "ERR unknown command '"+ev.Command+"'"))
			}
		}
	}()

	fmt.Println("listening on :6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("new connection:", conn.RemoteAddr())
		ctx := &ConnContext{Conn: conn}
		go handleConnection(ctx, eventChan)
	}
}

func handleConnection(ctx *ConnContext, eventChan chan Event) {
	defer ctx.Conn.Close()
	reader := bufio.NewReader(ctx.Conn)

	for {
		resp, err := readRESP(reader)
		if err != nil {
			fmt.Println("Error reading response: ", err.Error())
			return
		}

		if resp.Type == Array && resp.Length > 0 {
			cmd := strings.ToUpper(string(resp.Arr[0].Data))
			args := make([][]byte, 0, resp.Length-1)
			for _, b := range resp.Arr[1:] {
				args = append(args, b.Data)
			}
			eventChan <- Event{Command: cmd, Args: args, Ctx: ctx}
		}
	}
}
