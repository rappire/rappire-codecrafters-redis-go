package client

import (
	"fmt"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

type Client struct {
	conn net.Conn
	ip   string
	port int
}

func NewClient(ip string, port int) (*Client, error) {
	conn, err := net.Dial("tcp", ip+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, ip: ip, port: port}, nil
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

	fmt.Println("send replconf listening-port " + strconv.Itoa(c.port))
	msg = protocol.AppendArray([]byte{}, 3)
	msg = protocol.AppendBulkString(msg, []byte("REPLCONF"))
	msg = protocol.AppendBulkString(msg, []byte("listening-port"))
	msg = protocol.AppendBulkString(msg, []byte(strconv.Itoa(c.port)))

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
