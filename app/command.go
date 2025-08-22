package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/entity"
)

type CommandEvent struct {
	Ctx     *ConnContext
	Command string
	Args    [][]byte
}

type Handler func(CommandEvent)

func CommandHandler(store *Store) map[string]Handler {
	return map[string]Handler{
		"PING": func(e CommandEvent) {
			e.Ctx.Write(AppendString([]byte{}, "PONG"))
		},
		"ECHO": func(e CommandEvent) {
			if len(e.Args) > 0 {
				e.Ctx.Write(AppendBulkString([]byte{}, e.Args[0]))
			}
		},
		"SET": func(e CommandEvent) {
			if len(e.Args) >= 2 {
				key := string(e.Args[0])
				value := string(e.Args[1])
				expire := time.Time{}
				if len(e.Args) == 4 {
					opt := strings.ToUpper(string(e.Args[2]))
					if opt == "PX" {
						ms, err := strconv.Atoi(string(e.Args[3]))
						if err == nil {
							expire = time.Now().Add(time.Duration(ms) * time.Millisecond)
						}
					}
				}

				store.Set(key, value, expire)
				e.Ctx.Write(AppendString([]byte{}, "OK"))
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'SET' command"))
			}
		},
		"GET": func(e CommandEvent) {
			if len(e.Args) >= 1 {
				key := string(e.Args[0])
				value, ok := store.Get(key)
				if ok {
					e.Ctx.Write(AppendBulkString([]byte{}, []byte(value)))
				} else {
					e.Ctx.Write([]byte("$-1\r\n")) // nil bulk string
				}
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'GET' command"))
			}
		},
		"RPUSH": func(e CommandEvent) {
			fmt.Println("RPUSH")
			if len(e.Args) >= 2 {
				key := string(e.Args[0])
				value := e.Args[1:]
				length, ok := store.RPush(key, value)
				if ok {
					e.Ctx.Write(AppendInt([]byte{}, length))
				} else {
					e.Ctx.Write([]byte("$-1\r\n"))
				}
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'RPUSH' command"))
			}
		},
		"LPUSH": func(e CommandEvent) {
			if len(e.Args) >= 2 {
				key := string(e.Args[0])
				value := e.Args[1:]
				length, ok := store.LPush(key, value)
				if ok {
					e.Ctx.Write(AppendInt([]byte{}, length))
				} else {
					e.Ctx.Write([]byte("$-1\r\n"))
				}
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'RPUSH' command"))
			}
		},
		"LRANGE": func(e CommandEvent) {
			if len(e.Args) == 3 {
				key := string(e.Args[0])
				startPos, err := strconv.Atoi(string(e.Args[1]))
				if err != nil {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LRANGE' command"))
					return
				}
				endPos, err := strconv.Atoi(string(e.Args[2]))
				if err != nil {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LRANGE' command"))
					return
				}
				valueList, ok := store.LRange(key, startPos, endPos)
				if !ok {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LRANGE' command"))
					return
				}
				msg := AppendArray([]byte{}, len(valueList))
				for _, value := range valueList {
					msg = AppendBulkString(msg, value)
				}
				e.Ctx.Write(msg)
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'LRANGE' command"))
			}
		},
		"LLEN": func(e CommandEvent) {
			if len(e.Args) >= 1 {
				key := string(e.Args[0])
				value, ok := store.LLen(key)
				if ok {
					e.Ctx.Write(AppendInt([]byte{}, value))
				} else {
					// TODO 확인
					e.Ctx.Write([]byte(":0\r\n"))
				}
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'LLEN' command"))
			}
		},
		"LPOP": func(e CommandEvent) {
			if len(e.Args) == 1 {
				key := string(e.Args[0])
				data, ok := store.LPop(key, 1)
				if !ok {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LPOP' command"))
					return
				}
				e.Ctx.Write(AppendBulkString([]byte{}, data[0]))
			} else if len(e.Args) == 2 {
				key := string(e.Args[0])
				count, err := strconv.Atoi(string(e.Args[1]))
				if err != nil {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LPOP' command"))
					return
				}
				data, ok := store.LPop(key, count)
				if !ok {
					e.Ctx.Write(AppendError([]byte{}, "ERR wrong arguments for 'LPOP' command"))
					return
				}
				result := AppendArray([]byte{}, len(data))
				for _, value := range data {
					result = AppendBulkString(result, value)
				}

				e.Ctx.Write(result)
			} else {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'LPOP' command"))
			}
		},
		"BLPOP": func(e CommandEvent) {
			fmt.Println("BLPOP")
			if len(e.Args) != 2 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'BLPOP' command"))
				return
			}

			key := string(e.Args[0])
			secs, err := strconv.ParseFloat(string(e.Args[1]), 64)
			if err != nil || secs < 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR invalid timeout"))
				return
			}

			var to time.Duration
			if secs == 0 {
				to = 0
			} else {
				to = time.Duration(secs * float64(time.Second))
			}

			go func() {
				val, ok := store.BLPop(key, to)
				if !ok {
					e.Ctx.Write(AppendError([]byte{}, "ERR BLPOP failed"))
					return
				}

				if val == nil {
					e.Ctx.Write([]byte("$-1\r\n"))
					return
				}

				msg := AppendArray([]byte{}, 2)
				msg = AppendBulkString(msg, []byte(key))
				msg = AppendBulkString(msg, val)
				e.Ctx.Write(msg)
			}()
		},
		"TYPE": func(e CommandEvent) {
			if len(e.Args) != 1 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'TYPE' command"))
				return
			}
			key := string(e.Args[0])
			dataType := store.Type(key)
			e.Ctx.Write(AppendString([]byte{}, dataType))
		},
		"XADD": func(e CommandEvent) {
			if len(e.Args) < 3 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'XADD' command"))
				return
			}

			key := string(e.Args[0])
			id := string(e.Args[1])
			fields := make([]entity.FieldValue, 0, len(e.Args[2:])/2)

			count := len(e.Args[2:])
			if count%2 != 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'XADD' command"))
				return
			}

			for i := range count / 2 {
				fields = append(fields, entity.FieldValue{
					Key:   string(e.Args[2*i+2]),
					Value: string(e.Args[2*i+3]),
				})
			}
			id, err := store.XAdd(key, id, fields)
			if err != nil {
				e.Ctx.Write(AppendError([]byte{}, err.Error()))
				return
			}
			e.Ctx.Write(AppendBulkString([]byte{}, []byte(id)))
		},
		"XRANGE": func(e CommandEvent) {
			if len(e.Args) != 3 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'XRANGE' command"))
				return
			}
			key := string(e.Args[0])
			start := string(e.Args[1])
			end := string(e.Args[2])
			entries, err := store.XRange(key, start, end)
			if err != nil {
				e.Ctx.Write(AppendError([]byte{}, err.Error()))
				return
			}

			cmd := AppendArray([]byte{}, len(entries))
			for _, entry := range entries {
				entryArray := AppendArray([]byte{}, 2)
				entryArray = AppendBulkString(entryArray, []byte(fmt.Sprintf("%d-%d", entry.Id.Millis, entry.Id.Seq)))
				entryArray = AppendArray(entryArray, len(entry.Fields)*2)
				for _, f := range entry.Fields {
					entryArray = AppendBulkString(entryArray, []byte(f.Key))
					entryArray = AppendBulkString(entryArray, []byte(f.Value))
				}

				cmd = append(cmd, entryArray...)
			}

			e.Ctx.Write(cmd)
		},
		"XREAD": func(e CommandEvent) {
			if len(e.Args) < 3 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'XREAD' command"))
				return
			}
			i := 0
			blockDur := time.Duration(0)

			if strings.ToUpper(string(e.Args[i])) == "BLOCK" {
				i++
				ms, err := strconv.Atoi(string(e.Args[i]))
				if err != nil || ms < 0 {
					e.Ctx.Write(AppendError([]byte{}, "ERR invalid BLOCK value"))
					return
				}
				blockDur = time.Duration(ms) * time.Millisecond
				i++
			}

			if strings.ToUpper(string(e.Args[i])) != "STREAMS" {
				e.Ctx.Write(AppendError([]byte{}, "ERR syntax error, expected STREAMS"))
				return
			}
			i++

			numKeys := (len(e.Args) - i) / 2
			if numKeys <= 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR syntax error, missing keys or IDs"))
				return
			}

			keys := make([]string, numKeys)
			ids := make([]string, numKeys)

			for k := 0; k < numKeys; k++ {
				keys[k] = string(e.Args[i+k])
			}

			for k := 0; k < numKeys; k++ {
				ids[k] = string(e.Args[i+k+numKeys])
			}
			go func() {
				entries, err := store.XRead(blockDur, keys, ids)
				if err != nil {
					e.Ctx.Write(AppendError([]byte{}, err.Error()))
				}

				if len(entries) == 0 {
					e.Ctx.Write([]byte("$-1\r\n"))
					return
				}
				cmd := AppendArray([]byte{}, numKeys)
				for j, key := range keys {
					cmd = AppendArray(cmd, 2)
					cmd = AppendBulkString(cmd, []byte(key))
					cmd = AppendArray(cmd, len(entries[j]))
					for _, entry := range entries[j] {
						entryArray := AppendArray([]byte{}, 2)
						entryArray = AppendBulkString(entryArray, []byte(fmt.Sprintf("%d-%d", entry.Id.Millis, entry.Id.Seq)))
						entryArray = AppendArray(entryArray, len(entry.Fields)*2)
						for _, f := range entry.Fields {
							entryArray = AppendBulkString(entryArray, []byte(f.Key))
							entryArray = AppendBulkString(entryArray, []byte(f.Value))
						}
						cmd = append(cmd, entryArray...)
					}
				}
				e.Ctx.Write(cmd)
			}()
		},
		"INCR": func(e CommandEvent) {
			if len(e.Args) != 1 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'INCR' command"))
				return
			}
			key := string(e.Args[0])

			incr, err := store.Incr(key)
			if err != nil {
				e.Ctx.Write(AppendError([]byte{}, err.Error()))
				return
			}
			e.Ctx.Write(AppendInt([]byte{}, incr))
		},
		"MULTI": func(e CommandEvent) {
			if len(e.Args) != 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'MULTI' command"))
				return
			}

			if e.Ctx.tx.IsInTransaction() {
				return
			}

			e.Ctx.tx.InTransaction = true

			e.Ctx.Write(AppendString([]byte{}, "OK"))
		},
		"EXEC": func(e CommandEvent) {
			if len(e.Args) != 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR wrong number of arguments for 'EXEC' command"))
				return
			}

			if !e.Ctx.tx.IsInTransaction() {
				e.Ctx.Write(AppendError([]byte{}, "ERR EXEC without MULTI"))
				return
			}

			e.Ctx.tx.InTransaction = false
			e.Ctx.Write(AppendArray([]byte{}, len(e.Ctx.tx.CmdQueue)))
			handlers := CommandHandler(store)

			for _, cmd := range e.Ctx.tx.CmdQueue {
				if handler, ok := handlers[cmd.Name]; ok {
					handler(e)
				} else {
					e.Ctx.Write(AppendError(nil, "ERR unknown command '"+cmd.Name+"'"))
				}
			}
		},
	}
}
