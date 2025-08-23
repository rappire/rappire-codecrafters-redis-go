package commands

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/types"
)

type CommandArgs interface {
	Validate() error
}

type ArgParser struct {
	event types.CommandEvent
}

func NewArgParser(event types.CommandEvent) *ArgParser {
	return &ArgParser{event: event}
}

func (ap *ArgParser) Parse(args CommandArgs) ParseResult {
	v := reflect.ValueOf(args)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return ValidationError(ap.event.Command, "args must be a pointer to struct")
	}

	v = v.Elem()
	t := v.Type()

	argIndex := 0

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 필드가 설정 가능한지 확인
		if !fieldValue.CanSet() {
			continue
		}

		tag := field.Tag.Get("redis")
		if tag == "-" {
			continue
		}

		// 태그 파싱
		tagParts := strings.Split(tag, ",")
		fieldName := tagParts[0]
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		// 옵션 파싱
		required := true
		variadic := false
		var specialParser string

		for _, option := range tagParts[1:] {
			switch option {
			case "optional":
				required = false
			case "variadic":
				variadic = true
			case "field_value_pairs":
				specialParser = "field_value_pairs"
			case "keyword_option":
				specialParser = "keyword_option"
			case "xread_block":
				specialParser = "xread_block"
			case "xread_timeout":
				specialParser = "xread_timeout"
			case "xread_keys":
				specialParser = "xread_keys"
			case "xread_ids":
				specialParser = "xread_ids"
			}
		}

		if specialParser != "" {
			newArgIndex, err := ap.handleSpecialParser(specialParser, fieldValue, argIndex, v)
			if err != nil {
				return ValidationError(ap.event.Command, err.Error())
			}
			argIndex = newArgIndex
			continue
		}

		// variadic 필드 처리 (나머지 모든 인수)
		if variadic {
			if fieldValue.Kind() == reflect.Slice {
				remaining := len(ap.event.Args) - argIndex
				if remaining < 0 {
					remaining = 0
				}

				slice := reflect.MakeSlice(fieldValue.Type(), remaining, remaining)
				for j := 0; j < remaining; j++ {
					elem := slice.Index(j)
					if err := ap.setFieldValue(elem, ap.event.Args[argIndex+j], field.Name); err != nil {
						return ValidationError(ap.event.Command, err.Error())
					}
				}
				fieldValue.Set(slice)
				argIndex += remaining
			}
			continue
		}

		// 일반 필드 처리
		if argIndex >= len(ap.event.Args) {
			if required {
				return ArgumentError(ap.event.Command)
			}
			continue
		}

		if err := ap.setFieldValue(fieldValue, ap.event.Args[argIndex], field.Name); err != nil {
			return ValidationError(ap.event.Command, err.Error())
		}

		argIndex++
	}

	// 남은 인수가 있는지 확인
	if argIndex < len(ap.event.Args) {
		return ArgumentError(ap.event.Command)
	}

	// 커스텀 검증 실행
	if err := args.Validate(); err != nil {
		return ValidationError(ap.event.Command, err.Error())
	}

	return Success()
}

func (ap *ArgParser) hasSpecialParsers(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("redis")
		if strings.Contains(tag, "variadic") ||
			strings.Contains(tag, "field_value_pairs") ||
			strings.Contains(tag, "keyword_option") ||
			strings.Contains(tag, "xread_") {
			return true
		}
	}
	return false
}

func (ap *ArgParser) handleSpecialParser(parserType string, fieldValue reflect.Value, argIndex int, structValue reflect.Value) (int, error) {
	switch parserType {
	case "field_value_pairs":
		return ap.parseFieldValuePairs(fieldValue, argIndex)
	case "keyword_option":
		return ap.parseKeywordOption(fieldValue, argIndex, structValue)
	case "xread_block":
		return ap.parseXReadBlock(fieldValue, argIndex)
	case "xread_timeout":
		return ap.parseXReadTimeout(fieldValue, argIndex, structValue)
	case "xread_keys":
		return ap.parseXReadKeys(fieldValue, argIndex)
	case "xread_ids":
		return ap.parseXReadIDs(fieldValue, argIndex)
	default:
		return argIndex, fmt.Errorf("unknown special parser: %s", parserType)
	}
}

func (ap *ArgParser) parseFieldValuePairs(fieldValue reflect.Value, argIndex int) (int, error) {
	remaining := len(ap.event.Args) - argIndex
	if remaining%2 != 0 {
		return remaining, fmt.Errorf("wrong number of field-value pairs: %d", remaining)
	}

	fields := make(map[string]string)
	for i := argIndex; i < len(ap.event.Args); i += 2 {
		key := string(ap.event.Args[i])
		value := string(ap.event.Args[i+1])
		fields[key] = value
	}
	fieldValue.Set(reflect.ValueOf(fields))
	return len(ap.event.Args), nil
}

func (ap *ArgParser) parseKeywordOption(fieldValue reflect.Value, argIndex int, structValue reflect.Value) (int, error) {
	if argIndex >= len(ap.event.Args) {
		return argIndex, nil // 옵션이 없음
	}

	option := strings.ToUpper(string(ap.event.Args[argIndex]))

	switch option {
	case "PX":
		fieldValue.SetString(option)
		argIndex++

		// 다음 인수가 있어야 함 (expiry time)
		if argIndex >= len(ap.event.Args) {
			return argIndex, fmt.Errorf("missing expiry time for PX option")
		}

		expiry, err := strconv.Atoi(string(ap.event.Args[argIndex]))
		if err != nil {
			return argIndex, fmt.Errorf("invalid expiry time")
		}

		if expiryField := structValue.FieldByName("Expiry"); expiryField.IsValid() && expiryField.CanSet() {
			expiryField.SetInt(int64(expiry))
		}
		argIndex++
	default:
		// 알려지지 않은 옵션은 무시
		return argIndex, nil
	}

	return argIndex, nil
}

// parseXAddFields는 XADD 명령어의 필드-값 쌍을 파싱합니다
func (ap *ArgParser) parseXAddFields(startIndex int) (map[string]string, error) {
	remaining := len(ap.event.Args) - startIndex
	if remaining%2 != 0 {
		return nil, fmt.Errorf("wrong number of field-value pairs")
	}

	fields := make(map[string]string)
	for i := startIndex; i < len(ap.event.Args); i += 2 {
		key := string(ap.event.Args[i])
		value := string(ap.event.Args[i+1])
		fields[key] = value
	}

	return fields, nil
}

// parseXReadArgs는 XREAD 명령어의 복잡한 구조를 파싱합니다
func (ap *ArgParser) parseXReadArgs() (*XReadArgs, error) {
	args := &XReadArgs{}
	argIndex := 0

	// BLOCK 옵션 확인
	if argIndex < len(ap.event.Args) && strings.ToUpper(string(ap.event.Args[argIndex])) == "BLOCK" {
		args.Block = true
		argIndex++

		if argIndex >= len(ap.event.Args) {
			return nil, fmt.Errorf("missing timeout for BLOCK")
		}

		timeout, err := strconv.Atoi(string(ap.event.Args[argIndex]))
		if err != nil || timeout < 0 {
			return nil, fmt.Errorf("invalid timeout")
		}
		args.Timeout = timeout
		argIndex++
	}

	// STREAMS 키워드 확인
	if argIndex >= len(ap.event.Args) || strings.ToUpper(string(ap.event.Args[argIndex])) != "STREAMS" {
		return nil, fmt.Errorf("expected STREAMS keyword")
	}
	argIndex++

	// 키와 ID 파싱
	remaining := len(ap.event.Args) - argIndex
	if remaining%2 != 0 || remaining == 0 {
		return nil, fmt.Errorf("invalid number of keys and IDs")
	}

	numStreams := remaining / 2
	args.Keys = make([]string, numStreams)
	args.IDs = make([]string, numStreams)

	// 키들 파싱
	for i := 0; i < numStreams; i++ {
		args.Keys[i] = string(ap.event.Args[argIndex+i])
	}

	// ID들 파싱
	for i := 0; i < numStreams; i++ {
		args.IDs[i] = string(ap.event.Args[argIndex+numStreams+i])
	}

	return args, nil
}

func (ap *ArgParser) setFieldValue(fieldValue reflect.Value, arg []byte, fieldName string) error {
	argStr := string(arg)

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(argStr)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(argStr, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be an integer", fieldName)
		}
		fieldValue.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(argStr, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be a positive integer", fieldName)
		}
		fieldValue.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(argStr, 64)
		if err != nil {
			return fmt.Errorf("%s must be a number", fieldName)
		}
		fieldValue.SetFloat(val)

	case reflect.Bool:
		val, err := strconv.ParseBool(argStr)
		if err != nil {
			return fmt.Errorf("%s must be true or false", fieldName)
		}
		fieldValue.SetBool(val)

	case reflect.Slice:
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 { // []byte
			fieldValue.SetBytes(arg)
		} else {
			return fmt.Errorf("unsupported slice type for field %s", fieldName)
		}

	default:
		return fmt.Errorf("unsupported field type %s for field %s", fieldValue.Kind(), fieldName)
	}

	return nil
}

type xreadState struct {
	argIndex     int
	hasBlock     bool
	timeout      int
	streamsIndex int
}

// getXReadState: XREAD 명령어의 전체 구조를 분석
func (ap *ArgParser) getXReadState() (*xreadState, error) {
	state := &xreadState{argIndex: 0}

	// BLOCK 옵션 확인
	if state.argIndex < len(ap.event.Args) && strings.ToUpper(string(ap.event.Args[state.argIndex])) == "BLOCK" {
		state.hasBlock = true
		state.argIndex++

		if state.argIndex >= len(ap.event.Args) {
			return nil, fmt.Errorf("missing timeout for BLOCK")
		}

		timeout, err := strconv.Atoi(string(ap.event.Args[state.argIndex]))
		if err != nil || timeout < 0 {
			return nil, fmt.Errorf("invalid timeout")
		}
		state.timeout = timeout
		state.argIndex++
	}

	// STREAMS 키워드 확인
	if state.argIndex >= len(ap.event.Args) || strings.ToUpper(string(ap.event.Args[state.argIndex])) != "STREAMS" {
		return nil, fmt.Errorf("expected STREAMS keyword")
	}
	state.streamsIndex = state.argIndex + 1

	// 키와 ID 개수 확인
	remaining := len(ap.event.Args) - state.streamsIndex
	if remaining%2 != 0 || remaining == 0 {
		return nil, fmt.Errorf("invalid number of keys and IDs")
	}

	return state, nil
}

// parseXReadBlock: BLOCK 옵션 파싱
func (ap *ArgParser) parseXReadBlock(fieldValue reflect.Value, argIndex int) (int, error) {
	state, err := ap.getXReadState()
	if err != nil {
		return argIndex, err
	}

	fieldValue.SetBool(state.hasBlock)
	return state.streamsIndex, nil
}

// parseXReadTimeout: TIMEOUT 값 파싱
func (ap *ArgParser) parseXReadTimeout(fieldValue reflect.Value, argIndex int, structValue reflect.Value) (int, error) {
	state, err := ap.getXReadState()
	if err != nil {
		return argIndex, err
	}

	fieldValue.SetInt(int64(state.timeout))
	return state.streamsIndex, nil
}

// parseXReadKeys: KEYS 배열 파싱
func (ap *ArgParser) parseXReadKeys(fieldValue reflect.Value, argIndex int) (int, error) {
	state, err := ap.getXReadState()
	if err != nil {
		return argIndex, err
	}

	remaining := len(ap.event.Args) - state.streamsIndex
	numStreams := remaining / 2

	keys := make([]string, numStreams)
	for i := 0; i < numStreams; i++ {
		keys[i] = string(ap.event.Args[state.streamsIndex+i])
	}

	fieldValue.Set(reflect.ValueOf(keys))
	return len(ap.event.Args), nil
}

// parseXReadIDs: IDs 배열 파싱
func (ap *ArgParser) parseXReadIDs(fieldValue reflect.Value, argIndex int) (int, error) {
	state, err := ap.getXReadState()
	if err != nil {
		return argIndex, err
	}

	remaining := len(ap.event.Args) - state.streamsIndex
	numStreams := remaining / 2

	ids := make([]string, numStreams)
	for i := 0; i < numStreams; i++ {
		ids[i] = string(ap.event.Args[state.streamsIndex+numStreams+i])
	}

	fieldValue.Set(reflect.ValueOf(ids))
	return len(ap.event.Args), nil
}

// 통합된 파싱과 실행 함수
func ParseAndExecute[T CommandArgs](event types.CommandEvent, handler func(T)) {
	var args T

	argsValue := reflect.New(reflect.TypeOf(args).Elem())
	args = argsValue.Interface().(T)

	parser := NewArgParser(event)
	if result := parser.Parse(args); !result.Valid {
		event.Ctx.Write(result.Error)
		return
	}

	handler(args)
}
