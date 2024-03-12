package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/log"
	"github.com/15mga/kiwi/util/mgo"
	"github.com/15mga/kiwi/util/rds"
	"github.com/15mga/kiwi/worker"
	"github.com/gomodule/redigo/redis"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type option struct {
	meta     *Meta
	mongo    *Mongo
	redis    *Redis
	worker   Worker
	node     kiwi.INode
	services []kiwi.IService
	gate     *Gate
	loggers  []kiwi.ILogger
}

type Meta struct {
	Id          int64
	SvcToVer    map[string]string
	SvcNameConv func(string) kiwi.TSvc
}

func SetMeta(meta *Meta) Option {
	return func(o *option) {
		o.meta = meta
	}
}

func SetLoggers(loggers ...kiwi.ILogger) Option {
	return func(o *option) {
		o.loggers = loggers
	}
}

func SetServices(services []kiwi.IService) Option {
	return func(o *option) {
		o.services = services
	}
}

type Mongo struct {
	uri     string
	db      string
	options *options.DatabaseOptions
}

func SetMongoDB(uri, db string, options *options.DatabaseOptions) Option {
	return func(o *option) {
		o.mongo = &Mongo{
			uri:     uri,
			db:      db,
			options: options,
		}
	}
}

type Redis struct {
	Addr     string
	User     string
	Password string
	Db       int
}

func SetRedis(addr, user, pw string, db int) Option {
	return func(o *option) {
		o.redis = &Redis{
			Addr:     addr,
			User:     user,
			Password: pw,
			Db:       db,
		}
	}
}

type Worker struct {
	active   bool
	share    bool
	parallel bool
	global   bool
}

func SetWorker(active, share, parallel, global bool) Option {
	return func(o *option) {
		o.worker = Worker{
			active:   active,
			share:    share,
			parallel: parallel,
			global:   global,
		}
	}
}

type Gate struct {
	receiver kiwi.FnAgentBytes
	options  []GateOption
}

func SetGate(receiver kiwi.FnAgentBytes, options ...GateOption) Option {
	return func(o *option) {
		o.gate = &Gate{
			receiver: receiver,
			options:  options,
		}
	}
}

type Option func(*option)

func StartDefault(opts ...Option) {
	opt := option{
		worker: Worker{
			active:   true,
			share:    true,
			parallel: true,
			global:   true,
		},
		node: NewNodeLocal(),
		loggers: []kiwi.ILogger{
			log.NewStd(),
		},
	}
	for _, o := range opts {
		o(&opt)
	}

	if len(opt.loggers) > 0 {
		for _, logger := range opt.loggers {
			kiwi.AddLogger(logger)
		}
	}

	nodeMeta := kiwi.GetNodeMeta()
	nodeMeta.Init(opt.meta.Id)
	for svc, ver := range opt.meta.SvcToVer {
		nodeMeta.AddService(opt.meta.SvcNameConv(svc), ver)
	}

	if opt.mongo != nil {
		clientOpt := options.Client().ApplyURI(opt.mongo.uri)
		err := mgo.Conn(opt.mongo.db, clientOpt, opt.mongo.options)
		if err != nil {
			panic(err)
		}

		err = mgo.CheckColl()
		if err != nil {
			kiwi.Fatal(err)
		}
	}

	if opt.redis != nil {
		rdsFac, rdsPool := getRedisFac(opt.redis)
		rds.InitRedis(
			rds.ConnFac(rdsFac),
			rds.ConnPool(rdsPool),
		)
	}

	if opt.worker.active {
		worker.InitActive()
	}
	if opt.worker.share {
		worker.InitShare()
	}
	if opt.worker.parallel {
		worker.InitParallel()
	}
	if opt.worker.global {
		worker.InitGlobal()
	}
	InitPacker()
	InitCodec()
	kiwi.SetNode(opt.node)
	InitRouter()
	RegisterSvc(opt.services...)

	if opt.gate != nil {
		InitGate(opt.gate.receiver, opt.gate.options...)
	}
	StartAllService()
	kiwi.WaitExit()
}

func getRedisFac(conf *Redis) (rds.ToRedisConnError, *redis.Pool) {
	redisFac := func() (redis.Conn, error) {
		return redis.Dial("tcp", conf.Addr,
			redis.DialUsername(conf.User),
			redis.DialPassword(conf.Password),
			redis.DialDatabase(conf.Db))
	}
	redisPool := &redis.Pool{
		Dial:        redisFac,
		IdleTimeout: 300 * time.Second,
		MaxActive:   512,
		MaxIdle:     512,
	}
	return redisFac, redisPool
}
