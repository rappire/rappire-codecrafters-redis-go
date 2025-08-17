package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CommandEvent struct {
	Ctx     *ConnContext
	Command string
	Args    [][]byte
}

type Handler func(CommandEvent)

func NewHandler(store *Store) map[string]Handler {

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
			secs, err := strconv.ParseFloat(string(e.Args[1]), 32)
			if err != nil || secs < 0 {
				e.Ctx.Write(AppendError([]byte{}, "ERR invalid timeout"))
				return
			}

			var to time.Duration
			if secs == 0 {
				to = 0
			} else {
				to = time.Duration(secs) * time.Second
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
	}
}
