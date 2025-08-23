package types

import (
	"net"
	"sync"

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
