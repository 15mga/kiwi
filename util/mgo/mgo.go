package mgo

import (
	"context"
	"errors"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/panjf2000/ants/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	_Client    *mongo.Client
	_Db        *mongo.Database
	_Coll      = make(map[string]*mongo.Collection)
	_IdxModel  = make(map[string]func() []mongo.IndexModel)
	_ExistColl = make(map[string]struct{})
)

func Client() *mongo.Client {
	return _Client
}

func Db() *mongo.Database {
	return _Db
}

func Conn(db string, clientOpt *options.ClientOptions, dbOpt *options.DatabaseOptions) *util.Err {
	client, e := mongo.Connect(context.TODO(), clientOpt)
	if e != nil {
		return util.WrapErr(util.EcConnectErr, e)
	}

	e = client.Ping(context.TODO(), readpref.Primary())
	if e != nil {
		return util.WrapErr(util.EcConnectErr, e)
	}

	kiwi.BeforeExitFn("mgo", DisConn)

	_Client = client
	_Db = _Client.Database(db, dbOpt)

	return nil
}

func DisConn() {
	if _Client == nil {
		return
	}
	_ = _Client.Disconnect(context.Background())
	_Client = nil
}

func InitColl(coll string, idxModel func() []mongo.IndexModel, opts ...*options.CollectionOptions) {
	cl := _Db.Collection(coll, opts...)
	_Coll[coll] = cl
	if idxModel != nil {
		_IdxModel[coll] = idxModel
	}
}

func ExistColl(coll string) bool {
	_, ok := _ExistColl[coll]
	return ok
}

func CheckColl() *util.Err {
	names, e := _Db.ListCollectionNames(context.TODO(), nil)
	if e != nil && !errors.Is(e, mongo.ErrNilDocument) {
		return util.NewErr(util.EcDbErr, util.M{
			"error": e.Error(),
		})
	}
	m := make(map[string]struct{}, len(names))
	for name, coll := range _Coll {
		if _, ok := m[name]; ok {
			_ExistColl[name] = struct{}{}
			continue
		}
		fn, ok := _IdxModel[name]
		if !ok {
			continue
		}
		idx := fn()
		if len(idx) == 0 {
			continue
		}
		_, e = coll.Indexes().CreateMany(context.TODO(), idx)
		if e != nil {
			return util.NewErr(util.EcDbErr, util.M{
				"schema": name,
				"error":  e.Error(),
			})
		}
	}
	return nil
}

func Coll(coll string) *mongo.Collection {
	return _Coll[coll]
}

func CountDoc(coll string, filter any, opts ...*options.CountOptions) (int64, error) {
	return Coll(coll).CountDocuments(context.TODO(), filter, opts...)
}

func FindOne(coll string, filter any, item any, opts ...*options.FindOneOptions) error {
	return Coll(coll).FindOne(context.TODO(), filter, opts...).Decode(item)
}

func AsyncFindOne[T any](coll string, filter any, fn func(*T, error), opts ...*options.FindOneOptions) {
	e := ants.Submit(func() {
		item := util.Default[T]()
		err := FindOne(coll, filter, &item, opts...)
		if err != nil {
			fn(&item, err)
			return
		}
		fn(&item, nil)
	})
	if e != nil {
		item := util.Default[T]()
		fn(&item, e)
	}
}

func Find[T any](coll string, filter any, items *[]*T, opts ...*options.FindOptions) error {
	cursor, e := Coll(coll).Find(context.TODO(), filter, opts...)
	if e != nil {
		return e
	}
	return cursor.All(context.TODO(), items)
}

func AsyncFind[T any](coll string, filter any, fn func([]*T, error), opts ...*options.FindOptions) {
	e := ants.Submit(func() {
		var items []*T
		err := Find[T](coll, filter, &items, opts...)
		if err != nil {
			fn(nil, err)
			return
		}
		fn(items, nil)
	})
	if e != nil {
		fn(nil, e)
	}
}

func FindWithTotal[T any](coll string, filter any, count *int64, list *[]*T, opts ...*options.FindOptions) error {
	total, e := CountDoc(coll, filter)
	if e != nil {
		return e
	}
	if total == 0 {
		return nil
	}
	*count = total
	Find[T](coll, filter, list, opts...)
	return nil
}

func AsyncFindWithTotal[T any](coll string, filter any, fn func(int64, []*T, error), opts ...*options.FindOptions) {
	total, e := CountDoc(coll, filter)
	if e != nil {
		fn(0, nil, e)
		return
	}
	if total == 0 {
		fn(0, nil, nil)
		return
	}
	AsyncFind[T](coll, filter, func(list []*T, e error) {
		if e != nil {
			fn(0, nil, e)
			return
		}
		fn(total, list, nil)
	}, opts...)
}

func InsertOne(coll string, item any, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return Coll(coll).InsertOne(context.TODO(), item, opts...)
}

func InsertMany(coll string, items []any, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	return Coll(coll).InsertMany(context.TODO(), items, opts...)
}

func UpdateOne(coll string, filter, update any, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return Coll(coll).UpdateOne(context.TODO(), filter, update, opts...)
}

func AsyncUpdateOne(coll string, filter, update any, fn func(*mongo.UpdateResult, error), opts ...*options.UpdateOptions) {
	e := ants.Submit(func() {
		fn(UpdateOne(coll, filter, update, opts...))
	})
	if e != nil {
		fn(nil, e)
	}
}

func FindOneAndUpdate(coll string, filter, update any, item any, opts ...*options.FindOneAndUpdateOptions) error {
	return Coll(coll).FindOneAndUpdate(context.TODO(), filter, update, opts...).Decode(item)
}

func AsyncFindOneAndUpdate[T any](coll string, filter, update any, fn func(*T, error), opts ...*options.FindOneAndUpdateOptions) {
	e := ants.Submit(func() {
		item := util.Default[T]()
		e := FindOneAndUpdate(coll, filter, update, &item, opts...)
		if e != nil {
			fn(nil, e)
			return
		}
		fn(&item, nil)
	})
	if e != nil {
		fn(nil, e)
	}
}

func UpdateMany(coll string, filter, update any, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return Coll(coll).UpdateMany(context.TODO(), filter, update, opts...)
}

func ReplaceOne(coll string, filter, replacement any, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	return Coll(coll).ReplaceOne(context.TODO(), filter, replacement, opts...)
}

func DelOne(coll string, filter any, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return Coll(coll).DeleteOne(context.TODO(), filter, opts...)
}

func DelMany(coll string, filter any, opts ...*options.DeleteOptions) (int64, *util.Err) {
	res, e := Coll(coll).DeleteMany(context.TODO(), filter, opts...)
	if e != nil {
		return 0, util.NewErr(util.EcDbErr, util.M{
			"error":  e.Error(),
			"coll":   coll,
			"filter": filter,
		})
	}
	return res.DeletedCount, nil
}

func FindOneAndDel(coll string, filter any, item any, opts ...*options.FindOneAndDeleteOptions) error {
	return Coll(coll).FindOneAndDelete(context.TODO(), filter, opts...).Decode(item)
}

func Tx(fn func(mongo.SessionContext) error) *util.Err {
	session, e := _Client.StartSession()
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	defer session.EndSession(context.TODO())

	e = session.StartTransaction()
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	mongo.WithSession(context.TODO(), session, fn)
	e = session.CommitTransaction(context.TODO())
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	return nil
}

func BuildProjectionD(d *bson.D, exclude []string, keys ...string) {
	if exclude == nil || len(exclude) == 0 {
		for _, key := range keys {
			*d = append(*d, bson.E{Key: key, Value: 1})
		}
		return
	}
	if len(keys) > 0 {
		m := make(map[string]struct{}, len(exclude))
		for _, s := range exclude {
			m[s] = struct{}{}
		}
		for _, key := range keys {
			_, ok := m[key]
			if ok {
				continue
			}
			*d = append(*d, bson.E{Key: key, Value: 1})
		}
	} else {
		for _, key := range exclude {
			*d = append(*d, bson.E{Key: key, Value: 0})
		}
	}
}
