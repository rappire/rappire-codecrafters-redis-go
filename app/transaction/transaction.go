package transaction

import "strings"

type Cmd struct {
	Name string
	Args [][]byte
}

type Transaction struct {
	CmdQueue      []Cmd
	InTransaction bool
}

func NewTransaction() *Transaction {
	return &Transaction{
		CmdQueue:      []Cmd{},
		InTransaction: false,
	}
}

func (tx *Transaction) AddCommand(cmd Cmd) bool {
	switch cmd.Name {
	case "BLPOP":
		return false
	case "XREAD":
		if len(cmd.Args) > 0 && strings.ToUpper(string(cmd.Args[0])) == "BLOCK" {
			return false
		}
	}

	tx.CmdQueue = append(tx.CmdQueue, cmd)
	return true
}

func (tx *Transaction) IsInTransaction() bool {
	return tx.InTransaction
}

func (tx *Transaction) Start() {
	tx.InTransaction = true
	tx.CmdQueue = make([]Cmd, 0) // 명령어 큐 초기화
}

func (tx *Transaction) Discard() {
	tx.InTransaction = false
	tx.CmdQueue = make([]Cmd, 0)
}

func (tx *Transaction) GetCommands() []Cmd {
	return tx.CmdQueue
}

func (tx *Transaction) Clear() {
	tx.CmdQueue = make([]Cmd, 0)
	tx.InTransaction = false
}

func IsTransactionCommand(commandName string) bool {
	switch commandName {
	case "MULTI", "EXEC", "DISCARD", "WATCH", "UNWATCH":
		return true
	default:
		return false
	}
}
