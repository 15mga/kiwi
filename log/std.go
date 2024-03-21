package log

import (
	"fmt"
	"github.com/15mga/kiwi"
	"io"
	"os"
	"strconv"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/15mga/kiwi/util"
)

const (
	_SDebug  = "[D]"
	_SInfo   = "[I]"
	_SWarn   = "[W]"
	_SError  = "[E]"
	_SFatal  = "[F]"
	_STrace  = "[TC]"
	_STDebug = "[TD]"
	_STInfo  = "[TI]"
	_STWarn  = "[TW]"
	_STError = "[TE]"
	_STFatal = "[TF]"
)

const (
	ColorRed      = "\033[31m"
	ColorGreen    = "\033[32m"
	ColorYellow   = "\033[33m"
	ColorBlue     = "\033[34m"
	ColorPurple   = "\033[35m"
	ColorCyan     = "\033[36m"
	ColorWhite    = "\033[37m"
	ColorHiRed    = "\033[91m"
	ColorHiGreen  = "\033[92m"
	ColorHiYellow = "\033[93m"
	ColorHiBlue   = "\033[94m"
	ColorHiPurple = "\033[95m"
	ColorHiCyan   = "\033[96m"
	ColorHiWhite  = "\033[97m"
	ColorReset    = "\033[0m"
)

func LogLvlToStr(l kiwi.TLevel) string {
	switch l {
	case kiwi.TDebug:
		return _SDebug
	case kiwi.TInfo:
		return _SInfo
	case kiwi.TWarn:
		return _SWarn
	case kiwi.TError:
		return _SError
	case kiwi.TFatal:
		return _SFatal
	default:
		return ""
	}
}

func TraceLvlToStr(l kiwi.TLevel) string {
	switch l {
	case kiwi.TDebug:
		return _STDebug
	case kiwi.TInfo:
		return _STInfo
	case kiwi.TWarn:
		return _STWarn
	case kiwi.TError:
		return _STError
	case kiwi.TFatal:
		return _STFatal
	default:
		return ""
	}
}

type (
	stdOption struct {
		logLvl, traceLvl kiwi.TLevel
		timeLayout       string
		color            bool
		writer           io.Writer
	}
	StdOption func(opt *stdOption)
)

func StdLogLvl(levels ...kiwi.TLevel) StdOption {
	return func(opt *stdOption) {
		opt.logLvl = kiwi.LvlToMask(levels...)
	}
}

func StdTraceLvl(levels ...kiwi.TLevel) StdOption {
	return func(opt *stdOption) {
		opt.traceLvl = kiwi.LvlToMask(levels...)
	}
}

func StdLogStrLvl(levels ...string) StdOption {
	return func(opt *stdOption) {
		opt.logLvl = kiwi.StrLvlToMask(levels...)
	}
}

func StdTraceStrLvl(levels ...string) StdOption {
	return func(opt *stdOption) {
		opt.traceLvl = kiwi.StrLvlToMask(levels...)
	}
}

func StdTimeLayout(layout string) StdOption {
	return func(opt *stdOption) {
		opt.timeLayout = layout
	}
}

func StdWriter(writer io.Writer) StdOption {
	return func(opt *stdOption) {
		opt.writer = writer
	}
}

func StdColor(color bool) StdOption {
	return func(opt *stdOption) {
		opt.color = color
	}
}

func StdFile(file string) StdOption {
	fmt.Println("log:", file)
	return func(opt *stdOption) {
		opt.writer = &lumberjack.Logger{
			Filename: file,
			MaxAge:   30,   //days
			Compress: true, // disabled by default
		}
	}
}

func NewStd(opts ...StdOption) *stdLogger {
	opt := &stdOption{
		logLvl:     kiwi.LvlToMask(kiwi.TestLevels...),
		traceLvl:   kiwi.LvlToMask(kiwi.TestLevels...),
		timeLayout: kiwi.DefTimeFormatter,
		color:      true,
		writer:     os.Stdout,
	}
	for _, o := range opts {
		o(opt)
	}
	f := &stdLogger{
		option: opt,
	}
	f.headLogDebug = _SDebug
	f.headLogInfo = _SInfo
	f.headLogWarn = _SWarn
	f.headLogError = _SError
	f.headLogFatal = _SFatal
	f.headSign = _STrace
	f.headTraceDebug = _STDebug
	f.headTraceInfo = _STInfo
	f.headTraceWarn = _STWarn
	f.headTraceError = _STError
	f.headTraceFatal = _STFatal
	f.tail = "\n"
	if opt.color {
		f.headLogDebug = ColorHiWhite + f.headLogDebug
		f.headLogInfo = ColorHiGreen + f.headLogInfo
		f.headLogWarn = ColorHiYellow + f.headLogWarn
		f.headLogError = ColorHiRed + f.headLogError
		f.headLogFatal = ColorHiPurple + f.headLogFatal
		f.headSign = ColorCyan + f.headSign
		f.headTraceDebug = ColorWhite + f.headTraceDebug
		f.headTraceInfo = ColorGreen + f.headTraceInfo
		f.headTraceWarn = ColorYellow + f.headTraceWarn
		f.headTraceError = ColorRed + f.headTraceError
		f.headTraceFatal = ColorPurple + f.headTraceFatal
		f.tail = ColorReset + f.tail
	}

	return f
}

type stdLogger struct {
	option         *stdOption
	headLogDebug   string
	headLogInfo    string
	headLogWarn    string
	headLogError   string
	headLogFatal   string
	headSign       string
	headTraceDebug string
	headTraceInfo  string
	headTraceWarn  string
	headTraceError string
	headTraceFatal string
	tail           string
}

func (l *stdLogger) getTimestamp() string {
	return time.Now().Format(l.option.timeLayout)
}

func (l *stdLogger) Log(level kiwi.TLevel, msg, caller string, stack []byte, params util.M) {
	if !util.TestMask(level, l.option.logLvl) {
		return
	}
	var buffer util.ByteBuffer
	if stack == nil {
		buffer.InitCap(512)
	} else {
		buffer.InitCap(1024)
	}
	switch level {
	case kiwi.TDebug:
		buffer.WStringNoLen(l.headLogDebug)
	case kiwi.TInfo:
		buffer.WStringNoLen(l.headLogInfo)
	case kiwi.TWarn:
		buffer.WStringNoLen(l.headLogWarn)
	case kiwi.TError:
		buffer.WStringNoLen(l.headLogError)
	case kiwi.TFatal:
		buffer.WStringNoLen(l.headLogFatal)
	}
	buffer.WStringNoLen(l.getTimestamp())
	if msg != "" {
		buffer.WStringNoLen(" ")
		buffer.WStringNoLen(msg)
	}
	buffer.WStringNoLen(l.tail)
	ps, _ := util.JsonMarshal(params)
	_, _ = buffer.Write(ps)
	buffer.WStringNoLen("\n")
	buffer.WStringNoLen(caller)
	if stack != nil {
		_, _ = buffer.Write(stack)
	}
	buffer.WStringNoLen("\n")
	_, _ = l.option.writer.Write(buffer.All())
	buffer.Dispose()
}

func (l *stdLogger) Trace(pid, tid int64, caller string, params util.M) {
	var buffer util.ByteBuffer
	buffer.InitCap(512)
	buffer.WStringNoLen(l.headSign)
	buffer.WStringNoLen(l.getTimestamp())
	buffer.WStringNoLen(" pid:")
	buffer.WStringNoLen(strconv.FormatInt(pid, 10))
	buffer.WStringNoLen(" tid:")
	buffer.WStringNoLen(strconv.FormatInt(tid, 10))
	buffer.WStringNoLen(l.tail)
	ps, _ := util.JsonMarshal(params)
	_, _ = buffer.Write(ps)
	buffer.WStringNoLen("\n")
	buffer.WStringNoLen(caller)
	buffer.WStringNoLen("\n")
	_, _ = l.option.writer.Write(buffer.All())
	buffer.Dispose()
}

func (l *stdLogger) Span(level kiwi.TLevel, tid int64, msg, caller string, stack []byte, params util.M) {
	if !util.TestMask(level, l.option.traceLvl) {
		return
	}
	var buffer util.ByteBuffer
	buffer.InitCap(1024)
	switch level {
	case kiwi.TDebug:
		buffer.WStringNoLen(l.headTraceDebug)
	case kiwi.TInfo:
		buffer.WStringNoLen(l.headTraceInfo)
	case kiwi.TWarn:
		buffer.WStringNoLen(l.headTraceWarn)
	case kiwi.TError:
		buffer.WStringNoLen(l.headTraceError)
	case kiwi.TFatal:
		buffer.WStringNoLen(l.headTraceFatal)
	}
	buffer.WStringNoLen(l.getTimestamp())
	buffer.WStringNoLen(" tid:")
	buffer.WStringNoLen(strconv.FormatInt(tid, 10))
	if msg != "" {
		buffer.WStringNoLen(" ")
		buffer.WStringNoLen(msg)
	}
	buffer.WStringNoLen(l.tail)
	ps, _ := util.JsonMarshal(params)
	_, _ = buffer.Write(ps)
	buffer.WStringNoLen("\n")
	buffer.WStringNoLen(caller)
	if stack != nil {
		_, _ = buffer.Write(stack)
	}
	buffer.WStringNoLen("\n")
	_, _ = l.option.writer.Write(buffer.All())
	buffer.Dispose()
}
