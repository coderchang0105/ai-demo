package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"error-tracker/internal/model"
	"error-tracker/internal/store"
)

type ReportHandler struct {
	store  store.Store
	buffer chan *model.ErrorEvent
}

func NewReportHandler(s store.Store) *ReportHandler {
	h := &ReportHandler{
		store:  s,
		buffer: make(chan *model.ErrorEvent, 1000),
	}
	go h.consumeBuffer()
	return h
}

func (h *ReportHandler) consumeBuffer() {
	batch := make([]*model.ErrorEvent, 0, 50)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := h.store.SaveEvents(batch); err != nil {
			log.Printf("[flush] SaveEvents error: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case event, ok := <-h.buffer:
			if !ok {
				flush()
				return
			}
			batch = append(batch, event)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (h *ReportHandler) Report(c *gin.Context) {
	contentType := c.ContentType()

	// support sendBeacon with text/plain
	if contentType == "text/plain" || contentType == "application/json" {
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}

		// try array first
		var events []*model.ErrorEvent
		if err := json.Unmarshal(body, &events); err == nil && len(events) > 0 {
			for _, e := range events {
				if e.AppID == "" || e.Type == "" || e.Message == "" {
					continue
				}
				if e.UserAgent == "" {
					e.UserAgent = c.GetHeader("User-Agent")
				}
				h.buffer <- e
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok", "count": len(events)})
			return
		}

		// try single object
		var event model.ErrorEvent
		if err := json.Unmarshal(body, &event); err == nil {
			if event.AppID == "" || event.Type == "" || event.Message == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "app_id, type, message are required"})
				return
			}
			if event.UserAgent == "" {
				event.UserAgent = c.GetHeader("User-Agent")
			}
			h.buffer <- &event
			c.JSON(http.StatusOK, gin.H{"status": "ok", "count": 1})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported content type"})
}

func (h *ReportHandler) GetEvents(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	errorType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	events, total, err := h.store.QueryEvents(appID, errorType, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  events,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (h *ReportHandler) GetStats(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	stats, err := h.store.GetEventStats(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ReportHandler) GetTrend(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	buckets, err := h.store.GetTimeTrend(appID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": buckets})
}
