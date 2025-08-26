package client

import (
	"fmt"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

type Client struct {
	conn net.Conn
	info *types.ServerInfo
}

func NewClient(info *types.ServerInfo) (*Client, error) {
	conn, err := net.Dial("tcp", info.GetMasterAddress())
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, info: info}, nil
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}

func (c *Client) CloseClient() {
	_ = c.conn.Close()
}

func (c *Client) Init() error {
	fmt.Println("start handshake")
	msg := protocol.AppendArray([]byte{}, 1)
	msg = protocol.AppendBulkString(msg, []byte("PING"))
	receive, err := sendAndReceive(c.conn, msg)
	if err != nil || receive != "+PONG\r\n" {
		fmt.Println("handshake failed")
		fmt.Printf("PING 응답이 잘못됨. got=%q, want=%q", receive, "+PONG\r\n")
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("send replconf listening-port " + strconv.Itoa(c.info.ServerPort))
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("REPLCONF"))
	msg = protocol.AppendBulkString(msg, []byte("listening-port"))
	msg = protocol.AppendBulkString(msg, []byte(strconv.Itoa(c.info.ServerPort)))

	receive, err = sendAndReceive(c.conn, msg)
	if err != nil || receive != "+OK\r\n" {
		fmt.Println("handshake failed")
		fmt.Println(receive)
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("send replconf capa psync2")
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("REPLCONF"))
	msg = protocol.AppendBulkString(msg, []byte("capa"))
	msg = protocol.AppendBulkString(msg, []byte("psync2"))

	receive, err = sendAndReceive(c.conn, msg)
	if err != nil || receive != "+OK\r\n" {
		fmt.Println("handshake failed")
		fmt.Println(receive)
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("send psync")
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("PSYNC"))
	msg = protocol.AppendBulkString(msg, []byte("?"))
	msg = protocol.AppendBulkString(msg, []byte("-1"))

	receive, err = sendAndReceive(c.conn, msg)
	if err != nil {
		fmt.Println("handshake failed")
		fmt.Println(receive)
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func sendAndReceive(conn net.Conn, msg []byte) (string, error) {
	_, err := conn.Write(msg)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	return string(buf[:n]), nil
}
