package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

// ParseResult는 파싱 결과를 나타냅니다
type ParseResult struct {
	Valid bool
	Error []byte
}

// Success는 성공적인 파싱 결과를 반환합니다
func Success() ParseResult {
	return ParseResult{Valid: true}
}

// ValidationError는 검증 실패 결과를 반환합니다
func ValidationError(command, message string) ParseResult {
	return ParseResult{
		Valid: false,
		Error: protocol.AppendError([]byte{}, fmt.Sprintf("ERR %s", message)),
	}
}

// ArgumentError는 인수 개수 오류를 반환합니다
func ArgumentError(command string) ParseResult {
	return ValidationError(command, fmt.Sprintf("wrong number of arguments for '%s' command", strings.ToLower(command)))
}

// Parser는 명령어 파싱을 도와주는 구조체입니다
type Parser struct {
	event   types.CommandEvent
	command string
}

// NewParser는 새로운 Parser를 생성합니다
func NewParser(event types.CommandEvent) *Parser {
	return &Parser{
		event:   event,
		command: strings.ToUpper(event.Command),
	}
}

// RequireExactArgs는 정확한 인수 개수를 요구합니다
func (p *Parser) RequireExactArgs(count int) ParseResult {
	if len(p.event.Args) != count {
		return ArgumentError(p.command)
	}
	return Success()
}

// RequireMinArgs는 최소 인수 개수를 요구합니다
func (p *Parser) RequireMinArgs(count int) ParseResult {
	if len(p.event.Args) < count {
		return ArgumentError(p.command)
	}
	return Success()
}

// RequireArgsInRange는 인수 개수 범위를 요구합니다
func (p *Parser) RequireArgsInRange(min, max int) ParseResult {
	argc := len(p.event.Args)
	if argc < min || argc > max {
		return ArgumentError(p.command)
	}
	return Success()
}

// GetKey는 첫 번째 인수를 키로 반환합니다
func (p *Parser) GetKey() string {
	if len(p.event.Args) == 0 {
		return ""
	}
	return string(p.event.Args[0])
}

// GetString은 지정된 인덱스의 인수를 문자열로 반환합니다
func (p *Parser) GetString(index int) string {
	if index >= len(p.event.Args) {
		return ""
	}
	return string(p.event.Args[index])
}

// GetInt는 지정된 인덱스의 인수를 정수로 파싱합니다
func (p *Parser) GetInt(index int) (int, ParseResult) {
	if index >= len(p.event.Args) {
		return 0, ValidationError(p.command, "missing integer argument")
	}

	value, err := strconv.Atoi(string(p.event.Args[index]))
	if err != nil {
		return 0, ValidationError(p.command, "value is not an integer or out of range")
	}

	return value, Success()
}

// GetFloat는 지정된 인덱스의 인수를 실수로 파싱합니다
func (p *Parser) GetFloat(index int) (float64, ParseResult) {
	if index >= len(p.event.Args) {
		return 0, ValidationError(p.command, "missing float argument")
	}

	value, err := strconv.ParseFloat(string(p.event.Args[index]), 64)
	if err != nil {
		return 0, ValidationError(p.command, "value is not a valid float")
	}

	return value, Success()
}

// GetBytes는 지정된 인덱스의 인수를 바이트 슬라이스로 반환합니다
func (p *Parser) GetBytes(index int) []byte {
	if index >= len(p.event.Args) {
		return nil
	}
	return p.event.Args[index]
}

// GetBytesSlice는 지정된 시작 인덱스부터의 모든 인수를 바이트 슬라이스 배열로 반환합니다
func (p *Parser) GetBytesSlice(startIndex int) [][]byte {
	if startIndex >= len(p.event.Args) {
		return nil
	}
	return p.event.Args[startIndex:]
}

// HasOptionAt는 지정된 위치에 특정 옵션이 있는지 확인합니다
func (p *Parser) HasOptionAt(index int, option string) bool {
	if index >= len(p.event.Args) {
		return false
	}
	return strings.ToUpper(string(p.event.Args[index])) == strings.ToUpper(option)
}

// FindOption은 특정 옵션의 인덱스를 찾습니다
func (p *Parser) FindOption(option string) int {
	option = strings.ToUpper(option)
	for i, arg := range p.event.Args {
		if strings.ToUpper(string(arg)) == option {
			return i
		}
	}
	return -1
}

// ValidateEvenFieldValuePairs는 필드-값 쌍이 짝수 개인지 확인합니다
func (p *Parser) ValidateEvenFieldValuePairs(startIndex int) ParseResult {
	remaining := len(p.event.Args) - startIndex
	if remaining%2 != 0 {
		return ValidationError(p.command, "wrong number of field-value pairs")
	}
	return Success()
}

// GetFieldValuePairs는 필드-값 쌍들을 맵으로 반환합니다
func (p *Parser) GetFieldValuePairs(startIndex int) map[string]string {
	pairs := make(map[string]string)

	for i := startIndex; i < len(p.event.Args)-1; i += 2 {
		key := string(p.event.Args[i])
		value := string(p.event.Args[i+1])
		pairs[key] = value
	}

	return pairs
}
