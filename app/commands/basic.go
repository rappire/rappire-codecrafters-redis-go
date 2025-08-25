package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

func (cm *CommandManger) registerBasicCommands() {
	cm.register("PING", cm.handlePing)
	cm.register("ECHO", cm.handleEcho)
	cm.register("TYPE", cm.handleType)
	cm.register("INFO", cm.handleInfo)
	cm.register("REPLCONF", cm.handleReplConf)
	cm.register("PSYNC", cm.handlePsync)
}

func (cm *CommandManger) handlePing(e types.CommandEvent) {
	e.Ctx.Write(protocol.AppendString([]byte{}, "PONG"))
}

func (cm *CommandManger) handleEcho(e types.CommandEvent) {
	ParseAndExecute(e, func(args *EchoArgs) {
		e.Ctx.Write(protocol.AppendBulkString([]byte{}, args.Message))
	})
}

func (cm *CommandManger) handleType(e types.CommandEvent) {
	ParseAndExecute(e, func(args *TypeArgs) {
		dataType := cm.store.Type(args.Key)
		e.Ctx.Write(protocol.AppendString([]byte{}, dataType))
	})
}

func (cm *CommandManger) handleInfo(e types.CommandEvent) {
	ParseAndExecute(e, func(args *InfoArgs) {
		e.Ctx.Write(protocol.AppendBulkString([]byte{}, []byte(cm.serverInfo.GetInfo())))
	})
}

func (cm *CommandManger) handleReplConf(e types.CommandEvent) {
	e.Ctx.Write(protocol.AppendString([]byte{}, "OK"))
}

func (cm *CommandManger) handlePsync(e types.CommandEvent) {
	e.Ctx.Write(protocol.AppendString([]byte{}, "FULLRESYNC "+cm.serverInfo.GetReplId()+" 0"))
}
