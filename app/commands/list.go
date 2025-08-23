package commands

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

// registerListCommands는 리스트 관련 명령어들을 등록합니다
func (cm *CommandManger) registerListCommands() {
	cm.register("RPUSH", cm.handleRPush)
	cm.register("LPUSH", cm.handleLPush)
	cm.register("LRANGE", cm.handleLRange)
	cm.register("LLEN", cm.handleLLen)
	cm.register("LPOP", cm.handleLPop)
	cm.register("BLPOP", cm.handleBLPop)
}

// handleRPush는 RPUSH 명령어를 처리합니다
func (cm *CommandManger) handleRPush(e CommandEvent) {
	ParseAndExecute(e, func(args *RPushArgs) {
		length, ok := cm.store.RPush(args.Key, args.Values)
		if ok {
			e.Ctx.Write(protocol.AppendInt([]byte{}, length))
		} else {
			e.Ctx.Write([]byte("$-1\r\n"))
		}
	})
}

// handleLPush는 LPUSH 명령어를 처리합니다
func (cm *CommandManger) handleLPush(e CommandEvent) {
	ParseAndExecute(e, func(args *LPushArgs) {
		length, ok := cm.store.LPush(args.Key, args.Values)
		if ok {
			e.Ctx.Write(protocol.AppendInt([]byte{}, length))
		} else {
			e.Ctx.Write([]byte("$-1\r\n"))
		}
	})
}

// handleLRange는 LRANGE 명령어를 처리합니다
func (cm *CommandManger) handleLRange(e CommandEvent) {
	ParseAndExecute(e, func(args *LRangeArgs) {
		values, ok := cm.store.LRange(args.Key, args.Start, args.End)
		if !ok {
			e.Ctx.Write(protocol.AppendError([]byte{}, "ERR wrong type"))
			return
		}

		msg := protocol.AppendArray([]byte{}, len(values))
		for _, value := range values {
			msg = protocol.AppendBulkString(msg, value)
		}
		e.Ctx.Write(msg)
	})
}

func (cm *CommandManger) handleLLen(e CommandEvent) {
	ParseAndExecute(e, func(args *LLenArgs) {
		length, ok := cm.store.LLen(args.Key)

		if ok {
			e.Ctx.Write(protocol.AppendInt([]byte{}, length))
		} else {
			e.Ctx.Write([]byte(":0\r\n"))
		}
	})
}

// handleLPop은 LPOP 명령어를 처리합니다
func (cm *CommandManger) handleLPop(e CommandEvent) {
	ParseAndExecute(e, func(args *LPopArgs) {
		data, ok := cm.store.LPop(args.Key, args.Count)
		if !ok {
			e.Ctx.Write(protocol.AppendError([]byte{}, "ERR wrong type"))
			return
		}

		if len(data) == 0 {
			e.Ctx.Write(protocol.AppendNilBulkString())
			return
		}

		if args.Count == 1 && len(data) > 0 {
			e.Ctx.Write(protocol.AppendBulkString([]byte{}, data[0]))
		} else {
			msg := protocol.AppendArray([]byte{}, len(data))
			for _, value := range data {
				msg = protocol.AppendBulkString(msg, value)
			}
			e.Ctx.Write(msg)
		}
	})
}

func (cm *CommandManger) handleBLPop(e CommandEvent) {
	ParseAndExecute(e, func(args *BLPopArgs) {
		go func() {
			value, ok := cm.store.BLPop(args.Key, args.GetTimeoutDuration())
			if !ok {
				e.Ctx.Write(protocol.AppendError([]byte{}, "ERR blpop failed"))
				return
			}

			if value == nil {
				e.Ctx.Write([]byte("$-1\r\n"))
				return
			}

			msg := protocol.AppendArray([]byte{}, 2)
			msg = protocol.AppendBulkString(msg, []byte(args.Key))
			msg = protocol.AppendBulkString(msg, value)
			e.Ctx.Write(msg)

			fmt.Println("BLPOP completed for key:", args.Key)
		}()
	})
}
