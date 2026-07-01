package model

import "time"

type LiveInfo struct {
	ID                 string
	Title              string
	URL                string
	Status             string
	ScheduledStartTime time.Time
}

const (
	StatusUpcoming = "upcoming"
	StatusLive     = "live"
	StatusNone     = "none"
)
