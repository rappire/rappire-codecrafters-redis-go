package main

import "sync"

type Store struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

func (store *Store) Get(key string) string {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.data[key]
}

func (store *Store) Set(key, value string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.data[key] = value
}
