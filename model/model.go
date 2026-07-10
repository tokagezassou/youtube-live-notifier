package model

import "time"

type LiveInfo struct {
	ID                 string
	Title              string
	URL                string
	Status             string
	ScheduledStartTime time.Time
	LifeCycleStatus    string
	PrivacyStatus      string
}

const (
	StatusUpcoming  = "upcoming"
	StatusLive      = "live"
	StatusCompleted = "completed"
	StatusPublic    = "public"
)
