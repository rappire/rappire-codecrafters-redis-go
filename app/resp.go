package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type RespType byte

const (
	SimpleString RespType = '+'
	Error        RespType = '-'
	Integer      RespType = ':'
	BulkString   RespType = '$'
	Array        RespType = '*'
)

type Resp struct {
	Type   RespType
	Data   []byte
	Length int
	Arr    []Resp
}

func readRESP(reader *bufio.Reader) (Resp, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return Resp{}, err
	}

	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return Resp{}, fmt.Errorf("empty Line")
	}

	respType := RespType(line[0])
	payload := line[1:]

	switch respType {
	case SimpleString, Error, Integer:
		return Resp{Type: respType, Data: []byte(payload)}, nil
	case Array:
		length, err := strconv.Atoi(payload)
		if err != nil {
			return Resp{}, fmt.Errorf("invalid array length: %v", err)
		}
		if length < 0 {
			return Resp{Type: respType, Data: nil, Length: length}, nil
		}

		arr := make([]Resp, 0, length)
		for i := 0; i < length; i++ {
			elem, err := readRESP(reader)
			if err != nil {
				return Resp{}, err
			}
			arr = append(arr, elem)
		}
		return Resp{Type: respType, Length: length, Arr: arr}, nil

	case BulkString:
		length, err := strconv.Atoi(payload)
		if err != nil {
			return Resp{}, fmt.Errorf("invalid bulk string length: %v", err)
		}
		if length < 0 {
			return Resp{Type: respType, Data: nil}, nil
		}

		buf := make([]byte, length)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return Resp{}, err
		}
		if !strings.HasPrefix(string(buf), "\r\n") {
			return Resp{}, fmt.Errorf("invalid response format")
		}
		data := buf[:length]
		return Resp{Type: respType, Data: data}, nil
	default:
		return Resp{}, fmt.Errorf("unknown Resp type: %c", respType)
	}
}
