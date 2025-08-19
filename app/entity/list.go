package entity

import "github.com/codecrafters-io/redis-starter-go/app/entity/list"

type ListEntity struct {
	ValueData *list.QuickList
	notify    chan struct{}
}

func (l ListEntity) Expired() bool {
	return false
}

func (l ListEntity) Notify() chan struct{} {
	return l.notify
}

func NewListEntity() *ListEntity {
	return &ListEntity{
		ValueData: list.NewQuickList(),
		notify:    make(chan struct{}, 1),
	}
}
