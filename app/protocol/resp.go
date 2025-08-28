package protocol

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
	Raw    []byte
}

func ReadRESP(reader *bufio.Reader) (Resp, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return Resp{}, err
	}

	raw := []byte(line)

	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return Resp{}, fmt.Errorf("empty line")
	}

	respType := RespType(line[0])
	payload := line[1:]

	switch respType {
	case SimpleString, Error, Integer:
		return Resp{Type: respType, Data: []byte(payload), Raw: raw}, nil

	case Array:
		length, err := strconv.Atoi(payload)
		if err != nil {
			return Resp{}, fmt.Errorf("invalid array length: %v", err)
		}
		if length < 0 {
			return Resp{Type: respType, Data: nil, Length: length, Raw: raw}, nil
		}

		arr := make([]Resp, 0, length)
		for i := 0; i < length; i++ {
			elem, err := ReadRESP(reader)
			if err != nil {
				return Resp{}, err
			}
			arr = append(arr, elem)
			raw = append(raw, elem.Raw...)
		}
		return Resp{Type: respType, Length: length, Arr: arr, Raw: raw}, nil

	case BulkString:
		length, err := strconv.Atoi(payload)
		if err != nil {
			return Resp{}, fmt.Errorf("invalid bulk string length: %v", err)
		}
		if length < 0 {
			return Resp{Type: respType, Data: nil}, nil
		}

		buf := make([]byte, length+2)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return Resp{}, err
		}
		if !strings.HasSuffix(string(buf), "\r\n") {
			return Resp{}, fmt.Errorf("invalid response format")
		}
		data := buf[:length]
		raw = append(raw, buf...)
		return Resp{Type: respType, Data: data, Raw: raw}, nil

	default:
		return Resp{}, fmt.Errorf("unknown RESP type: %c", respType)
	}
}

func AppendString(buf []byte, data string) []byte {
	buf = append(buf, '+')
	buf = append(buf, data...)
	return append(buf, '\r', '\n')
}

func AppendError(buf []byte, data string) []byte {
	buf = append(buf, '-')
	buf = append(buf, data...)
	return append(buf, '\r', '\n')
}

func AppendBulkString(buf []byte, bulk []byte) []byte {
	buf = appendPrefix(buf, '$', int64(len(bulk)))
	buf = append(buf, bulk...)
	return append(buf, '\r', '\n')
}

func AppendArray(buf []byte, n int) []byte {
	return appendPrefix(buf, '*', int64(n))
}

func AppendInt(buf []byte, n int) []byte {
	return appendPrefix(buf, ':', int64(n))
}

func AppendNilBulkString() []byte {
	return []byte("$-1\r\n")
}

func AppendNilArray() []byte {
	return []byte("*-1\r\n")
}

func appendPrefix(buf []byte, c byte, n int64) []byte {
	buf = append(buf, c)
	buf = strconv.AppendInt(buf, n, 10)
	return append(buf, '\r', '\n')
}
