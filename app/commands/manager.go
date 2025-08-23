package commands

import (
	"fmt"
	"net"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/codecrafters-io/redis-starter-go/app/transaction"
)

type ConnContext struct {
	Conn net.Conn
	mu   sync.Mutex
	tx   *transaction.Transaction
}

func NewConnContext(conn net.Conn, transaction *transaction.Transaction) *ConnContext {
	return &ConnContext{
		Conn: conn,
		tx:   transaction,
	}
}

func (ctx *ConnContext) Write(message []byte) int {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	n, err := ctx.Conn.Write(message)
	if err != nil {
		fmt.Printf("Error writing to client: %v\n", err)
		return 0
	}
	return n
}

func (ctx *ConnContext) GetTransaction() *transaction.Transaction {
	return ctx.tx
}

type CommandEvent struct {
	Ctx     *ConnContext
	Command string
	Args    [][]byte
}

type Handler func(CommandEvent)

type CommandManger struct {
	handlers map[string]Handler
	store    *store.Store
}

func NewCommandManger(store *store.Store) *CommandManger {
	commandManger := &CommandManger{
		handlers: make(map[string]Handler),
		store:    store,
	}
	commandManger.registerBasicCommands()
	commandManger.registerStringCommands()
	commandManger.registerStreamCommands()
	commandManger.registerTransactionCommands()
	commandManger.registerListCommands()

	return commandManger
}

func (cm *CommandManger) register(command string, handler Handler) {
	cm.handlers[command] = handler
}

func (cm *CommandManger) GetHandler(command string) (*Handler, bool) {
	handler, exists := cm.handlers[command]
	return &handler, exists
}
