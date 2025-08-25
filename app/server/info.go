package server

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
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
	masterServerIp             string
	masterServerPort           int
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

func NewServerInfo(masterServerInfo string) *ServerInfo {

	if masterServerInfo == "" {
		return &ServerInfo{
			role:             "master",
			masterReplId:     createMasterReplId(),
			masterReplOffset: 0,
		}
	}

	split := strings.Split(masterServerInfo, " ")
	masterServerIp := split[0]
	masterServerPort, err := strconv.Atoi(split[1])
	// TODO 에러 처리
	if err != nil {
		return nil
	}

	return &ServerInfo{
		role:             "slave",
		masterServerIp:   masterServerIp,
		masterServerPort: masterServerPort,
	}
}
