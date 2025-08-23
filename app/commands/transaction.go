package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

// registerTransactionCommands는 트랜잭션 관련 명령어들을 등록합니다
func (cm *CommandManger) registerTransactionCommands() {
	cm.register("MULTI", cm.handleMulti)
	cm.register("EXEC", cm.handleExec)
	cm.register("DISCARD", cm.handleDiscard)
}

// handleMulti는 MULTI 명령어를 처리합니다
func (cm *CommandManger) handleMulti(e types.CommandEvent) {
	ParseAndExecute(e, func(args *MultiArgs) {
		tx := e.Ctx.GetTransaction()
		if tx.IsInTransaction() {
			e.Ctx.Write(protocol.AppendError([]byte{}, "ERR MULTI calls can not be nested"))
			return
		}
		tx.Start()
		e.Ctx.Write(protocol.AppendString([]byte{}, "OK"))
	})
}

// handleExec는 EXEC 명령어를 처리합니다
func (cm *CommandManger) handleExec(e types.CommandEvent) {
	if len(e.Args) != 0 {
		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR wrong number of arguments for 'exec' command"))
		return
	}

	tx := e.Ctx.GetTransaction()
	if !tx.IsInTransaction() {
		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR EXEC without MULTI"))
		return
	}

	commands := tx.GetCommands()
	tx.Clear() // 트랜잭션 상태 클리어

	// 배열 응답 시작
	e.Ctx.Write(protocol.AppendArray([]byte{}, len(commands)))

	// 각 명령어를 순차적으로 실행
	for _, cmd := range commands {
		handler, exists := cm.handlers[cmd.Name]
		if exists {
			// 새로운 이벤트 생성해서 핸들러 실행
			cmdEvent := types.CommandEvent{
				Ctx:     e.Ctx,
				Command: cmd.Name,
				Args:    cmd.Args,
			}
			handler(cmdEvent)
		} else {
			e.Ctx.Write(protocol.AppendError([]byte{}, "ERR unknown command '"+cmd.Name+"'"))
		}
	}
}

// handleDiscard는 DISCARD 명령어를 처리합니다
func (cm *CommandManger) handleDiscard(e types.CommandEvent) {
	if len(e.Args) != 0 {
		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR wrong number of arguments for 'discard' command"))
		return
	}

	tx := e.Ctx.GetTransaction()
	if !tx.IsInTransaction() {
		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR DISCARD without MULTI"))
		return
	}

	tx.Discard()
	e.Ctx.Write(protocol.AppendString([]byte{}, "OK"))
}
