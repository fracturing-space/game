package service

import "time"

// RecordClock provides journal timestamp metadata for stored events.
type RecordClock interface {
	Now() time.Time
}

type realRecordClock struct{}

func (realRecordClock) Now() time.Time { return time.Now().UTC() }
