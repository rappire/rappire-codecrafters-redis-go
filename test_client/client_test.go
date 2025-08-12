package main_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func sendAndReceive(t *testing.T, message string) string {
	conn, err := net.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		t.Fatalf("서버에 연결 실패: %v", err)
	}
	fmt.Println("서버에 연결 성공")
	defer conn.Close()

	if _, err := io.WriteString(conn, message); err != nil {
		t.Fatalf("데이터 전송 실패: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("응답 읽기 실패: %v", err)
	}

	return string(buf[:n])
}

func TestPing(t *testing.T) {
	fmt.Println("Ping 테스트")
	message := "*1\r\n$4\r\nPING\r\n"
	resp := sendAndReceive(t, message)

	expected := "+PONG\r\n"
	if resp != expected {
		t.Errorf("PING 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestEcho(t *testing.T) {
	fmt.Println("ECHO 테스트")
	message := "*2\r\n$4\r\nECHO\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)

	expected := "$5\r\ngrape\r\n"
	if resp != expected {
		t.Errorf("ECHO 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestSetAndGet(t *testing.T) {
	fmt.Println("Get Set 테스트")
	message := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$7\r\nmyvalue\r\n"
	resp := sendAndReceive(t, message)

	expected := "+OK\r\n"
	if resp != expected {
		t.Errorf("SET 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}

	message = "*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"
	resp = sendAndReceive(t, message)

	expected = "$7\r\nmyvalue\r\n"
	if resp != expected {
		t.Errorf("GET 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestExpire(t *testing.T) {
	fmt.Println("Set Expire 테스트")
	message := "*5\r\n$3\r\nSET\r\n$10\r\nstrawberry\r\n$5\r\ngrape\r\n$2\r\nPX\r\n$3\r\n100\r\n"
	resp := sendAndReceive(t, message)
	expected := "+OK\r\n"
	if resp != expected {
		t.Errorf("SET 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
	time.Sleep(150 * time.Millisecond)

	message = "*2\r\n$3\r\nGET\r\n$10\r\nstrawberry\r\n"
	resp = sendAndReceive(t, message)
	expected = "$-1\r\n"
	if resp != expected {
		t.Errorf("Expire이 되지 않음")
	}
}

func TestRPush(t *testing.T) {
	fmt.Println("RPush 테스트")
	message := "*3\r\n$5\r\nRPUSH\r\n$9\r\nRPushTest\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestRPushDouble(t *testing.T) {
	fmt.Println("RPush 두번 테스트")
	message := "*3\r\n$5\r\nRPUSH\r\n$11\r\nRPushDouble\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}

	message = "*3\r\n$5\r\nRPUSH\r\n$11\r\nRPushDouble\r\n$10\r\nstrawberry\r\n"
	resp = sendAndReceive(t, message)
	expected = ":2\r\n"
	if resp != expected {
		t.Errorf("2번째 RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}

func TestRPushTriple(t *testing.T) {
	fmt.Println("RPush 세번 테스트")
	message := "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$9\r\nraspberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}

	message = "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$9\r\npineapple\r\n"
	resp = sendAndReceive(t, message)
	expected = ":2\r\n"
	if resp != expected {
		t.Errorf("2번째 RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}

	message = "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$5\r\nmango\r\n"
	resp = sendAndReceive(t, message)
	expected = ":3\r\n"
	if resp != expected {
		t.Errorf("3번째 RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
	}
}
