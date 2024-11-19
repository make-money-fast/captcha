package leveldb

import (
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

func (LeveldbStore) getKey(key string) []byte {
	return []byte(keyPrefix + key)
}

func (l *LeveldbStore) Set(id string, digits []byte) {
	_ = l.db.Put(l.getKey(id), newEntity(digits).Marshal(), nil)
}

func (l *LeveldbStore) Get(id string) (digits []byte) {
	val, _ := l.db.Get(l.getKey(id), nil)
	e := entityFromBytes(val)
	if e.expire(l.expireTime) {
		l.Del(id)
		return nil
	}
	return e.Digits
}

func (l *LeveldbStore) Del(id string) {
	_ = l.db.Delete(l.getKey(id), nil)
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
