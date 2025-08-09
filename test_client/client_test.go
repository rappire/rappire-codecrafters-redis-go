package main_test

import (
	"fmt"
	"io"
	"net"
	"testing"
)

func sendAndReceive(t *testing.T, message string) string {
	conn, err := net.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		t.Fatalf("서버에 연결 실패: %v", err)
	}
	fmt.Println("서버에 연결 성공")
	defer conn.Close()

	// 문자열 전송 (RESP 형식)
	if _, err := io.WriteString(conn, message); err != nil {
		t.Fatalf("데이터 전송 실패: %v", err)
	}

	// 응답 읽기
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("응답 읽기 실패: %v", err)
	}

	return string(buf[:n])
}

func TestPing(t *testing.T) {
	// PING 명령 RESP 형식: ["PING"]
	fmt.Println("Ping 테스트")
	message := "*1\r\n$4\r\nPING\r\n"
	resp := sendAndReceive(t, message)

	expected := "+PONG\r\n"
	if resp != expected {
		t.Errorf("PING 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestEcho(t *testing.T) {
	// ECHO grape 명령 RESP 형식: ["ECHO", "grape"]
	fmt.Println("ECHO 테스트")
	message := "*2\r\n$4\r\nECHO\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)

	// Bulk String: "$5\r\ngrape\r\n"
	expected := "$5\r\ngrape\r\n"
	if resp != expected {
		t.Errorf("ECHO 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}
