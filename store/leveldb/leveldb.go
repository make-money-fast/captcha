package leveldb

import (
	"context"
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"time"
)

const (
	keyPrefix = "__captcha"
)

type LeveldbStore struct {
	db         *leveldb.DB
	expireTime time.Duration
}

func NewLeveldbStore(path string, expireTime time.Duration) *LeveldbStore {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		panic("open leveldb path failed: " + err.Error())
	}
	return &LeveldbStore{
		db:         db,
		expireTime: expireTime,
	}
}

func NewWithLeveldb(db *leveldb.DB, expireTime time.Duration) *LeveldbStore {
	return &LeveldbStore{
		db:         db,
		expireTime: expireTime,
	}
}

type entity struct {
	Digits   []byte `json:"d"`
	CreateTs int64  `json:"c"`
}

func newEntity(data []byte) *entity {
	return &entity{
		Digits:   data,
		CreateTs: time.Now().Unix(),
	}
}

func entityFromBytes(data []byte) *entity {
	var e entity
	json.Unmarshal(data, &e)
	return &e
}

func (e *entity) expire(duration time.Duration) bool {
	if duration == 0 {
		return false
	}
	return time.Since(time.Unix(e.CreateTs, 0).Add(duration)) < 0
}

func (e *entity) Marshal() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (l *LeveldbStore) getKey(key string) []byte {
	return []byte(keyPrefix + key)
}

func (l *LeveldbStore) Set(ctx context.Context, id string, digits []byte) {
	_ = l.db.Put(l.getKey(id), newEntity(digits).Marshal(), nil)
}

func (l *LeveldbStore) Get(ctx context.Context, id string) (digits []byte) {
	val, _ := l.db.Get(l.getKey(id), nil)
	e := entityFromBytes(val)
	if e.expire(l.expireTime) {
		l.Del(ctx, id)
		return nil
	}
	return e.Digits
}

func (l *LeveldbStore) Del(ctx context.Context, id string) {
	_ = l.db.Delete(l.getKey(id), nil)
}

func (l *LeveldbStore) GCOnce() {
	iter := l.db.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		val, _ := l.db.Get(iter.Key(), nil)
		if len(val) == 0 {
			l.db.Delete(iter.Key(), nil)
			continue
		}
		if entityFromBytes(val).expire(l.expireTime) {
			l.db.Delete(iter.Key(), nil)
		}
	}
}
