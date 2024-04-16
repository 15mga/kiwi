package util

import (
	"regexp"
	"runtime"
	"strconv"
)

const (
	EcMin TErrCode = iota + 60000
	EcNil
	EcRecover
	EcWrongType
	EcBusy
	EcNotEnough
	EcNoAuth
	EcTimeout
	EcClosed
	EcOpened
	EcEmpty
	EcBadHead
	EcBadPacket
	EcExist
	EcNotExist
	EcMarshallErr
	EcUnmarshallErr
	EcIllegalOp
	EcIllegalConn
	EcTooManyConn
	EcParamsErr
	EcParseErr
	EcIo
	EcOutOfRange
	EcTooLong
	EcTooMuch
	EcLengthErr
	EcNotExistAgent
	EcSendErr
	EcReceiveErr
	EcAcceptErr
	EcConnectErr
	EcListenErr
	EcAddrErr
	EcUnavailable
	EcNotImplement
	EcType
	EcWriteFail
	EcFail
	EcTooSlow
	EcServiceErr
	EcDbErr
	EcRedisErr
	EcDiscoveryErr
)

var (
	_ErrCodeToString = map[TErrCode]string{
		EcNil:           "object_nil",
		EcRecover:       "recover",
		EcWrongType:     "wrong_type",
		EcBusy:          "busy",
		EcNotEnough:     "not_enough",
		EcNoAuth:        "no_auth",
		EcTimeout:       "timeout",
		EcClosed:        "closed",
		EcOpened:        "opened",
		EcEmpty:         "empty",
		EcBadHead:       "bad_head",
		EcBadPacket:     "bad_packet",
		EcExist:         "exist",
		EcNotExist:      "not_exist",
		EcMarshallErr:   "marshall_error",
		EcUnmarshallErr: "unmarshall_error",
		EcIllegalOp:     "illegal_operation",
		EcIllegalConn:   "illegal_conn",
		EcTooManyConn:   "too_many_conn",
		EcParamsErr:     "args_error",
		EcParseErr:      "parse_error",
		EcIo:            "io_error",
		EcOutOfRange:    "out_of_range",
		EcTooLong:       "too_long",
		EcTooMuch:       "too_much",
		EcLengthErr:     "length_wrong",
		EcNotExistAgent: "not_exist_agent",
		EcSendErr:       "send_error",
		EcReceiveErr:    "receive_error",
		EcAcceptErr:     "accept_error",
		EcConnectErr:    "connect_error",
		EcListenErr:     "listen_error",
		EcServiceErr:    "service_error",
		EcDbErr:         "database_error",
		EcRedisErr:      "redis_wrong",
		EcDiscoveryErr:  "discovery_wrong",
		EcAddrErr:       "get_addr_error",
		EcUnavailable:   "unavailable",
		EcNotImplement:  "not_implement",
		EcType:          "err_type",
		EcWriteFail:     "write_fail",
		EcFail:          "fail_code",
		EcTooSlow:       "too_slow",
	}
)

func SetErrCodeToStr(ec TErrCode, str string) {
	_ErrCodeToString[ec] = str
}

func ErrCodeToStr(ec TErrCode) string {
	str, ok := _ErrCodeToString[ec]
	if ok {
		return str
	}
	return strconv.FormatInt(int64(ec), 10)
}

func WrapErr(code TErrCode, e error) *Err {
	if e == nil {
		return &Err{code: code, stack: GetStack(3)}
	}
	return &Err{code: code, stack: GetStack(3), params: M{"error": e.Error()}}
}

func NewErr(code TErrCode, params M) *Err {
	return &Err{code: code, stack: GetStack(3), params: params}
}

func NewNoStackErr(code TErrCode, params M) *Err {
	return &Err{code: code, stack: nil, params: params}
}

func NewErrWithStack(code TErrCode, stack []byte, params M) *Err {
	return &Err{code: code, stack: stack, params: params}
}

type Err struct {
	code   TErrCode
	stack  []byte
	params M
}

func (e *Err) Code() TErrCode {
	return e.code
}

func (e *Err) ToBytes() []byte {
	if e.params == nil {
		e.params = map[string]any{}
	}
	e.params["code"] = e.code
	if e.stack != nil {
		e.params["stack"] = BytesToStr(e.stack)
	}
	bytes, _ := JsonMarshal(e.params)
	return bytes
}

func (e *Err) Error() string {
	err, ok := e.params["error"]
	if ok {
		return err.(string)
	}
	return BytesToStr(e.ToBytes())
}

func (e *Err) Params() M {
	return e.params
}

func (e *Err) Stack() []byte {
	return e.stack
}

func (e *Err) IsNoStack() bool {
	return e.stack == nil
}

func (e *Err) AddParam(k string, v any) {
	if e.params == nil {
		e.params = map[string]any{}
	}
	e.params[k] = v
}

func (e *Err) GetParam(k string) (v any, ok bool) {
	if e.params == nil {
		return nil, false
	}
	v, ok = e.params[k]
	return
}

func (e *Err) AddParams(params M) {
	if e.params == nil {
		e.params = map[string]any{}
	}
	for k, v := range params {
		e.params[k] = v
	}
}

func (e *Err) UpdateParam(k string, action AnyBoolToAnyBool) {
	if e.params == nil {
		e.params = map[string]any{}
	}
	ov, ok := e.params[k]
	v, ok := action(ov, ok)
	if ok {
		e.params[k] = v
	} else {
		delete(e.params, k)
	}
}

func (e *Err) String() string {
	return ErrCodeToStr(e.code)
}

func GetStack(skip int) []byte {
	const depth = 16
	var rpc [depth]uintptr
	n := runtime.Callers(skip, rpc[:])
	if n < 1 {
		return nil
	}
	frames := runtime.CallersFrames(rpc[:])

	var buffer ByteBuffer
	buffer.InitCap(256)
	for i := 0; i < StackMaxDeep; i++ {
		frame, ok := frames.Next()
		if !ok {
			break
		}
		buffer.WUint8('\n')
		buffer.WUint8('\t')
		buffer.WStringNoLen(LogTrim(frame.File))
		buffer.WUint8(':')
		buffer.WStringNoLen(strconv.Itoa(frame.Line))
	}
	return buffer.All()
}

var (
	StackMaxDeep = 8
	LogPrefix    = ".."
	LogReg       = regexp.MustCompile(`(\/.+\.(com)|(org))|(\/.+go\d{1}\.\d{1,2}.\d{1,2}|/src)`)
)

func LogTrim(file string) string {
	s := LogReg.FindStringIndex(file)
	if len(s) > 0 {
		return LogPrefix + file[s[1]:]
	}
	return file
}
