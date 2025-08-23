package server

import "fmt"

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

func NewServerInfo(role string) *ServerInfo {
	if role == "" {
		role = "master"
	} else {
		role = "slave"
	}

	return &ServerInfo{
		role: role,
	}
}
