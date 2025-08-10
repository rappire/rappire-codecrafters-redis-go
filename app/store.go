package main

import (
	"sync"
	"time"
)

type Store struct {
	items map[string]Entry
	mu    sync.RWMutex
}

type Entry struct {
	Value  string
	Expire time.Time
}

func NewStore() *Store {
	return &Store{items: make(map[string]Entry)}
}

func (store *Store) Get(key string) (string, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	entry, ok := store.items[key]
	if !ok {
		return "", false
	}

	if !entry.Expire.IsZero() && time.Now().After(entry.Expire) {
		store.mu.RUnlock()
		store.mu.Lock()
		delete(store.items, key)
		store.mu.Unlock()
		store.mu.RLock()
		return "", false
	}

	return entry.Value, true
}

func (store *Store) Set(key, value string, expire time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := Entry{Value: value, Expire: expire}
	store.items[key] = entry
}
