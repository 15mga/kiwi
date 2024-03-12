package rds

import (
	"strconv"
	"strings"

	"github.com/15mga/kiwi/util"
	"github.com/go-redsync/redsync/v4"
	"github.com/gomodule/redigo/redis"
)

const (
	DEL      = "DEL"      // DEL 用于在key存在时删除key
	DUMP     = "DUMP"     // DUMP 返回指定key序列化的值
	EXIST    = "EXIST"    // EXIST key是否存在
	EXPIRE   = "EXPIRE"   // EXPIRE 设置秒为单位的过期时间
	EXPIREAT = "EXPIREAT" // EXPIREAT 设置时间戳为过期时间
	PEXPIRE  = "PEXPIRE"  // PEXPIRE 设置毫秒为单位的过期时间
	KEYS     = "KEYS"     // KEYS 获取所有符合模式的键
	MOVE     = "MOVE"     // MOVE 将当前数据库的键值对移动到指定数据库
	PERSIST  = "PERSIST"  // PERSIST 移除key的过期时间
	PTTL     = "PTTL"     // PTTL 返回key以毫秒为单位的过期时间
	TTL      = "TTL"      // TTL 返回key以秒为单位的过期时间
	RENAME   = "RENAME"   // RENAME 修改key名称
	RENAMENX = "RENAMENX" // RENAMENX 将key修改为不存在的新key
	TYPE     = "TYPE"     // TYPE 返回key对应值的类型
	SCAN     = "SCAN"     // SCAN 迭代数据库中的数据库键
	MATCH    = "MATCH"    // MATCH 匹配

	SET         = "SET"         // SET 设置健值
	GET         = "GET"         // GET 获取指定key的值
	GETRANGE    = "GETRANGE"    // GETRANGE 返回指定key的值的子字符串
	GETSET      = "GETSET"      // GETSET 将给定key的值设为value,并返回key的旧值
	GETBIT      = "GETBIT"      // GETBIT 对 key 所储存的字符串值，获取指定偏移量上的位(bit)
	MGET        = "MGET"        // MGET 获取所有(一个或多个)给定 key 的值。
	SETBIT      = "SETBIT"      // SETBIT 对 key 所储存的字符串值，设置或清除指定偏移量上的位(bit)
	SETEX       = "SETEX"       // SETEX 将值 value 关联到 key ，并将 key 的过期时间设为 seconds (以秒为单位)。
	SETNX       = "SETNX "      // SETNX 只有在 key 不存在时设置 key 的值
	SETRANGE    = "SETRANGE"    // SETRANGE 用 value 参数覆写给定 key 所储存的字符串值，从偏移量 offset 开始
	STRLEN      = "STRLEN"      // STRLEN 返回 key 所储存的字符串值的长度
	MSET        = "MSET"        // MSET 同时设置一个或多个 key-value 对
	MSETNX      = "MSETNX"      // MSETNX 同时设置一个或多个 key-value 对，当且仅当所有给定 key 都不存在
	PSETEX      = "PSETEX"      // PSETEX 这个命令和 SETEX 命令相似，但它以毫秒为单位设置 key 的生存时间，而不是像 SETEX 命令那样，以秒为单位
	INCR        = "INCR"        // INCR 将 key 中储存的数字值增一
	INCRBY      = "INCRBY"      // INCRBY 将 key 所储存的值加上给定的增量值（increment）
	INCRBYFLOAT = "INCRBYFLOAT" // INCRBYFLOAT 将 key 所储存的值加上给定的浮点增量值（increment）
	DECR        = "DECR"        // DECR 将 key 中储存的数字值减一
	DECRBY      = "DECRBY"      // DECRBY key 所储存的值减去给定的减量值（decrement）
	APPEND      = "APPEND"      // APPEND 如果 key 已经存在并且是一个字符串， APPEND 命令将指定的 value 追加到该 key 原来值（value）的末尾

	HDEL         = "HDEL"         // HDEL 删除一个或多个哈希表字段
	HEXISTS      = "HEXISTS"      // HEXISTS 查看哈希表 key 中，指定的字段是否存在
	HGET         = "HGET"         // HGET 获取存储在哈希表中指定字段的值
	HGETALL      = "HGETALL"      // HGETALL 获取在哈希表中指定 key 的所有字段和值
	HINCRBY      = "HINCRBY"      // HINCRBY 为哈希表 key 中的指定字段的整数值加上增量 increment
	HINCRBYFLOAT = "HINCRBYFLOAT" // HINCRBYFLOAT 为哈希表 key 中的指定字段的浮点数值加上增量 increment
	HKEYS        = "HKEYS"        // HKEYS 获取所有哈希表中的字段
	HLEN         = "HLEN"         // HLEN 获取哈希表中字段的数量
	HMGET        = "HMGET"        // HMGET 获取所有给定字段的值
	HMSET        = "HMSET"        // HMSET 同时将多个 field-value (域-值)对设置到哈希表 key 中
	HSET         = "HSET"         // HSET 将哈希表 key 中的字段 field 的值设为 value
	HSETNX       = "HSETNX"       // HSETNX 只有在字段 field 不存在时，设置哈希表字段的值
	HVALS        = "HVALS"        // HVALS 获取哈希表中所有值
	HSCAN        = "HSCAN"        // HSCAN 迭代哈希表中的键值对

	BLPOP      = "BLPOP"      // 移出并获取列表的第一个元素， 如果列表没有元素会阻塞列表直到等待超时或发现可弹出元素为止
	BRPOP      = "BRPOP"      // 移出并获取列表的最后一个元素， 如果列表没有元素会阻塞列表直到等待超时或发现可弹出元素为止
	BRPOPLPUSH = "BRPOPLPUSH" // 从列表中弹出一个值，将弹出的元素插入到另外一个列表中并返回它； 如果列表没有元素会阻塞列表直到等待超时或发现可弹出元素为止
	LINDEX     = "LINDEX"     // 通过索引获取列表中的元素
	LINSERT    = "LINSERT"    // 在列表的元素前或者后插入元素
	BEFORE     = "BEFORE"
	AFTER      = "AFTER"
	LLEN       = "LLEN"      // 获取列表长度
	LPOP       = "LPOP"      // 移出并获取列表的第一个元素
	LPUSH      = "LPUSH"     // 将一个或多个值插入到列表头部
	LPUSHX     = "LPUSHX"    // 将一个值插入到已存在的列表头部
	LRANGE     = "LRANGE"    // 获取列表指定范围内的元素
	LREM       = "LREM"      // 移除列表元素
	LSET       = "LSET"      // 通过索引设置列表元素的值
	LTRIM      = "LTRIM"     // 对一个列表进行修剪(trim)，就是说，让列表只保留指定区间内的元素，不在指定区间之内的元素都将被删除
	RPOP       = "RPOP"      // 移除列表的最后一个元素，返回值为移除的元素
	RPOPLPUSH  = "RPOPLPUSH" // 移除列表的最后一个元素，并将该元素添加到另一个列表并返回
	RPUSH      = "RPUSH"     // 在列表中添加一个或多个值
	RPUSHX     = "RPUSHX"    // 为已存在的列表添加值

	SADD        = "SADD"        // SADD 向集合添加一个或多个成员
	SCARD       = "SCARD"       // SCARD 获取集合的成员数
	SDIFF       = "SDIFF"       // SDIFF 返回第一个集合与其他集合之间的差异
	SDIFFSTORE  = "SDIFFSTORE"  // SDIFFSTORE 返回给定所有集合的差集并存储在 destination 中
	SINTER      = "SINTER"      // SINTER 返回给定所有集合的交集
	SINTERSTORE = "SINTERSTORE" // SINTERSTORE 返回给定所有集合的交集并存储在 destination 中
	SISMEMBER   = "SISMEMBER"   // SISMEMBER 判断 member 元素是否是集合 key 的成员
	SMEMBERS    = "SMEMBERS"    // SMEMBERS 返回集合中的所有成员
	SMOVE       = "SMOVE"       // SMOVE 将 member 元素从 source 集合移动到 destination 集合
	SPOP        = "SPOP"        // SPOP 移除并返回集合中的一个随机元素
	SRANDMEMBER = "SRANDMEMBER" // SRANDMEMBER 返回集合中一个或多个随机数
	SREM        = "SREM"        // SREM 移除集合中一个或多个成员
	SUNION      = "SUNION"      // SUNION 返回所有给定集合的并集
	SUNIONSTORE = "SUNIONSTORE" // SUNIONSTORE 所有给定集合的并集存储在 destination 集合中
	SSCAN       = "SSCAN"       // SSCAN 迭代集合中的元素

	ZADD             = "ZADD"             // ZADD 向有序集合添加一个或多个成员，或者更新已存在成员的分数
	ZCARD            = "ZCARD"            // ZCARD 获取有序集合的成员数
	ZCOUNT           = "ZCOUNT"           // ZCOUNT 计算在有序集合中指定区间分数的成员数
	ZINCRBY          = "ZINCRBY"          // ZINCRBY 有序集合中对指定成员的分数加上增量 increment
	ZINTERSTOR       = "ZINTERSTOR"       // ZINTERSTOR 计算给定的一个或多个有序集的交集并将结果集存储在新的有序集合 destination 中
	ZLEXCOUNT        = "ZLEXCOUNT"        // ZLEXCOUNT 在有序集合中计算指定字典区间内成员数量
	ZRANGE           = "ZRANGE"           // ZRANGE 通过索引区间返回有序集合指定区间内的成员
	ZRANGEBYLEX      = "ZRANGEBYLEX"      // ZRANGEBYLEX 通过字典区间返回有序集合的成员
	ZRANGEBYSCORE    = "ZRANGEBYSCORE"    // ZRANGEBYSCORE 通过分数返回有序集合指定区间内的成员
	ZRANK            = "ZRANK"            // ZRANK 返回有序集合中指定成员的索引
	ZREM             = "ZREM"             // ZREM 移除有序集合中的一个或多个成员
	ZREMRANGEBYLEX   = "ZREMRANGEBYLEX"   // ZREMRANGEBYLEX 移除有序集合中给定的字典区间的所有成员
	ZREMRANGEBYRANK  = "ZREMRANGEBYRANK"  // ZREMRANGEBYRANK 移除有序集合中给定的排名区间的所有成员
	ZREMRANGEBYSCORE = "ZREMRANGEBYSCORE" // ZREMRANGEBYSCORE 移除有序集合中给定的分数区间的所有成员
	ZREVRANGE        = "ZREVRANGE"        // ZREVRANGE 返回有序集中指定区间内的成员，通过索引，分数从高到低
	ZREVRANGEBYSCORE = "ZREVRANGEBYSCORE" // ZREVRANGEBYSCORE 返回有序集中指定分数区间内的成员，分数从高到低排序
	ZREVRANK         = "ZREVRANK"         // ZREVRANK 返回有序集合中指定成员的排名，有序集成员按分数值递减(从大到小)排序
	ZSCORE           = "ZSCORE"           // ZSCORE 返回有序集中，成员的分数值
	ZUNIONSTORE      = "ZUNIONSTORE"      // ZUNIONSTORE 计算给定的一个或多个有序集的并集，并存储在新的 key 中
	ZSCAN            = "ZSCAN"            // ZSCAN 迭代有序集合中的元素（包括元素成员和元素分值）

	PFADD   = "PFADD"   // PFADD 添加指定元素到 HyperLogLog 中
	PFCOUNT = "PFCOUNT" // PFCOUNT 返回给定 HyperLogLog 的基数估算值
	PFMERGE = "PFMERGE" // PFMERGE 将多个 HyperLogLog 合并为一个 HyperLogLog

	PSUBSCRIBE   = "PSUBSCRIBE"   // PSUBSCRIBE 订阅一个或多个符合给定模式的频道
	PUBSUB       = "PUBSUB"       // PUBSUB 查看订阅与发布系统状态
	PUBLISH      = "PUBLISH"      // PUBLISH 将信息发送到指定的频道
	PUNSUBSCRIBE = "PUNSUBSCRIBE" // PUNSUBSCRIBE 退订所有给定模式的频道
	SUBSCRIBE    = "SUBSCRIBE"    // SUBSCRIBE 订阅给定的一个或多个频道的信息
	UNSUBSCRIBE  = "UNSUBSCRIBE"  // UNSUBSCRIBE 指退订给定的频道

	DISCARD = "DISCARD" // DISCARD 取消事务，放弃执行事务块内的所有命令
	EXEC    = "EXEC"    // EXEC 执行所有事务块内的命令
	MULTI   = "MULTI"   // MULTI 标记一个事务块的开始
	UNWATCH = "UNWATCH" // UNWATCH 取消 WATCH 命令对所有 key 的监视
	WATCH   = "WATCH"   // 监视一个(或多个) key ，如果在事务执行之前这个(或这些) key 被其他命令所改动，那么事务将被打断

	EVAL          = "EVAL"          // EVAL 执行 Lua 脚本
	EVALSHA       = "EVALSHA"       // EVALSHA 执行 Lua 脚本
	SCRIPT_EXISTS = "SCRIPT EXISTS" // SCRIPT_EXISTS 查看指定的脚本是否已经被保存在缓存当中
	SCRIPT_FLUSH  = "SCRIPT FLUSH"  // SCRIPT_FLUSH 从脚本缓存中移除所有脚本
	SCRIPT_KILL   = "SCRIPT KILL"   // SCRIPT_KILL 杀死当前正在运行的 Lua 脚本
	SCRIPT_LOAD   = "SCRIPT LOAD"   // SCRIPT_LOAD 将脚本 script 添加到脚本缓存中，但并不立即执行这个脚本

	AUTH   = "AUTH"   // AUTH 验证密码是否正确
	ECHO   = "ECHO"   // ECHO 打印字符串
	PING   = "PING"   // PING 查看服务是否运行
	QUIT   = "QUIT"   // QUIT 关闭当前连接
	SELECT = "SELECT" // SELECT 切换到指定的数据库

	BGREWRITEAOF     = "BGREWRITEAOF"     // BGREWRITEAOF 异步执行一个 AOF（AppendOnly File） 文件重写操作
	BGSAVE           = "BGSAVE"           // BGSAVE 在后台异步保存当前数据库的数据到磁盘
	CLIENT_KILL      = "CLIENT KILL"      // CLIENT_KILL 关闭客户端连接
	CLIENT_LIST      = "CLIENT LIST"      // CLIENT_LIST 获取连接到服务器的客户端连接列表
	CLIENT_GETNAME   = "CLIENT GETNAME"   // CLIENT_GETNAME 获取连接的名称
	CLIENT_PAUSE     = "CLIENT PAUSE"     // CLIENT_PAUSE 在指定时间内终止运行来自客户端的命令
	CLIENT_SETNAME   = "CLIENT SETNAME"   // CLIENT_SETNAME 设置当前连接的名称
	CLUSTER_SLOTS    = "CLUSTER SLOTS"    // CLUSTER_SLOTS 获取集群节点的映射数组
	COMMAND          = "COMMAND"          // COMMAND 获取 Redis 命令详情数组
	COMMAND_COUNT    = "COMMAND COUNT"    // COMMAND_COUNT 获取 Redis 命令总数
	COMMAND_GETKEYS  = "COMMAND GETKEYS"  // COMMAND_GETKEYS 获取给定命令的所有键
	TIME             = "TIME"             // TIME 返回当前服务器时间
	COMMAND_INFO     = "COMMAND INFO"     // COMMAND_INFO 获取指定 Redis 命令描述的数组
	CONFIG_GET       = "CONFIG GET"       // CONFIG_GET 获取指定配置参数的值
	CONFIG_REWRITE   = "CONFIG REWRITE"   // CONFIG_REWRITE 对启动 Redis 服务器时所指定的 redis.conf 配置文件进行改写
	CONFIG_SET       = "CONFIG SET"       // CONFIG_SET 修改 redis 配置参数，无需重启
	CONFIG_RESETSTAT = "CONFIG RESETSTAT" // CONFIG_RESETSTAT 重置 INFO 命令中的某些统计数据
	DBSIZE           = "DBSIZE"           // DBSIZE 返回当前数据库的 key 的数量
	DEBUG_OBJECT     = "DEBUG OBJECT"     // DEBUG_OBJECT 获取 key 的调试信息
	DEBUG_SEGFAULT   = "DEBUG SEGFAULT"   // DEBUG_SEGFAULT 让 Redis 服务崩溃
	FLUSHALL         = "FLUSHALL"         // FLUSHALL 删除所有数据库的所有key
	FLUSHDB          = "FLUSHDB"          // FLUSHDB 删除当前数据库的所有key
	INFO             = "INFO"             // INFO 获取 Redis 服务器的各种信息和统计数值
	LASTSAVE         = "LASTSAVE"         // LASTSAVE 返回最近一次 Redis 成功将数据保存到磁盘上的时间，以 UNIX 时间戳格式表示
	MONITOR          = "MONITOR"          // MONITOR 实时打印出 Redis 服务器接收到的命令，调试用
	ROLE             = "ROLE"             // ROLE 返回主从实例所属的角色
	SAVE             = "SAVE"             // SAVE 同步保存数据到硬盘
	SHUTDOWN         = "SHUTDOWN"         // SHUTDOWN 异步保存数据到硬盘，并关闭服务器
	SLAVEOF          = "SLAVEOF"          // SLAVEOF 将当前服务器转变为指定服务器的从属服务器(slave server)
	SLOWLOG          = "SLOWLOG"          // SLOWLOG 管理 redis 的慢日志
	SYNC             = "SYNC"             // SYNC 用于复制功能(replication)的内部命令

	GEOADD            = "GEOADD"            // GEOADD 存储指定的地理空间位置
	GEOPOS            = "GEOPOS "           // GEOPOS 获取key的地理位置
	GEODIST           = "GEODIST"           // GEODIST 返回2个点之间的距离
	GEORADIUS         = "GEORADIUS"         // GEORADIUS 返回以指定经纬中心范围内的的地理位置
	GEORADIUSBYMEMBER = "GEORADIUSBYMEMBER" // GEORADIUSBYMEMBER 返回以指定位置中心范围内的的地理位置
	GEOHASH           = "GEOHASH"           // GEOHASH 使用 geohash 来保存地理位置的坐标

	XADD               = "XADD"               // XADD 添加消息到末尾
	XTRIM              = "XTRIM"              // XTRIM 对流进行修剪，限制长度
	XDEL               = "XDEL"               // XDEL 删除消息
	XLEN               = "XLEN"               // XLEN 获取流包含的元素数量，即消息长度
	XRANGE             = "XRANGE"             // XRANGE 获取消息列表，会自动过滤已经删除的消息
	XREVRANGE          = "XREVRANGE"          // XREVRANGE 反向获取消息列表，ID 从大到小
	XREAD              = "XREAD"              // XREAD 以阻塞或非阻塞方式获取消息列表
	STREAMS            = "STREAMS"            // STREAMS 流
	COUNT              = "COUNT"              // COUNT 数量
	XGROUP             = "XGROUP"             // XGROUP 消费者组
	GROUP              = "GROUP"              // GROUP 消费者组
	CREATE             = "CREATE"             // CREATE 创建
	CREATECONSUMER     = "CREATECONSUMER"     // CREATECONSUMER 创建消费者
	XREADGROUP         = "XREADGROUP"         // XREADGROUP 读取消费者组中的消息
	XACK               = "XACK"               // XACK 将消息标记为已处理
	XGROUP_SETID       = "XGROUP SETID"       // XGROUP_SETID 为消费者组设置新的最后递送消息ID
	XGROUP_DELCONSUMER = "XGROUP DELCONSUMER" // XGROUP_DELCONSUMER 删除消费者
	XGROUP_DESTROY     = "XGROUP DESTROY"     // XGROUP_DESTROY 删除消费者组
	XPENDING           = "XPENDING"           // XPENDING 显示待处理消息的相关信息
	XCLAIM             = "XCLAIM"             // XCLAIM 转移消息的归属权
	XINFO              = "XINFO"              // XINFO 查看流和消费者组的相关信息
	XINFO_GROUPS       = "XINFO GROUPS"       // XINFO_GROUPS 打印消费者组的信息
	XINFO_STREAM       = "XINFO STREAM"       // XINFO_STREAM 打印流信息

	Json              = "$"
	JSON_ARR_APPEND   = "JSON.ARRAPPEND"    //json数组添加元素
	JSON_ARR_INDEX    = "JSON.ARRINDEX"     //
	JSON_ARR_INSERT   = "JSON.ARRINSERT"    //
	JSON_ARR_LEN      = "JSON.ARRLEN"       //
	JSON_ARR_POP      = "JSON.ARRPOP"       //
	JSON_ARR_TRIM     = "JSON.ARRTRIM"      //
	JSON_CLEAR        = "JSON.CLEAR"        //
	JSON_DEBUG        = "JSON.DEBUG"        //
	JSON_DEBUG_MEMORY = "JSON.DEBUG MEMORY" //
	JSON_DEL          = "JSON.DEL"          //
	JSON_FORGET       = "JSON.FORGET"       //
	JSON_GET          = "JSON.GET"          //
	JSON_MGET         = "JSON.MGET"         //
	JSON_NUM_INCRBY   = "JSON.NUMINCRBY"    //
	JSON_NUM_MULTBY   = "JSON.NUMMULTBY"    //
	JSON_OBJ_KEYS     = "JSON.OBJKEYS"      //
	JSON_OBJLEN       = "JSON.OBJLEN"       //
	JSON_RESP         = "JSON.RESP"         //
	JSON_SET          = "JSON.SET"          //
	JSON_STR_APPEND   = "JSON.STRAPPEND"    //
	JSON_STR_LEN      = "JSON.STRLEN"       //
	JSON_TOGGLE       = "JSON.TOGGLE"       //
	JSON_TYPE         = "JSON.TYPE"         //
)

type (
	ToRedisConn      func() redis.Conn
	ToRedisConnError func() (redis.Conn, error)
	ToRedisConnErr   func() (redis.Conn, *util.Err)
	ConnToErr        func(redis.Conn) *util.Err
	redisRedisOption struct {
		connFac  ToRedisConnError
		connPool *redis.Pool
	}
	RedisOption func(*redisRedisOption)
)

func ConnFac(fac ToRedisConnError) RedisOption {
	return func(opt *redisRedisOption) {
		opt.connFac = fac
	}
}

func ConnPool(pool *redis.Pool) RedisOption {
	return func(opt *redisRedisOption) {
		opt.connPool = pool
	}
}

func InitRedis(opts ...RedisOption) {
	_Redis = NewRedis(opts...)
}

func NewRedis(opts ...RedisOption) *Redis {
	opt := &redisRedisOption{}
	for _, o := range opts {
		o(opt)
	}
	return &Redis{
		redisRedisOption: opt,
	}
}

type Redis struct {
	redisRedisOption *redisRedisOption
	locker           *redsync.Redsync
}

func (r *Redis) GetConn() (redis.Conn, *util.Err) {
	conn, err := r.redisRedisOption.connFac()
	return conn, util.WrapErr(util.EcRedisErr, err)
}

func (r *Redis) SpawnConn() redis.Conn {
	return r.redisRedisOption.connPool.Get()
}

func (r *Redis) Lock(key string, fn func(), opts ...redsync.Option) *util.Err {
	if fn == nil {
		return nil
	}
	m := r.locker.NewMutex(key, opts...)
	e := m.Lock()
	if e != nil {
		return util.WrapErr(util.EcRedisErr, e)
	}
	fn()
	_, e = m.Unlock()
	return util.WrapErr(util.EcRedisErr, e)
}

var (
	_Redis *Redis
)

func GetConn() (redis.Conn, *util.Err) {
	return _Redis.GetConn()
}

func SpawnConn() redis.Conn {
	return _Redis.SpawnConn()
}

func FnSpawnConn(fn ConnToErr) (err *util.Err) {
	conn := _Redis.SpawnConn()
	err = fn(conn)
	_ = conn.Close()
	return
}

func Lock(key string, fn func(), opts ...redsync.Option) *util.Err {
	return _Redis.Lock(key, fn, opts...)
}

func JsonSet(conn redis.Conn, key string, obj any, fields ...string) *util.Err {
	str, _ := util.JsonMarshal(obj)
	f := mergeJsonFields(fields...)
	_, e := conn.Do(JSON_SET, key, f, str)
	if e != nil {
		return util.NewErr(util.EcRedisErr, util.M{
			"error":  "empty or more than one",
			"key":    key,
			"obj":    obj,
			"fields": fields,
		})
	}
	return nil
}

func JsonSetStr(conn redis.Conn, key string, str string, fields ...string) *util.Err {
	f := mergeJsonFields(fields...)
	_, e := conn.Do(JSON_SET, key, f, str)
	if e != nil {
		return util.NewErr(util.EcRedisErr, util.M{
			"error":  e.Error(),
			"key":    key,
			"str":    str,
			"fields": fields,
		})
	}
	return nil
}

func JsonGet[T any](conn redis.Conn, key string, v *T, fields ...string) *util.Err {
	var values []T
	err := JsonGetSlice[T](conn, key, &values, fields...)
	if err != nil {
		return err
	}
	if len(values) != 1 {
		return util.NewErr(util.EcRedisErr, util.M{
			"error":  "empty or more than one",
			"key":    key,
			"fields": fields,
		})
	}
	*v = values[0]
	return nil
}

func mergeJsonFields(fields ...string) string {
	f := Json
	if len(fields) > 0 {
		f += "." + strings.Join(fields, ".")
	}
	return f
}

func JsonGetSlice[T any](conn redis.Conn, key string, v *[]T, fields ...string) *util.Err {
	f := mergeJsonFields(fields...)
	reply, e := redis.Bytes(conn.Do(JSON_GET, key, f))
	if e != nil {
		return util.NewErr(util.EcRedisErr, util.M{
			"error":  e.Error(),
			"key":    key,
			"fields": fields,
		})
	}
	err := util.JsonUnmarshal(reply, v)
	if err != nil {
		return util.NewErr(util.EcRedisErr, util.M{
			"error":  err.Error(),
			"key":    key,
			"fields": fields,
		})
	}
	return nil
}

func Scan(conn redis.Conn, match string, step int, fn util.FnStrSlc) *util.Err {
	cursor := 0
	for {
		slc, e := redis.Values(conn.Do(SCAN, cursor, MATCH, match, COUNT, step))
		if e != nil {
			return util.WrapErr(util.EcRedisErr, e)
		}
		items := slc[1].([]any)
		keys := make([]string, len(items))
		for i, item := range items {
			keys[i] = string(item.([]byte))
		}
		fn(keys)
		idxStr := string(slc[0].([]byte))
		cursor, e = strconv.Atoi(idxStr)
		if e != nil {
			return util.NewErr(util.EcRedisErr, util.M{
				"error": e.Error(),
				"match": match,
				"step":  step,
			})
		}
		if cursor == 0 {
			return nil
		}
	}
}
