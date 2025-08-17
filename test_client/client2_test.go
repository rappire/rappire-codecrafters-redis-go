package main_test

import (
	"context"
	"fmt"
	"os"
	"testing"

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
		Addr: "localhost:6379",
		DB:   0,
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

func TestBLPOP(t *testing.T) {
	result := rdb.BLPop(ctx, 1, "testBLPOP")
	fmt.Println(result.Val())
}
