package commands

import (
	"fmt"
	"slices"

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
	if slices.Contains(cm.replicas, e.Ctx) {
		return
	}

	cm.replicas = append(cm.replicas, e.Ctx)
}

func (cm *CommandManger) handlePsync(e types.CommandEvent) {
	e.Ctx.Write(protocol.AppendString([]byte{}, "FULLRESYNC "+cm.serverInfo.GetReplId()+" 0"))

	emptyRDBHex := []byte{
		0x52, 0x45, 0x44, 0x49, 0x53, 0x30, 0x30, 0x30,
		0x39, 0xFA, 0x0A, 0x6D, 0x69, 0x6E, 0x64, 0x75,
		0x6D, 0x70, 0xFA, 0x09, 0x72, 0x65, 0x64, 0x69,
		0x73, 0x2D, 0x76, 0x65, 0x72, 0x05, 0x37, 0x2E,
		0x30, 0x2E, 0x30, 0xFF, 0xA7, 0x5E, 0xED, 0xB3,
		0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0xFF, 0xFF, 0xFF, 0xFF,
	}

	header := fmt.Sprintf("$%d\r\n", len(emptyRDBHex))
	e.Ctx.Write([]byte(header))
	e.Ctx.Write(emptyRDBHex)
}
