package test_client

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

func TestMain(m *testing.M) {
	// 테스트 전체에서 1회만 실행됨
	fmt.Println("🔌 Redis 연결 초기화...")
	rdb = redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           0,
		DialTimeout:  0, // 연결 무제한 대기
		ReadTimeout:  0, // 읽기 무제한 대기
		WriteTimeout: 0, // 쓰기 무제한 대기
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Println("Redis 연결 실패:", err)
		os.Exit(1)
	}

	// 테스트 실행
	code := m.Run()

	// 종료 시 리소스 정리
	fmt.Println("🔌 Redis 연결 종료...")
	_ = rdb.Close()

	os.Exit(code)
}

func TestRedisSet(t *testing.T) {
	err := rdb.Set(ctx, "foo", "bar", 0).Err()
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := rdb.Get(ctx, "foo").Result()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if val != "bar" {
		t.Fatalf("expected bar, got %s", val)
	}
}

func TestRedisType(t *testing.T) {
	err := rdb.Set(ctx, "foo2", "bar", 0).Err()
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	cmd := rdb.Type(ctx, "foo2")
	if cmd.Err() != nil {
		t.Fatalf("Type failed: %v", cmd.Err())
	}
	if cmd.Val() != "string" {
		t.Fatalf("expected string got %s", cmd.Val())
	}
}

func TestXAdd(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test",
		ID:     "0-1",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	fmt.Println(add.Val())

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	if add.Val() != "0-1" {
		t.Fatalf("expected 0-1, got %s", add.Val())
	}
}

func TestXAddType(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test2",
		ID:     "1-1",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	if add.Val() != "1-1" {
		t.Fatalf("expected 1-1, got %s", add.Val())
	}

	cmd := rdb.Type(ctx, "test2")
	if cmd.Err() != nil {
		t.Fatalf("XAdd failed: %v", cmd.Err())
	}

	if cmd.Val() != "stream" {
		t.Fatalf("expected stream, got %s", cmd.Val())
	}
}

func TestXAddIdFail(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test4",
		ID:     "0-1",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	if add.Val() != "0-1" {
		t.Fatalf("expected 0-1, got %s", add.Val())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test4",
		ID:     "0-1",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() == nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	fmt.Println(add.Err())
}

func TestXAddStarSeq(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test9",
		ID:     "0-*",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	if add.Val() != "0-1" {
		t.Fatalf("expected 0-1, got %s", add.Val())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test9",
		ID:     "0-*",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	if add.Val() != "0-2" {
		t.Fatalf("expected 0-2, got %s", add.Val())
	}
}

func TestXAddStar(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "test10",
		ID:     "*",
		Values: map[string]interface{}{
			"foo": "bar",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}
	fmt.Println(add.Val())
}

func TestXRange(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xrange",
		ID:     "0-1",
		Values: map[string]interface{}{
			"foo1": "bar1",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xrange",
		ID:     "0-2",
		Values: map[string]interface{}{
			"foo2": "bar2",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xrange",
		ID:     "0-3",
		Values: map[string]interface{}{
			"foo3": "bar3",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	xRange := rdb.XRange(ctx, "xrange", "0-2", "0-3")
	fmt.Println(xRange.Val())
}

func TestXRead(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xread",
		ID:     "0-1",
		Values: map[string]interface{}{
			"foo1": "bar1",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xread",
		ID:     "0-2",
		Values: map[string]interface{}{
			"foo2": "bar2",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "xread",
		ID:     "0-3",
		Values: map[string]interface{}{
			"foo3": "bar3",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}
	read := rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"xread"},
		ID:      "0-1",
	})

	fmt.Println(read.Val())
}

func TestXRead2(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "banana",
		ID:     "0-1",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	read := rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"banana"},
		ID:      "0-0",
	})

	fmt.Println(read.Val())
}

func TestXReadBlock(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block",
		ID:     "0-1",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	go func() {
		read := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"block"},
			ID:      "0-1",
			Block:   0,
		})
		fmt.Println(read.Val())
	}()

	time.Sleep(100 * time.Millisecond)

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block",
		ID:     "0-2",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}
}

func TestXReadBlock2(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block2",
		ID:     "0-1",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	go func() {
		read := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"block2"},
			ID:      "$",
			Block:   0,
		})
		fmt.Println(read.Val())
	}()

	time.Sleep(100 * time.Millisecond)

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block2",
		ID:     "0-2",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}
}

func TestXRead3(t *testing.T) {
	add := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block3",
		ID:     "0-1",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}

	go func() {
		read := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"block3"},
			ID:      "$",
			Block:   0,
		})
		fmt.Println(read.Val())
	}()

	time.Sleep(1000 * time.Millisecond)

	add = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "block3",
		ID:     "0-2",
		Values: map[string]interface{}{
			"temperature": "93",
		},
	})

	if add.Err() != nil {
		t.Fatalf("XAdd failed: %v", add.Err())
	}
}
