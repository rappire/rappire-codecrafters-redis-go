package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	info   *types.ServerInfo
}

func NewClient(info *types.ServerInfo) (*Client, error) {
	conn, err := net.Dial("tcp", info.GetMasterAddress())
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		info:   info,
	}, nil
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}

func (c *Client) CloseClient() {
	_ = c.conn.Close()
}

func (c *Client) Init() error {
	fmt.Println("start handshake")

	// 1) PING
	msg := protocol.AppendArray([]byte{}, 1)
	msg = protocol.AppendBulkString(msg, []byte("PING"))
	receive, err := c.sendAndReceive(msg)
	if err != nil || receive.Raw == nil {
		fmt.Println("handshake failed")
		fmt.Printf("PING 응답이 잘못됨. got=%q, want=%q\n", receive, "+PONG\r\n")
		if err != nil {
			fmt.Println(err.Error())
		}
		return fmt.Errorf("ping failed")
	}

	// 2) REPLCONF listening-port
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("REPLCONF"))
	msg = protocol.AppendBulkString(msg, []byte("listening-port"))
	msg = protocol.AppendBulkString(msg, []byte(strconv.Itoa(c.info.ServerPort)))

	receive, err = c.sendAndReceive(msg)
	if err != nil || receive.Raw == nil {
		fmt.Println("handshake failed")
		fmt.Println("REPLCONF listening-port response:", receive)
		if err != nil {
			fmt.Println(err.Error())
		}
		return fmt.Errorf("replconf listening-port failed")
	}

	// 3) REPLCONF capa psync2
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("REPLCONF"))
	msg = protocol.AppendBulkString(msg, []byte("capa"))
	msg = protocol.AppendBulkString(msg, []byte("psync2"))

	receive, err = c.sendAndReceive(msg)
	if err != nil || receive.Raw == nil {
		fmt.Println("handshake failed")
		fmt.Println("REPLCONF capa response:", receive)
		if err != nil {
			fmt.Println(err.Error())
		}
		return fmt.Errorf("replconf capa failed")
	}

	// 4) PSYNC ? -1
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("PSYNC"))
	msg = protocol.AppendBulkString(msg, []byte("?"))
	msg = protocol.AppendBulkString(msg, []byte("-1"))

	// 여기서는 master가 +FULLRESYNC ... (SimpleString) 과 이어서 RDB Bulk를 보냄.
	// sendAndReceive는 이를 올바르게 읽고 에러 없이 반환해야 함.
	go func() {
		time.Sleep(time.Millisecond * 200)
		c.conn.Write(msg)
		rdb, err := ReadRDB(c.reader)
		fmt.Println(string(rdb))
		if err != nil {
			fmt.Println("handshake failed on PSYNC")
			fmt.Println(err.Error())
		}
	}()
	return nil
}

func (c *Client) sendAndReceive(msg []byte) (protocol.Resp, error) {
	_, err := c.conn.Write(msg)
	if err != nil {
		return protocol.Resp{}, err
	}

	// RESP 단위를 하나 읽음
	resp, err := protocol.ReadRESP(c.reader)
	if err != nil {
		return protocol.Resp{}, err
	}
	return resp, nil
}

func ReadRDB(reader *bufio.Reader) ([]byte, error) {
	// 1. 첫 줄 읽기: "$<len>\r\n"
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")

	if len(line) == 0 || line[0] != '$' {
		return nil, fmt.Errorf("invalid RDB bulk string header: %q", line)
	}

	// 2. 길이 파싱
	length, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid bulk string length: %v", err)
	}
	if length < 0 {
		return nil, fmt.Errorf("null bulk string (no RDB)")
	}

	// 3. 본문 + CRLF 읽기
	buf := make([]byte, length+2)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}

	// 4. 끝에 CRLF 체크
	if buf[length] != '\r' || buf[length+1] != '\n' {
		return nil, fmt.Errorf("invalid RDB format, missing CRLF")
	}

	// 5. RDB payload만 반환
	return buf[:length], nil
}
