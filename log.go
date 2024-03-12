package kiwi

import (
	"github.com/15mga/kiwi/sid"
	"github.com/15mga/kiwi/util"
	"os"
	"runtime"
	"strconv"
)

// ILogger 日志
type ILogger interface {
	// Log 记录日志
	Log(level TLevel, msg, caller string, stack []byte, params util.M)
	// Trace 标记链路
	Trace(pid, tid int64, caller string, params util.M)
	// Span 链路日志
	Span(level TLevel, tid int64, msg, caller string, stack []byte, params util.M)
}

var (
	TestLevels = []TLevel{TDebug, TInfo, TWarn, TError, TFatal}
	DevLevels  = []TLevel{TInfo, TWarn, TError, TFatal}
	ProdLevels = []TLevel{TWarn, TError, TFatal}
)

type TLevel = int64

func StrToLevel(l string) TLevel {
	switch l {
	case SDebug:
		return TDebug
	case SInfo:
		return TInfo
	case SWarn:
		return TWarn
	case SError:
		return TError
	case SFatal:
		return TFatal
	default:
		return TInfo
	}
}

func LevelToStr(l TLevel) string {
	switch l {
	case TDebug:
		return SDebug
	case TInfo:
		return SInfo
	case TWarn:
		return SWarn
	case TError:
		return SError
	case TFatal:
		return SFatal
	default:
		return SInfo
	}
}

const (
	TDebug TLevel = 1 << iota
	TInfo
	TWarn
	TError
	TFatal
)

const (
	SDebug = "debug"
	SInfo  = "info"
	SWarn  = "warn"
	SError = "error"
	SFatal = "fatal"
)

const (
	DefTimeFormatter = "2006-01-02 15:04:05.999"
)

func StrLvlToMask(levels ...string) TLevel {
	slc := make([]TLevel, 0, len(levels))
	for _, level := range levels {
		slc = append(slc, StrToLevel(level))
	}
	return util.GenMask(slc...)
}

func LvlToMask(levels ...int64) TLevel {
	slc := make([]TLevel, 0, len(levels))
	for _, level := range levels {
		slc = append(slc, level)
	}
	return util.GenMask(slc...)
}

var (
	_LogDefParams    = util.M{}
	_LogDefParamsLen int
)

func SetLogDefParams(params util.M) {
	for k, v := range params {
		_LogDefParams[k] = v
	}
	_LogDefParamsLen = len(_LogDefParams)
}

func copyLogParams(params util.M) {
	for k, v := range _LogDefParams {
		params[k] = v
	}
}

var (
	_Loggers    []ILogger
	_CallerSkip = 2
)

func AddLogger(logger ILogger) {
	_Loggers = append(_Loggers, logger)
}

func SetCallerSkip(skip int) {
	_CallerSkip = skip
}

func log(level TLevel, msg string, stack []byte, params util.M) {
	var caller string
	for _, l := range _Loggers {
		if params == nil && _LogDefParamsLen > 0 {
			params = make(util.M, _LogDefParamsLen)
		}
		copyLogParams(params)
		if caller == "" {
			caller = GetCaller(_CallerSkip + 1)
		}
		l.Log(level, msg, caller, stack, params)
	}
}

func span(level TLevel, tid int64, msg string, stack []byte, params util.M) {
	var caller string
	for _, l := range _Loggers {
		if params == nil && _LogDefParamsLen > 0 {
			params = make(util.M, _LogDefParamsLen)
		}
		copyLogParams(params)
		if caller == "" {
			caller = GetCaller(_CallerSkip + 1)
		}
		l.Span(level, tid, msg, caller, stack, params)
	}
}

func Debug(str string, params util.M) {
	log(TDebug, str, nil, params)
}

func Info(str string, params util.M) {
	log(TInfo, str, nil, params)
}

func Warn(err *util.Err) {
	if err == nil {
		return
	}
	log(TWarn, err.String(), err.Stack(), err.Params())
}

func Warn2(code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	log(TWarn, err.String(), err.Stack(), err.Params())
}

func Warn3(code util.TErrCode, e error) {
	err := util.WrapErr(code, e)
	log(TWarn, err.String(), err.Stack(), err.Params())
}

func Error(err *util.Err) {
	if err == nil {
		return
	}
	log(TError, err.String(), err.Stack(), err.Params())
}

func Error2(code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	log(TError, err.String(), err.Stack(), err.Params())
}

func Error3(code util.TErrCode, e error) {
	err := util.WrapErr(code, e)
	log(TError, err.String(), err.Stack(), err.Params())
}

func Fatal(err *util.Err) {
	if err == nil {
		return
	}
	log(TFatal, err.String(), err.Stack(), err.Params())
	os.Exit(1)
}

func Fatal2(code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	log(TFatal, err.String(), err.Stack(), err.Params())
	os.Exit(1)
}

func Fatal3(code util.TErrCode, e error) {
	err := util.WrapErr(code, e)
	log(TFatal, err.String(), err.Stack(), err.Params())
	os.Exit(1)
}

// TC 链路标记
func TC(pid int64, params util.M, exclude bool) int64 {
	tid := sid.GetId()
	if !exclude {
		var caller string
		for _, l := range _Loggers {
			if params == nil && _LogDefParamsLen > 0 {
				params = make(util.M, _LogDefParamsLen)
			}
			copyLogParams(params)
			if caller == "" {
				caller = GetCaller(_CallerSkip)
			}
			l.Trace(pid, tid, caller, params)
		}
	}
	return tid
}

// TD 链路Debug
func TD(tid int64, msg string, params util.M) {
	span(TDebug, tid, msg, nil, params)
}

// TI 链路Info
func TI(tid int64, msg string, params util.M) {
	span(TInfo, tid, msg, nil, params)
}

// TW 链路Warn
func TW(tid int64, err *util.Err) {
	if err == nil {
		return
	}
	span(TWarn, tid, err.String(), err.Stack(), err.Params())
}

func TW2(tid int64, code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	span(TWarn, tid, err.String(), err.Stack(), err.Params())
}

func TW3(tid int64, code util.TErrCode, e error) {
	err := util.WrapErr(code, e)
	span(TWarn, tid, err.String(), err.Stack(), err.Params())
}

// TE 链路Error
func TE(tid int64, err *util.Err) {
	if err == nil {
		return
	}
	span(TError, tid, err.String(), err.Stack(), err.Params())
}

func TE2(tid int64, code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	span(TError, tid, err.String(), err.Stack(), err.Params())
}

func TE3(tid int64, code util.TErrCode, e error) {
	err := util.WrapErr(code, e)
	span(TError, tid, err.String(), err.Stack(), err.Params())
}

// TF 链路Fatal
func TF(tid int64, err *util.Err) {
	if err == nil {
		return
	}
	span(TFatal, tid, err.String(), err.Stack(), err.Params())
}

func TF2(tid int64, code util.TErrCode, m util.M) {
	err := util.NewErr(code, m)
	span(TFatal, tid, err.String(), err.Stack(), err.Params())
}

func GetCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	str := util.LogTrim(file) + ":" + strconv.Itoa(line)
	return str
}
