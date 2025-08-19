package entity

import (
	"fmt"
	"strconv"
	"strings"
)

type StreamId struct {
	Millis int
	Seq    int
}

type StreamEntry struct {
	Id     *StreamId
	Fields map[string]string
}

type StreamEntity struct {
	Entries    []StreamEntry
	LastMillis int
	LastSeq    int
}

func (s *StreamEntity) Expired() bool {
	return false
}

func NewStreamEntity() *StreamEntity {
	return &StreamEntity{LastMillis: 0, LastSeq: 0, Entries: make([]StreamEntry, 10)}
}

func parseStreamId(id string) (*StreamId, error) {
	if id == "*" {
		return &StreamId{Millis: -1, Seq: -1}, nil
	}

	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return &StreamId{}, fmt.Errorf("invalid id")
	}

	t, err := strconv.Atoi(parts[0])
	if err != nil {
		return &StreamId{}, err
	}

	if parts[1] == "*" {
		return &StreamId{Millis: t, Seq: -1}, nil
	}
	sq, err := strconv.Atoi(parts[1])
	if err != nil {
		return &StreamId{}, err
	}
	if t <= 0 && sq <= 0 {
		return &StreamId{}, fmt.Errorf("ERR The ID specified in XADD must be greater than 0-0")
	}

	return &StreamId{Millis: t, Seq: sq}, nil
}

func (s *StreamEntity) GenerateId(requestedId string) (*StreamId, error) {
	id, err := parseStreamId(requestedId)

	if err != nil {
		return nil, err
	}

	fmt.Println(s.LastMillis, s.LastSeq)
	fmt.Println(id.Millis, id.Seq)

	if id.Millis == -1 {
		id.Millis = s.LastMillis
		id.Seq = s.LastSeq + 1
		s.LastMillis = id.Millis
		s.LastSeq = id.Seq
		return id, nil
	}

	if id.Seq == -1 {
		if id.Millis < s.LastMillis {
			return nil, fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}

		if id.Millis == s.LastMillis {
			id.Seq = s.LastSeq + 1
			s.LastMillis = id.Millis
			s.LastSeq = id.Seq
			return id, nil
		}

		id.Seq = 0
		s.LastMillis = id.Millis
		s.LastSeq = id.Seq
		return id, nil
	}

	if id.Millis < s.LastMillis {
		return nil, fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	} else if id.Millis == s.LastMillis && id.Seq <= s.LastSeq {
		return nil, fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}

	s.LastMillis = id.Millis
	s.LastSeq = id.Seq
	return id, nil
}
