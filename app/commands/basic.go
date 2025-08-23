package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

func (cm *CommandManger) registerBasicCommands() {
	cm.register("PING", cm.handlePing)
	cm.register("ECHO", cm.handleEcho)
	cm.register("TYPE", cm.handleType)
}

func (cm *CommandManger) handlePing(e CommandEvent) {
	e.Ctx.Write(protocol.AppendString([]byte{}, "PONG"))
}

func (cm *CommandManger) handleEcho(e CommandEvent) {
	ParseAndExecute(e, func(args *EchoArgs) {
		e.Ctx.Write(protocol.AppendBulkString([]byte{}, args.Message))
	})
}

func (cm *CommandManger) handleType(e CommandEvent) {
	ParseAndExecute(e, func(args *TypeArgs) {
		dataType := cm.store.Type(args.Key)
		e.Ctx.Write(protocol.AppendString([]byte{}, dataType))
	})
}
