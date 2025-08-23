package commands

import (
	"fmt"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/store/entity"
)

// registerStreamCommands는 스트림 관련 명령어들을 등록합니다
func (cm *CommandManger) registerStreamCommands() {
	cm.register("XADD", cm.handleXAdd)
	cm.register("XRANGE", cm.handleXRange)
	cm.register("XREAD", cm.handleXRead)
}

// handleXAdd는 XADD 명령어를 처리합니다
func (cm *CommandManger) handleXAdd(e CommandEvent) {
	ParseAndExecute(e, func(args *XAddArgs) {
		fields := make([]entity.FieldValue, 0, len(args.Fields))
		for _, field := range args.Fields {
			fields = append(fields, entity.FieldValue{Key: field, Value: args.Fields[field]})
		}
		generatedId, err := cm.store.XAdd(args.Key, args.ID, fields)
		if err != nil {
			e.Ctx.Write(protocol.AppendError([]byte{}, err.Error()))
			return
		}

		e.Ctx.Write(protocol.AppendBulkString([]byte{}, []byte(generatedId)))
	})
}

// handleXRange는 XRANGE 명령어를 처리합니다
func (cm *CommandManger) handleXRange(e CommandEvent) {
	ParseAndExecute(e, func(args *XRangeArgs) {
		entries, err := cm.store.XRange(args.Key, args.Start, args.End)
		if err != nil {
			e.Ctx.Write(protocol.AppendError([]byte{}, err.Error()))
			return
		}

		// 결과를 RESP 배열로 변환
		msg := protocol.AppendArray([]byte{}, len(entries))
		for _, entry := range entries {
			// 각 엔트리는 [ID, [field1, value1, field2, value2, ...]] 형태
			entryArray := protocol.AppendArray([]byte{}, 2)

			// ID 추가
			entryArray = protocol.AppendBulkString(entryArray, []byte(fmt.Sprintf("%d-%d", entry.Id.Millis, entry.Id.Seq)))

			// 필드-값 배열 추가
			entryArray = protocol.AppendArray(entryArray, len(entry.Fields)*2)
			for _, field := range entry.Fields {
				entryArray = protocol.AppendBulkString(entryArray, []byte(field.Key))
				entryArray = protocol.AppendBulkString(entryArray, []byte(field.Value))
			}

			msg = append(msg, entryArray...)
		}

		e.Ctx.Write(msg)
	})
}

// handleXRead는 XREAD 명령어를 처리합니다
func (cm *CommandManger) handleXRead(e CommandEvent) {
	ParseAndExecute(e, func(args *XReadArgs) {
		//if len(e.Args) < 3 {
		//	e.Ctx.Write(protocol.AppendError([]byte{}, "ERR wrong number of arguments for 'xread' command"))
		//	return
		//}
		//
		//argIndex := 0
		//var timeout time.Duration
		//
		//// BLOCK 옵션 확인
		//if strings.ToUpper(string(e.Args[argIndex])) == "BLOCK" {
		//	argIndex++
		//	if argIndex >= len(e.Args) {
		//		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR syntax error"))
		//		return
		//	}
		//
		//	timeoutMs, err := strconv.Atoi(string(e.Args[argIndex]))
		//	if err != nil || timeoutMs < 0 {
		//		e.Ctx.Write(protocol.AppendError([]byte{}, "ERR invalid timeout"))
		//		return
		//	}
		//
		//	if timeoutMs == 0 {
		//		timeout = time.Duration(0) // 무한 대기
		//	} else {
		//		timeout = time.Duration(timeoutMs) * time.Millisecond
		//	}
		//	argIndex++
		//}
		//
		//// STREAMS 키워드 확인
		//if argIndex >= len(e.Args) || strings.ToUpper(string(e.Args[argIndex])) != "STREAMS" {
		//	e.Ctx.Write(protocol.AppendError([]byte{}, "ERR syntax error"))
		//	return
		//}
		//argIndex++
		//
		//// 키와 ID 파싱
		//remainingArgs := e.Args[argIndex:]
		//if len(remainingArgs)%2 != 0 || len(remainingArgs) == 0 {
		//	e.Ctx.Write(protocol.AppendError([]byte{}, "ERR syntax error"))
		//	return
		//}
		//
		//numStreams := len(remainingArgs) / 2
		//keys := make([]string, numStreams)
		//ids := make([]string, numStreams)
		//
		//// 키 파싱
		//for i := 0; i < numStreams; i++ {
		//	keys[i] = string(remainingArgs[i])
		//}
		//
		//// ID 파싱
		//for i := 0; i < numStreams; i++ {
		//	ids[i] = string(remainingArgs[i+numStreams])
		//}

		// 블로킹 연산이므로 고루틴에서 실행
		go func() {
			entries, err := cm.store.XRead(time.Duration(args.Timeout), args.Keys, args.IDs)
			if err != nil {
				e.Ctx.Write(protocol.AppendError([]byte{}, err.Error()))
				return
			}

			if len(entries) == 0 || allEntriesEmpty(entries) {
				e.Ctx.Write(protocol.AppendNilBulkString())
				return
			}

			// 결과를 RESP 배열로 변환
			msg := protocol.AppendArray([]byte{}, len(args.Keys))
			for i, key := range args.Keys {
				keyArray := protocol.AppendArray([]byte{}, 2)
				keyArray = protocol.AppendBulkString(keyArray, []byte(key))

				streamEntries := entries[i]
				keyArray = protocol.AppendArray(keyArray, len(streamEntries))

				for _, entry := range streamEntries {
					entryArray := protocol.AppendArray([]byte{}, 2)
					entryArray = protocol.AppendBulkString(entryArray, []byte(fmt.Sprintf("%d-%d", entry.Id.Millis, entry.Id.Seq)))
					entryArray = protocol.AppendArray(entryArray, len(entry.Fields)*2)

					for _, field := range entry.Fields {
						entryArray = protocol.AppendBulkString(entryArray, []byte(field.Key))
						entryArray = protocol.AppendBulkString(entryArray, []byte(field.Value))
					}

					keyArray = append(keyArray, entryArray...)
				}

				msg = append(msg, keyArray...)
			}

			e.Ctx.Write(msg)
			fmt.Println("XREAD completed")
		}()
	})
}

// allEntriesEmpty는 모든 스트림 엔트리가 비어있는지 확인합니다
func allEntriesEmpty(entries [][]entity.StreamEntry) bool {
	for _, streamEntries := range entries {
		if len(streamEntries) > 0 {
			return false
		}
	}
	return true
}
