package log

import (
	"context"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"os"
	"time"
)

const (
	cmdLog   = "log"
	cmdTrace = "trace"
	cmdSpan  = "span"
	cmdFlush = "flush"
	cmdOver  = "over"
)

type (
	mgoOption struct {
		logLvl, traceLvl kiwi.TLevel
		timeLayout       string
		db               string
		ttl              int32
		dbOpts           *options.DatabaseOptions
		clientOpts       *options.ClientOptions
		logOpt           *options.CreateCollectionOptions
		logIdx           []mongo.IndexModel
		spanOpt          *options.CreateCollectionOptions
		spanIdx          []mongo.IndexModel
		traceOpt         *options.CreateCollectionOptions
		traceIdx         []mongo.IndexModel
	}
	MgoOption func(opt *mgoOption)
)

func MgoLogLvl(levels ...string) MgoOption {
	return func(opt *mgoOption) {
		opt.logLvl = kiwi.StrLvlToMask(levels...)
	}
}

func MgoTraceLvl(levels ...string) MgoOption {
	return func(opt *mgoOption) {
		opt.traceLvl = kiwi.StrLvlToMask(levels...)
	}
}

func MgoTimeLayout(layout string) MgoOption {
	return func(opt *mgoOption) {
		opt.timeLayout = layout
	}
}

func MgoDb(db string) MgoOption {
	return func(opt *mgoOption) {
		opt.db = db
	}
}

func MgoTtl(ttl int32) MgoOption {
	return func(opt *mgoOption) {
		opt.ttl = ttl
	}
}

func MgoClientOptions(opts *options.ClientOptions) MgoOption {
	return func(opt *mgoOption) {
		opt.clientOpts = opts
	}
}

func MgoDbOptions(opts *options.DatabaseOptions) MgoOption {
	return func(opt *mgoOption) {
		opt.dbOpts = opts
	}
}

func MgoLogOpt(option *options.CreateCollectionOptions) MgoOption {
	return func(opt *mgoOption) {
		opt.logOpt = option
	}
}

func MgoLogIdx(index ...mongo.IndexModel) MgoOption {
	return func(opt *mgoOption) {
		opt.logIdx = index
	}
}

func MgoSpanOpt(option *options.CreateCollectionOptions) MgoOption {
	return func(opt *mgoOption) {
		opt.spanOpt = option
	}
}

func MgoSpanIdx(index ...mongo.IndexModel) MgoOption {
	return func(opt *mgoOption) {
		opt.spanIdx = index
	}
}

func MgoTraceOpt(option *options.CreateCollectionOptions) MgoOption {
	return func(opt *mgoOption) {
		opt.traceOpt = option
	}
}

func MgoTraceIdx(index ...mongo.IndexModel) MgoOption {
	return func(opt *mgoOption) {
		opt.traceIdx = index
	}
}

func NewMgo(opts ...MgoOption) *mgoLogger {
	opt := &mgoOption{
		logLvl:     kiwi.LvlToMask(kiwi.TestLevels...),
		traceLvl:   kiwi.LvlToMask(kiwi.TestLevels...),
		timeLayout: kiwi.DefTimeFormatter,
		db:         "log",
		ttl:        3600 * 24 * 7,
	}
	for _, o := range opts {
		o(opt)
	}
	if opt.clientOpts == nil {
		opt.clientOpts = options.Client().ApplyURI("mongodb://localhost:27017")
	}
	l := &mgoLogger{
		option: opt,
	}
	err := l.conn()
	if err != nil {
		panic(err.Error())
	}

	names, e := l.db.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		panic(e.Error())
	}
	nameMap := make(map[string]struct{})
	for _, name := range names {
		nameMap[name] = struct{}{}
	}

	if _, ok := nameMap[cmdLog]; !ok {
		e = l.db.CreateCollection(context.TODO(), cmdLog, l.option.logOpt)
		if e != nil {
			panic(e.Error())
		}
	}
	logColl := l.db.Collection(cmdLog)
	_, _ = logColl.Indexes().CreateMany(context.TODO(),
		append(l.option.logIdx,
			mongo.IndexModel{
				Keys:    bson.D{{"ts", -1}},
				Options: options.Index().SetExpireAfterSeconds(opt.ttl),
			},
			mongo.IndexModel{
				Keys: bson.D{{"lvl", 1}},
			}))
	l.logBuffer = newMgoBuffer(16, logColl)

	if _, ok := nameMap[cmdLog]; !ok {
		e = l.db.CreateCollection(context.TODO(), cmdTrace, l.option.traceOpt)
		if e != nil {
			panic(e.Error())
		}
	}
	traceColl := l.db.Collection(cmdTrace)
	_, _ = traceColl.Indexes().CreateMany(context.TODO(),
		append(l.option.spanIdx,
			mongo.IndexModel{
				Keys:    bson.D{{"ts", -1}},
				Options: options.Index().SetExpireAfterSeconds(opt.ttl),
			},
			mongo.IndexModel{
				Keys: bson.D{{"pid", -1}},
			},
			mongo.IndexModel{
				Keys: bson.D{{"tid", -1}},
			},
			mongo.IndexModel{
				Keys: bson.D{{"msg", 1}},
			}))
	l.traceBuffer = newMgoBuffer(32, traceColl)

	if _, ok := nameMap[cmdSpan]; !ok {
		e = l.db.CreateCollection(context.TODO(), cmdSpan, l.option.spanOpt)
		if e != nil {
			panic(e.Error())
		}
	}
	spanColl := l.db.Collection(cmdSpan)
	_, _ = spanColl.Indexes().CreateMany(context.TODO(),
		append(l.option.traceIdx,
			mongo.IndexModel{
				Keys:    bson.D{{"ts", -1}},
				Options: options.Index().SetExpireAfterSeconds(opt.ttl),
			},
			mongo.IndexModel{
				Keys: bson.D{{"tid", -1}},
			},
			mongo.IndexModel{
				Keys: bson.D{{"lvl", 1}},
			},
			mongo.IndexModel{
				Keys: bson.D{{"msg", 1}},
			}))
	l.spanBuffer = newMgoBuffer(128, spanColl)
	l.worker = worker.NewJobWorker(l.process)
	l.worker.Start()
	clearCh := make(chan chan struct{}, 1)
	kiwi.BeforeExitFn("mgo log", func() {
		overCh := make(chan struct{}, 1)
		go func() {
			time.Sleep(time.Millisecond * 100)
			clearCh <- overCh
		}()
		<-overCh
	})
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-ticker.C:
				l.worker.Push(cmdFlush)
			case ch := <-clearCh:
				l.worker.Push(cmdOver, ch)
				return
			}
		}
	}()
	return l
}

type mgoLogger struct {
	option      *mgoOption
	client      *mongo.Client
	db          *mongo.Database
	worker      *worker.JobWorker
	logBuffer   *mgoBuffer
	traceBuffer *mgoBuffer
	spanBuffer  *mgoBuffer
}

func (l *mgoLogger) conn() *util.Err {
	client, e := mongo.Connect(context.TODO(), l.option.clientOpts)
	if e != nil {
		return util.WrapErr(util.EcConnectErr, e)
	}

	e = client.Ping(context.TODO(), readpref.Primary())
	if e != nil {
		return util.WrapErr(util.EcConnectErr, e)
	}
	l.client = client
	l.db = client.Database(l.option.db, l.option.dbOpts)
	return nil
}

func (l *mgoLogger) Log(level kiwi.TLevel, msg, caller string, stack []byte, params util.M) {
	if !util.TestMask(level, l.option.logLvl) {
		return
	}
	l.worker.Push(cmdLog, &log{
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Message:   msg,
		Stack:     string(stack),
		Caller:    caller,
		Params:    params,
	})
}

func (l *mgoLogger) Trace(pid, tid int64, caller string, params util.M) {
	l.worker.Push(cmdTrace, &trace{
		Timestamp: time.Now().UnixMilli(),
		Pid:       pid,
		Tid:       tid,
		Caller:    caller,
		Params:    params,
	})
}

func (l *mgoLogger) Span(level kiwi.TLevel, tid int64, msg, caller string, stack []byte, params util.M) {
	l.worker.Push(cmdSpan, &span{
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Tid:       tid,
		Message:   msg,
		Stack:     string(stack),
		Caller:    caller,
		Params:    params,
	})
}

func (l *mgoLogger) process(job *worker.Job) {
	switch job.Name {
	case cmdLog:
		l.logBuffer.push(job.Data[0])
	case cmdTrace:
		l.traceBuffer.push(job.Data[0])
	case cmdSpan:
		l.spanBuffer.push(job.Data[0])
	case cmdFlush:
		l.logBuffer.flush()
		l.traceBuffer.flush()
		l.spanBuffer.flush()
	case cmdOver:
		l.logBuffer.flush()
		l.traceBuffer.flush()
		l.spanBuffer.flush()
		job.Data[0].(chan struct{}) <- struct{}{}
	}
}

func newMgoBuffer(cap int, coll *mongo.Collection) *mgoBuffer {
	b := &mgoBuffer{
		buffer: make([]any, cap),
		idx:    0,
		cap:    cap,
		coll:   coll,
	}
	return b
}

type mgoBuffer struct {
	buffer []any
	idx    int
	cap    int
	coll   *mongo.Collection
}

func (b *mgoBuffer) push(m any) {
	b.buffer[b.idx] = m
	b.idx++
	if b.idx < b.cap {
		return
	}
	b.flush()
}

func (b *mgoBuffer) flush() {
	if b.idx == 0 {
		return
	}
	_, err := b.coll.InsertMany(context.TODO(), b.buffer[:b.idx])
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
	b.idx = 0
}

type log struct {
	Timestamp int64       `bson:"ts"`
	Level     kiwi.TLevel `bson:"lvl"`
	Message   string      `bson:"msg"`
	Stack     string      `bson:"stk"`
	Caller    string      `bson:"cl"`
	Params    util.M      `bson:"p"`
}

type trace struct {
	Timestamp int64  `bson:"ts"`
	Pid       int64  `bson:"pid"`
	Tid       int64  `bson:"tid"`
	Caller    string `bson:"cl"`
	Params    util.M `bson:"p"`
}

type span struct {
	Timestamp int64       `bson:"ts"`
	Level     kiwi.TLevel `bson:"lvl"`
	Tid       int64       `bson:"tid"`
	Message   string      `bson:"msg"`
	Stack     string      `bson:"stk"`
	Caller    string      `bson:"cl"`
	Params    util.M      `bson:"p"`
}
