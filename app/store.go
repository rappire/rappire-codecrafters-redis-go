package main

import (
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

type ListEntity struct {
	ValueDate []StringEntity
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

	valueCount := len(value)
	stringValues := make([]StringEntity, valueCount)
	for i, v := range value {
		stringValues[i] = StringEntity{ValueData: string(v)}
	}

	if store.items[key] == nil {
		store.mu.Lock()
		defer store.mu.Unlock()
		store.items[key] = ListEntity{ValueDate: stringValues}
		return valueCount, true
	}

	listEntity, ok := store.items[key].(ListEntity)
	if !ok {
		return 0, false
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	for _, v := range stringValues {
		listEntity.ValueDate = append(listEntity.ValueDate, StringEntity{ValueData: v.Value()})
	}
	store.items[key] = listEntity
	return len(listEntity.ValueDate), true
}

func (store *Store) LRange(key string, startPos int, endPos int) ([][]byte, bool) {
	if startPos > endPos {
		return [][]byte{}, true
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	if store.items[key] == nil {
		return [][]byte{}, true
	}

	listEntity, ok := store.items[key].(ListEntity)
	if !ok {
		return nil, false
	}

	length := len(listEntity.ValueDate)
	if startPos >= length {
		return [][]byte{}, false
	}

	if endPos >= length {
		endPos = length - 1
	}

	byteValues := make([][]byte, endPos-startPos+1)
	for i, v := range listEntity.ValueDate[startPos : endPos+1] {
		byteValues[i] = []byte(v.ValueData)
	}

	return byteValues, true
}
