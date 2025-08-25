package types

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type ServerInfo struct {
	role                       string
	ServerPort                 int
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

func (s *ServerInfo) GetMasterAddress() string {
	return s.masterServerIp + ":" + strconv.Itoa(s.masterServerPort)
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

func (s *ServerInfo) IsSlave() bool {
	return s.role == "slave"
}

func (s *ServerInfo) GetReplId() string {
	return s.masterReplId
}

func NewServerInfo(serverPort int, masterServerInfo string) *ServerInfo {
	if serverPort == 0 {
		serverPort = 6379
	}

	if masterServerInfo == "" {
		return &ServerInfo{
			role:             "master",
			masterReplId:     createMasterReplId(),
			masterReplOffset: 0,
			ServerPort:       serverPort,
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
		ServerPort:       serverPort,
	}
}
