package main

import (
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/list"
)

type Store struct {
	items map[string]Entity
	mu    sync.RWMutex
}

type Entity interface {
	Expired() bool
}

type ListEntity struct {
	ValueData *list.QuickList
}

func (l ListEntity) Expired() bool {
	return false
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
	defer store.mu.RUnlock()
	entry, ok := store.items[key]
	if !ok {
		return "", false
	}
	if entry.Expired() {
		store.mu.RUnlock()
		store.mu.Lock()
		delete(store.items, key)
		store.mu.Unlock()
		store.mu.RLock()
		return "", false
	}
	stringEntity, ok := entry.(StringEntity)
	if !ok {
		return "", false
	}
	return stringEntity.Value(), true
}

func (store *Store) Set(key, value string, expire time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := StringEntity{ValueData: value, Expire: expire}
	store.items[key] = entry
}

func (store *Store) RPush(key string, value [][]byte) (int, bool) {

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.items[key] == nil {
		store.items[key] = ListEntity{ValueData: list.NewQuickList()}
	}

	listEntity, ok := store.items[key].(ListEntity)
	if !ok {
		return 0, false
	}

	return listEntity.ValueData.RPush(value), ok
}

func (store *Store) LPush(key string, value [][]byte) (int, bool) {

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.items[key] == nil {
		store.items[key] = ListEntity{ValueData: list.NewQuickList()}
	}

	listEntity, ok := store.items[key].(ListEntity)
	if !ok {
		return 0, false
	}

	return listEntity.ValueData.LPush(value), ok

}

func (store *Store) LRange(key string, startPos int, endPos int) ([][]byte, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if store.items[key] == nil {
		return [][]byte{}, true
	}

	listEntity, ok := store.items[key].(ListEntity)
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

	listEntity, ok := store.items[key].(ListEntity)
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

	listEntity, ok := store.items[key].(ListEntity)
	if !ok {
		return nil, false
	}

	return listEntity.ValueData.LPop(count), true
}
