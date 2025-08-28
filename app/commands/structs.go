package commands

import (
	"fmt"
	"strings"
	"time"
)

// 기본 명령어 구조체들

type EchoArgs struct {
	Message []byte `redis:"message"`
}

func (args *EchoArgs) Validate() error {
	return nil
}

type TypeArgs struct {
	Key string `redis:"key"`
}

func (args *TypeArgs) Validate() error {
	return nil
}

// 문자열 명령어 구조체들

type GetArgs struct {
	Key string `redis:"key"`
}

func (args *GetArgs) Validate() error {
	return nil
}

type SetArgs struct {
	Key    string `redis:"key"`
	Value  string `redis:"value"`
	Option string `redis:"option,optional"`
	Expiry int    `redis:"expiry,optional"`
}

func (args *SetArgs) Validate() error {
	if args.Option != "" && strings.ToUpper(args.Option) != "PX" {
		return fmt.Errorf("unsupported SET option: %s", args.Option)
	}
	if strings.ToUpper(args.Option) == "PX" && args.Expiry <= 0 {
		return fmt.Errorf("PX expiry must be positive")
	}
	return nil
}

func (args *SetArgs) GetExpiration() *time.Time {
	if strings.ToUpper(args.Option) == "PX" && args.Expiry > 0 {
		exp := time.Now().Add(time.Duration(args.Expiry) * time.Millisecond)
		return &exp
	}
	return nil
}

// IncrArgs는 INCR 명령어의 인수입니다
type IncrArgs struct {
	Key string `redis:"key"`
}

func (args *IncrArgs) Validate() error {
	return nil
}

// 리스트 명령어 구조체들

// RPushArgs는 RPUSH 명령어의 인수입니다
type RPushArgs struct {
	Key    string   `redis:"key"`
	Values [][]byte `redis:"values,variadic"`
}

func (args *RPushArgs) Validate() error {
	if len(args.Values) == 0 {
		return fmt.Errorf("at least one value required")
	}
	return nil
}

// LPushArgs는 LPUSH 명령어의 인수입니다
type LPushArgs struct {
	Key    string   `redis:"key"`
	Values [][]byte `redis:"values,variadic"`
}

func (args *LPushArgs) Validate() error {
	if len(args.Values) == 0 {
		return fmt.Errorf("at least one value required")
	}
	return nil
}

// LRangeArgs는 LRANGE 명령어의 인수입니다
type LRangeArgs struct {
	Key   string `redis:"key"`
	Start int    `redis:"start"`
	End   int    `redis:"end"`
}

func (args *LRangeArgs) Validate() error {
	return nil
}

type LLenArgs struct {
	Key string `redis:"key"`
}

func (args *LLenArgs) Validate() error {
	return nil
}

type LPopArgs struct {
	Key   string `redis:"key"`
	Count int    `redis:"count,optional"`
}

func (args *LPopArgs) Validate() error {
	if args.Count <= 0 {
		args.Count = 1 // 기본값
	}
	return nil
}

type BLPopArgs struct {
	Key     string  `redis:"key"`
	Timeout float64 `redis:"timeout"`
}

func (args *BLPopArgs) Validate() error {
	if args.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	return nil
}

func (args *BLPopArgs) GetTimeoutDuration() time.Duration {
	if args.Timeout == 0 {
		return 0 // 무한 대기
	}
	return time.Duration(args.Timeout * float64(time.Second))
}

// 트랜잭션 명령어 구조체들

type MultiArgs struct{}

func (args *MultiArgs) Validate() error {
	return nil
}

type ExecArgs struct{}

func (args *ExecArgs) Validate() error {
	return nil
}

type DiscardArgs struct{}

func (args *DiscardArgs) Validate() error {
	return nil
}

// 스트림 명령어 구조체들

type XAddArgs struct {
	Key    string            `redis:"key"`
	ID     string            `redis:"id"`
	Fields map[string]string `redis:"fields,field_value_pairs"`
}

func (args *XAddArgs) Validate() error {
	if len(args.Fields) == 0 {
		return fmt.Errorf("at least one field-value pair required")
	}
	return nil
}

type XRangeArgs struct {
	Key   string `redis:"key"`
	Start string `redis:"start"`
	End   string `redis:"end"`
}

func (args *XRangeArgs) Validate() error {
	return nil
}

type XReadArgs struct {
	Block   bool     `redis:"block,xread_block"`
	Timeout int      `redis:"timeout,xread_timeout"`
	Keys    []string `redis:"keys,xread_keys"`
	IDs     []string `redis:"ids,xread_ids"`
}

func (args *XReadArgs) Validate() error {
	if len(args.Keys) != len(args.IDs) {
		return fmt.Errorf("number of keys must match number of IDs")
	}
	if len(args.Keys) == 0 {
		return fmt.Errorf("at least one stream required")
	}
	return nil
}

type InfoArgs struct {
	Type string `redis:"type"`
}

func (args *InfoArgs) Validate() error {
	return nil
}

type ReplConfArgs struct {
	Reps2 string `redis:"reps2"`
	Reps3 string `redis:"reps3"`
}

func (args *ReplConfArgs) Validate() error {
	return nil
}
