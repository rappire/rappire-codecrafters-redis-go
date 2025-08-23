package server

import (
	"fmt"
	"math/rand"
	"time"
)

type ServerInfo struct {
	role                       string
	connectedSlave             int
	masterReplId               string
	masterReplOffset           int
	secondReplOffset           int
	replBacklogActive          int
	replBacklogSize            int
	replBacklogFirstByteOffset int
	replBacklogHistLen         int
}

func (s *ServerInfo) GetInfo() string {
	return fmt.Sprintf("role:%s\nconnected_slaves:%d\nmaster_replid:%s\nmaster_repl_offset:%d\nsecond_repl_offset:%d\nrepl_backlog_active:%d\nrepl_backlog_size:%d\nrepl_backlog_first_byte_offset:%d\nrepl_backlog_histlen:%d",
		s.role,
		s.connectedSlave,
		s.masterReplId,
		s.masterReplOffset,
		s.secondReplOffset,
		s.replBacklogActive,
		s.replBacklogSize,
		s.replBacklogFirstByteOffset,
		s.replBacklogHistLen)
}

func createMasterReplId() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 40

	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = chars[seededRand.Intn(len(chars))]
	}
	return string(b)
}

func NewServerInfo(role string) *ServerInfo {
	masterReplId := ""
	masterReplOffset := 0

	if role == "" {
		role = "master"
		masterReplId = createMasterReplId()
	} else {
		role = "slave"
	}

	return &ServerInfo{
		role:             role,
		masterReplId:     masterReplId,
		masterReplOffset: masterReplOffset,
	}
}
