package commands

import (
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
}

func NewCommandManger(store *store.Store, serverInfo ServerInfoProvider) *CommandManger {
	commandManger := &CommandManger{
		handlers:   make(map[string]types.Handler),
		store:      store,
		serverInfo: serverInfo,
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
