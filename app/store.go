package main

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/list"

	"sync"
	"time"
)

type Store struct {
	items map[string]Entity
	mu    sync.RWMutex
}

type Entity interface {
	Expired() bool
}

type StreamEntry struct {
	Id     string
	Fields map[string]string
}

type StreamEntity struct {
	Entries []StreamEntry
	LastMs  int64
	Seq     int64
}

func (s StreamEntity) Expired() bool {
	return false
}

func (s StreamEntity) nextId() string {
	now := time.Now().UnixMilli()
	if now == s.LastMs {
		s.Seq++
	} else {
		s.LastMs = now
		s.Seq = 0
	}
	return fmt.Sprintf("%d-%d", s.LastMs, s.Seq)
}

type ListEntity struct {
	ValueData *list.QuickList
	notify    chan struct{}
}

func (l ListEntity) Expired() bool {
	return false
}

func newListEntity() *ListEntity {
	return &ListEntity{
		ValueData: list.NewQuickList(),
		notify:    make(chan struct{}, 1),
	}
}

type StringEntity struct {
	ValueData string
	Expire    time.Time
}

func (e StringEntity) Value() string {
	return e.ValueData
}

func (e StringEntity) Expired() bool {
	return !e.Expire.IsZero() && time.Now().After(e.Expire)
}

func NewStore() *Store {
	return &Store{items: make(map[string]Entity)}
}

func (store *Store) Get(key string) (string, bool) {
	store.mu.RLock()
	entry, ok := store.items[key]
	store.mu.RUnlock()
	if !ok {
		return "", false
	}
	if entry.Expired() {
		store.mu.Lock()
		if cur, exists := store.items[key]; exists && cur == entry {
			delete(store.items, key)
		}
		store.mu.Unlock()
		return "", false
	}
	stringEntity, ok := entry.(*StringEntity)
	if !ok {
		return "", false
	}
	return stringEntity.Value(), true
}

func (store *Store) Set(key, value string, expire time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := &StringEntity{ValueData: value, Expire: expire}
	store.items[key] = entry
}

func (store *Store) ensureList(key string) *ListEntity {
	if e, ok := store.items[key].(*ListEntity); ok {
		return e
	}
	le := newListEntity()
	store.items[key] = le
	return le
}

func (store *Store) RPush(key string, value [][]byte) (int, bool) {

	store.mu.Lock()
	defer store.mu.Unlock()
	listEntity := store.ensureList(key)
	wasEmpty := listEntity.ValueData.Len() == 0
	n := listEntity.ValueData.RPush(value)

	if wasEmpty && n > 0 {
		select {
		case listEntity.notify <- struct{}{}:
		default:
		}
	}

	return n, true
}

func (store *Store) LPush(key string, value [][]byte) (int, bool) {

	store.mu.Lock()
	defer store.mu.Unlock()
	listEntity := store.ensureList(key)
	wasEmpty := listEntity.ValueData.Len() == 0
	n := listEntity.ValueData.LPush(value)

	if wasEmpty && n > 0 {
		select {
		case listEntity.notify <- struct{}{}:
		default:
		}
	}

	return n, true

}

func (store *Store) LRange(key string, startPos int, endPos int) ([][]byte, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if store.items[key] == nil {
		return [][]byte{}, true
	}

	listEntity, ok := store.items[key].(*ListEntity)
	if !ok {
		return nil, false
	}

	return listEntity.ValueData.LRange(startPos, endPos), true
}

func (store *Store) LLen(key string) (int, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if store.items[key] == nil {
		return 0, true
	}

	listEntity, ok := store.items[key].(*ListEntity)
	if !ok {
		return 0, false
	}

	return listEntity.ValueData.Len(), true

}

func (store *Store) LPop(key string, count int) ([][]byte, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.items[key] == nil {
		return nil, true
	}

	listEntity, ok := store.items[key].(*ListEntity)
	if !ok {
		return nil, false
	}

	return listEntity.ValueData.LPop(count), true
}

func (store *Store) BLPop(key string, timeOut time.Duration) ([]byte, bool) {
	store.mu.Lock()

	if store.items[key] == nil {
		store.items[key] = newListEntity()
	}

	listEntity, ok := store.items[key].(*ListEntity)
	if !ok {
		return []byte{}, false
	}

	if listEntity.ValueData.Len() > 0 {
		val := listEntity.ValueData.LPop(1)
		store.mu.Unlock()
		if len(val) == 0 {
			return nil, true
		}
		return val[0], true
	}
	notify := listEntity.notify
	store.mu.Unlock()

	if timeOut > 0 {
		select {
		case <-notify:
		case <-time.After(timeOut):
			return nil, true
		}
	} else {
		<-notify
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	listEntity, ok = store.items[key].(*ListEntity)
	if !ok {
		return nil, true
	}

	out := listEntity.ValueData.LPop(1)
	if len(out) == 0 {
		return nil, true
	}
	return out[0], true
}

func (store *Store) Type(key string) string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	entity := store.items[key]
	if entity == nil {
		return "none"
	}

	switch entity.(type) {
	case *StringEntity:
		return "string"
	case *ListEntity:
		return "list"
	case *StreamEntity:
		return "stream"
	default:
		return "none"
	}
}

func (store *Store) ensureStream(key string) *StreamEntity {
	if streamEntity, ok := store.items[key].(*StreamEntity); ok {
		return streamEntity
	}

	streamEntity := &StreamEntity{}
	store.items[key] = streamEntity
	return streamEntity
}

func (store *Store) XAdd(key string, id string, fields map[string]string) (string, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	streamEntity := store.ensureStream(key)

	if id == "*" {
		id = streamEntity.nextId()
	}

	entry := StreamEntry{Id: id, Fields: fields}
	streamEntity.Entries = append(streamEntity.Entries, entry)

	return id, true
}
