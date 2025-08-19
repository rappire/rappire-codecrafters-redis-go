package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/entity"
)

type Store struct {
	items map[string]entity.Entity
	mu    sync.RWMutex
}

func NewStore() *Store {
	return &Store{items: make(map[string]entity.Entity)}
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
	stringEntity, ok := entry.(*entity.StringEntity)
	if !ok {
		return "", false
	}
	return stringEntity.Value(), true
}

func (store *Store) Set(key, value string, expire time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := &entity.StringEntity{ValueData: value, Expire: expire}
	store.items[key] = entry
}

func (store *Store) ensureList(key string) *entity.ListEntity {
	if e, ok := store.items[key].(*entity.ListEntity); ok {
		return e
	}
	le := entity.NewListEntity()
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
		case listEntity.Notify() <- struct{}{}:
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
		case listEntity.Notify() <- struct{}{}:
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

	listEntity, ok := store.items[key].(*entity.ListEntity)
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

	listEntity, ok := store.items[key].(*entity.ListEntity)
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

	listEntity, ok := store.items[key].(*entity.ListEntity)
	if !ok {
		return nil, false
	}

	return listEntity.ValueData.LPop(count), true
}

func (store *Store) BLPop(key string, timeOut time.Duration) ([]byte, bool) {
	store.mu.Lock()

	if store.items[key] == nil {
		store.items[key] = entity.NewListEntity()
	}

	listEntity, ok := store.items[key].(*entity.ListEntity)
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
	notify := listEntity.Notify()
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

	listEntity, ok = store.items[key].(*entity.ListEntity)
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

	entry := store.items[key]
	if entry == nil {
		return "none"
	}

	switch entry.(type) {
	case *entity.StringEntity:
		return "string"
	case *entity.ListEntity:
		return "list"
	case *entity.StreamEntity:
		return "stream"
	default:
		return "none"
	}
}

func (store *Store) ensureStream(key string) *entity.StreamEntity {
	if streamEntity, ok := store.items[key].(*entity.StreamEntity); ok {
		return streamEntity
	}

	streamEntity := entity.NewStreamEntity()
	store.items[key] = streamEntity
	return streamEntity
}

func (store *Store) XAdd(key string, id string, fields []entity.FieldValue) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	streamEntity := store.ensureStream(key)
	generateId, err := streamEntity.GenerateId(id)
	if err != nil {
		return "", err
	}
	entry := entity.StreamEntry{Id: generateId, Fields: fields}
	streamEntity.Entries = append(streamEntity.Entries, entry)
	return fmt.Sprintf("%d-%d", generateId.Millis, generateId.Seq), nil
}

func (store *Store) XRange(key string, start string, end string) ([]entity.StreamEntry, error) {
	store.mu.RLock()
	streamEntity := store.ensureStream(key)
	store.mu.RUnlock()

	startId, err := entity.ParseBound(start)
	if err != nil {
		return nil, err
	}
	endId, err := entity.ParseBound(end)
	if err != nil {
		return nil, err
	}

	var result []entity.StreamEntry
	for _, e := range streamEntity.Entries {
		if e.Id == nil {
			continue
		}
		if !e.Id.Less(startId) && !endId.Less(e.Id) {
			result = append(result, e)
		}
	}
	return result, nil
}

// TODO 포인터로 최적화 필요
func (store *Store) XRead(dur time.Duration, keys []string, ids []string) ([][]entity.StreamEntry, error) {
	var streamIds []*entity.StreamId
	result := make([][]entity.StreamEntry, len(ids))

	for i, id := range ids {
		bound, err := entity.ParseBound(id)
		if err != nil {
			return result, nil
		}
		streamIds[i] = bound
	}

	store.mu.RLock()
	for i, key := range keys {
		stream := store.ensureStream(key)
		result[i] = []entity.StreamEntry{}
		for _, e := range stream.Entries {
			if e.Id.Less(streamIds[i]) {
				continue
			}
			result[i] = append(result[i], e)
		}
	}
	store.mu.RUnlock()

	return result, nil
}
