package store

import "error-tracker/internal/model"

type TimeBucket struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type Store interface {
	SaveEvent(event *model.ErrorEvent) error
	SaveEvents(events []*model.ErrorEvent) error
	QueryEvents(appID, errorType string, page, size int) ([]*model.ErrorEvent, int64, error)
	GetEventStats(appID string) (*model.EventStats, error)
	GetTimeTrend(appID string, hours int) ([]*TimeBucket, error)
	Close() error
}
