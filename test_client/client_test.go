package test_client

import (
	"fmt"
	"io"
	"net"
	"sync"
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
		return
	}
}

func TestEcho(t *testing.T) {
	fmt.Println("ECHO 테스트")
	message := "*2\r\n$4\r\nECHO\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)

	expected := "$5\r\ngrape\r\n"
	if resp != expected {
		t.Errorf("ECHO 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestSetAndGet(t *testing.T) {
	fmt.Println("Get Set 테스트")
	message := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$7\r\nmyvalue\r\n"
	resp := sendAndReceive(t, message)

	expected := "+OK\r\n"
	if resp != expected {
		t.Errorf("SET 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"
	resp = sendAndReceive(t, message)

	expected = "$7\r\nmyvalue\r\n"
	if resp != expected {
		t.Errorf("GET 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestExpire(t *testing.T) {
	fmt.Println("Set Expire 테스트")
	message := "*5\r\n$3\r\nSET\r\n$10\r\nstrawberry\r\n$5\r\ngrape\r\n$2\r\nPX\r\n$3\r\n100\r\n"
	resp := sendAndReceive(t, message)
	expected := "+OK\r\n"
	if resp != expected {
		t.Errorf("SET 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
	time.Sleep(150 * time.Millisecond)

	message = "*2\r\n$3\r\nGET\r\n$10\r\nstrawberry\r\n"
	resp = sendAndReceive(t, message)
	expected = "$-1\r\n"
	if resp != expected {
		t.Errorf("Expire이 되지 않음")
		return
	}
}

func TestRPush(t *testing.T) {
	fmt.Println("RPush 테스트")
	message := "*3\r\n$5\r\nRPUSH\r\n$9\r\nRPushTest\r\n$5\r\ngrape\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestRPushMultiple(t *testing.T) {
	fmt.Println("RPush 같은 키 여러번 테스트")
	message := "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$9\r\nraspberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$9\r\npineapple\r\n"
	resp = sendAndReceive(t, message)
	expected = ":2\r\n"
	if resp != expected {
		t.Errorf("2번째 RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*3\r\n$5\r\nRPUSH\r\n$10\r\nstrawberry\r\n$5\r\nmango\r\n"
	resp = sendAndReceive(t, message)
	expected = ":3\r\n"
	if resp != expected {
		t.Errorf("3번째 RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestRPushMultipleArgs(t *testing.T) {
	fmt.Println("RPush 인자 여러개 테스트")
	message := "*5\r\n$5\r\nRPUSH\r\n$13\r\nRPushMultiple\r\n$5\r\ngrape\r\n$10\r\nstrawberry\r\n$5\nmango\r\n"
	resp := sendAndReceive(t, message)
	expected := ":3\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestLRange(t *testing.T) {
	fmt.Println("LRange 테스트")
	message := "*5\r\n$5\r\nRPUSH\r\n$6\r\nLRange\r\n$1\r\na\r\n$1\r\nb\r\n$1\nc\r\n"
	resp := sendAndReceive(t, message)
	expected := ":3\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*4\r\n$6\r\nLRANGE\r\n$6\r\nLRange\r\n:0\r\n:2\r\n"
	resp = sendAndReceive(t, message)
	expected = "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"
	if resp != expected {
		t.Errorf("LRange 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestLRangeMinusRange(t *testing.T) {
	fmt.Println("LRange 테스트")
	message := "*7\r\n$5\r\nRPUSH\r\n$11\r\nLRangeMinus\r\n$1\r\na\r\n$1\r\nb\r\n$1\nc\r\n$1\nd\r\n$1\ne\r\n"
	resp := sendAndReceive(t, message)
	expected := ":5\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*4\r\n$6\r\nLRANGE\r\n$11\r\nLRangeMinus\r\n:-2\r\n:-1\r\n"
	resp = sendAndReceive(t, message)
	expected = "*2\r\n$1\r\nd\r\n$1\r\ne\r\n"
	if resp != expected {
		t.Errorf("LRange 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestLRangeMinusRangeOver(t *testing.T) {
	fmt.Println("LRange 테스트")
	message := "*9\r\n$5\r\nRPUSH\r\n$5\r\ngrape\r\n$5\r\napple\r\n$9\r\npineapple\r\n$4\r\npear\r\n$5\r\nmango\r\n$9\r\nraspberry\r\n$6\r\nbanana\r\n$10\r\nstrawberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":7\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*4\r\n$6\r\nLRANGE\r\n$5\r\ngrape\r\n$2\r\n-8\r\n$2\r\n-1\r\n"
	resp = sendAndReceive(t, message)
	expected = "*7\r\n$5\r\napple\r\n$9\r\npineapple\r\n$4\r\npear\r\n$5\r\nmango\r\n$9\r\nraspberry\r\n$6\r\nbanana\r\n$10\r\nstrawberry\r\n"
	if resp != expected {
		t.Errorf("LRange 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestLPush(t *testing.T) {
	fmt.Println("LPush 테스트")
	message := "*3\r\n$5\r\nLPUSH\r\n$6\r\nbanana\r\n$9\r\nblueberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("LPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}

	message = "*4\r\n$5\r\nLPUSH\r\n$6\r\nbanana\r\n$9\r\nraspberry\r\n$5\r\nmango\r\n"
	resp = sendAndReceive(t, message)
	expected = ":3\r\n"
	if resp != expected {
		t.Errorf("LPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func blPush(t *testing.T) {
	fmt.Println("BLPUSH")
	message := "*3\r\n$5\r\nBLPOP\r\n$9\r\npineapple\r\n$1\r\n0\r\n"
	resp := sendAndReceive(t, message)
	expected := "*2\r\n$9\r\npineapple\r\n$10\r\nstrawberry\r\n"
	if resp != expected {
		t.Errorf("BLPUSH 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func rPush(t *testing.T) {
	fmt.Println("RPush")
	message := "*3\r\n$5\r\nRPUSH\r\n$9\r\npineapple\r\n$10\r\nstrawberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestBLPush(t *testing.T) {
	fmt.Println("BLPUSH 테스트")
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		blPush(t)
	}()
	time.Sleep(1 * time.Second)
	fmt.Println("After Sleep")
	go func() {
		defer wg.Done()
		rPush(t)
	}()

	wg.Wait()
	fmt.Println("Finish")
}

func blPush2(t *testing.T) {
	fmt.Println("BLPUSH")
	message := "*3\r\n$5\r\nBLPOP\r\n$5\r\ngrape\r\n$3\r\n0.2\r\n"
	resp := sendAndReceive(t, message)
	expected := "*2\r\n$9\r\npineapple\r\n$10\r\nstrawberry\r\n"
	if resp != expected {
		t.Errorf("BLPUSH 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func rPush2(t *testing.T) {
	fmt.Println("RPush")
	message := "*3\r\n$5\r\nRPUSH\r\n$9\r\npineapple\r\n$10\r\nstrawberry\r\n"
	resp := sendAndReceive(t, message)
	expected := ":1\r\n"
	if resp != expected {
		t.Errorf("RPush 응답이 잘못됨. got=%q, want=%q", resp, expected)
		return
	}
}

func TestBLPush2(t *testing.T) {
	fmt.Println("BLPUSH 테스트")
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		blPush2(t)
	}()
	time.Sleep(1 * time.Second)
	//fmt.Println("After Sleep")
	//go func() {
	//	defer wg.Done()
	//	rPush2(t)
	//}()

	wg.Wait()
	fmt.Println("Finish")
}
