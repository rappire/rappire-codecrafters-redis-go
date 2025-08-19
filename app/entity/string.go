package entity

import "time"

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
