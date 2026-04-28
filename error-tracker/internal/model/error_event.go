package model

type ErrorEvent struct {
	ID        int64  `json:"id"`
	AppID     string `json:"app_id" binding:"required"`
	Type      string `json:"type" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Stack     string `json:"stack"`
	URL       string `json:"url"`
	UserAgent string `json:"user_agent"`
	UserID    string `json:"user_id"`
	Meta      string `json:"meta"`
	Timestamp int64  `json:"timestamp"`
	CreatedAt int64  `json:"created_at"`
}

type EventStats struct {
	Total   int64            `json:"total"`
	ByType  map[string]int64 `json:"by_type"`
	AppID   string           `json:"app_id"`
}
