package commands

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

type ServerInfoProvider interface {
	GetInfo() string
	GetReplId() string
}

type CommandManger struct {
	handlers   map[string]types.Handler
	store      *store.Store
	serverInfo ServerInfoProvider
	replicas   []*types.ConnContext
}

func NewCommandManger(store *store.Store, serverInfo ServerInfoProvider) *CommandManger {
	commandManger := &CommandManger{
		handlers:   make(map[string]types.Handler),
		store:      store,
		serverInfo: serverInfo,
		replicas:   make([]*types.ConnContext, 0),
	}
	commandManger.registerBasicCommands()
	commandManger.registerStringCommands()
	commandManger.registerStreamCommands()
	commandManger.registerTransactionCommands()
	commandManger.registerListCommands()

	return commandManger
}

func (cm *CommandManger) register(command string, handler types.Handler) {
	cm.handlers[command] = handler
}

func (cm *CommandManger) GetHandler(command string) (*types.Handler, bool) {
	handler, exists := cm.handlers[command]
	return &handler, exists
}

func (cm *CommandManger) Replicate(e types.CommandEvent) {
	msg := protocol.AppendArray([]byte{}, len(e.Args)+1)
	msg = protocol.AppendBulkString(msg, []byte(e.Command))
	for _, arg := range e.Args {
		msg = protocol.AppendBulkString(msg, arg)
	}
	for _, replica := range cm.replicas {
		replica.Write(msg)
	}
	fmt.Println(msg)
	fmt.Println(string(msg))
}
