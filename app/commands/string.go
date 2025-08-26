package commands

import (
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

func (cm *CommandManger) registerStringCommands() {
	cm.register("GET", cm.handleGet)
	cm.register("SET", cm.handleSet)
	cm.register("INCR", cm.handleIncr)
}

func (cm *CommandManger) handleGet(e types.CommandEvent) {
	ParseAndExecute(e, func(args *GetArgs) {
		value, exists := cm.store.Get(args.Key)

		if exists {
			e.Ctx.Write(protocol.AppendBulkString([]byte{}, []byte(value)))
		} else {
			e.Ctx.Write(protocol.AppendNilBulkString())
		}
	})
}

func (cm *CommandManger) handleSet(e types.CommandEvent) {
	ParseAndExecute(e, func(args *SetArgs) {
		var expire time.Time
		if exp := args.GetExpiration(); exp != nil {
			expire = *exp
		}

		cm.store.Set(args.Key, args.Value, expire)
		e.Ctx.Write(protocol.AppendString([]byte{}, "OK"))
	})

	cm.Replicate(e)
}

func (cm *CommandManger) handleIncr(e types.CommandEvent) {
	ParseAndExecute(e, func(args *IncrArgs) {
		result, err := cm.store.Incr(args.Key)

		if err != nil {
			e.Ctx.Write(protocol.AppendError([]byte{}, err.Error()))
			return
		}

		e.Ctx.Write(protocol.AppendInt([]byte{}, result))
	})
}
