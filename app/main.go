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

type ConnContext struct {
	Conn net.Conn
	mu   sync.Mutex
	tx   *Transaction
}

type Cmd struct {
	Name string
	Args [][]byte
}

type Transaction struct {
	CmdQueue      []Cmd
	InTransaction bool
}

func (tx *Transaction) AddCommand(cmd Cmd) bool {
	switch cmd.Name {
	case "BLPOP":
		{
			return false
		}
	case "XRANGE":
		{
			if strings.ToUpper(string(cmd.Args[0])) == "BLOCK" {
				return false
			}
		}
	}
	tx.CmdQueue = append(tx.CmdQueue, cmd)
	return true
}

func (tx *Transaction) IsInTransaction() bool {
	return tx.InTransaction
}

func WrapHandlerForTransaction(handler Handler, commandName string) Handler {
	return func(e CommandEvent) {
		if commandName == "MULTI" || commandName == "EXEC" || commandName == "DISCARD" || commandName == "WATCH" || commandName == "UNWATCH" {
			handler(e)
			return
		}

		if e.Ctx.tx.IsInTransaction() {
			e.Ctx.tx.AddCommand(Cmd{Name: e.Command, Args: e.Args})
			e.Ctx.Write(AppendString([]byte{}, "QUEUED"))
			return
		}
		handler(e)
	}

}

// TODO 임시로 무시
func (connContext *ConnContext) Write(message []byte) int {
	connContext.mu.Lock()
	defer connContext.mu.Unlock()
	n, _ := connContext.Conn.Write(message)
	return n
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	eventChan := make(chan CommandEvent, 100)
	store := NewStore()
	handlers := CommandHandler(store)

	go func() {
		for ev := range eventChan {
			if handler, ok := handlers[ev.Command]; ok {
				handlerWithTransaction := WrapHandlerForTransaction(handler, ev.Command)
				handlerWithTransaction(ev)
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
		ctx := &ConnContext{Conn: conn, tx: &Transaction{CmdQueue: []Cmd{}, InTransaction: false}}
		go handleConnection(ctx, eventChan)
	}
}

func handleConnection(ctx *ConnContext, eventChan chan CommandEvent) {
	defer ctx.Conn.Close()
	reader := bufio.NewReader(ctx.Conn)

	for {
		resp, err := readRESP(reader)
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			fmt.Println("Error reading response: ", err.Error())
			return
		}

		if resp.Type == Array && resp.Length > 0 {
			cmd := strings.ToUpper(string(resp.Arr[0].Data))
			args := make([][]byte, 0, resp.Length-1)
			for _, b := range resp.Arr[1:] {
				args = append(args, b.Data)
			}
			fmt.Print(cmd)
			for _, b := range args {
				fmt.Printf(" %s", string(b))
			}
			fmt.Println()
			eventChan <- CommandEvent{Command: cmd, Args: args, Ctx: ctx}
		}
	}
}
