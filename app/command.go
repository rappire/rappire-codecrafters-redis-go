package main

import (
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
			if len(e.Args) == 2 {
				key := string(e.Args[0])
				value := string(e.Args[1])
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
	}
}
