package client

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

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
	if err != nil || receive != "+PONG\r\n" {
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
	if err != nil || receive != "+OK\r\n" {
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
	if err != nil || receive != "+OK\r\n" {
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
	_, err = c.sendAndReceive(msg)
	if err != nil {
		fmt.Println("handshake failed on PSYNC")
		if err != nil {
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}

// sendAndReceive는 연결의 reader를 재사용하여 RESP 단위로 읽습니다.
// - simple string -> "+...\\r\\n" 형태로 반환
// - error -> "-...\\r\\n"
// - bulk string -> "$<len>\\r\\n<data>\\r\\n"
// 특별히, master가 "+FULLRESYNC ..."를 보낼 경우 다음에 오는 RDB Bulk를 추가로 읽어버립니다.
func (c *Client) sendAndReceive(msg []byte) (string, error) {
	_, err := c.conn.Write(msg)
	if err != nil {
		return "", err
	}

	// RESP 단위를 하나 읽음
	resp, err := protocol.ReadRESP(c.reader)
	if err != nil {
		return "", err
	}

	// simple string
	if resp.Type == protocol.SimpleString {
		s := string(resp.Data)
		// master가 FULLRESYNC를 보내면 그 다음에 RDB Bulk가 온다 — 이 경우 추가로 읽어서 버린다.
		if strings.HasPrefix(s, "FULLRESYNC") {
			// 다음 RESP (RDB bulk) 읽기
			next, err := protocol.ReadRESP(c.reader)
			if err != nil {
				return "", err
			}
			// bulk일 것으로 기대. 무시(혹은 디버그용으로 길이 리턴)
			if next.Type == protocol.BulkString {
				// 반환은 FULLRESYNC 라인만으로 (테스트 비교가 필요하면 이 값을 사용)
				return "+" + s + "\r\n", nil
			}
			// bulk가 아닌 경우에도 그냥 반환
			return "+" + s + "\r\n", nil
		}
		return "+" + s + "\r\n", nil
	}

	// error
	if resp.Type == protocol.Error {
		return "-" + string(resp.Data) + "\r\n", nil
	}

	// bulk string
	if resp.Type == protocol.BulkString {
		// resp.Length, resp.Data 사용
		return "$" + strconv.Itoa(resp.Length) + "\r\n" + string(resp.Data) + "\r\n", nil
	}

	// array (간단 디버깅 출력)
	if resp.Type == protocol.Array {
		var sb strings.Builder
		for _, elem := range resp.Arr {
			sb.WriteString(string(elem.Data))
			sb.WriteByte(' ')
		}
		return sb.String(), nil
	}

	// 그 외: 그대로 데이터로 반환 시도
	return string(resp.Data), nil
}
