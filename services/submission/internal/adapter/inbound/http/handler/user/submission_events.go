package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubmissionEventsHandler struct {
	snapshotRepo      outbound.SubmissionStreamSnapshotRepository
	ticketService     outbound.SubmissionStreamTicketService
	eventHub          outbound.SubmissionEventHub
	heartbeatInterval time.Duration
	allowedOrigin     string
	logger            *zap.Logger
}

func NewSubmissionEventsHandler(
	snapshotRepo outbound.SubmissionStreamSnapshotRepository,
	ticketService outbound.SubmissionStreamTicketService,
	eventHub outbound.SubmissionEventHub,
	sseCfg config.SSEConfig,
	logger *zap.Logger,
) *SubmissionEventsHandler {
	return &SubmissionEventsHandler{
		snapshotRepo:      snapshotRepo,
		ticketService:     ticketService,
		eventHub:          eventHub,
		heartbeatInterval: sseCfg.HeartbeatInterval,
		allowedOrigin:     strings.TrimSpace(sseCfg.AllowedOrigin),
		logger:            logger,
	}
}

func (h *SubmissionEventsHandler) Handle(c *gin.Context) {
	submissionID, ok := parsePositiveInt64(c.Param("submission_id"))
	if !ok {
		response.HandleError(c, domain.ErrInvalidSubmissionID)
		return
	}

	if !h.applyCORS(c) {
		response.HandleError(c, domain.ErrSubmissionForbidden)
		return
	}

	claims, err := h.ticketService.Verify(c.Query("ticket"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if claims.SubmissionID != submissionID {
		response.HandleError(c, domain.ErrInvalidStreamTicket)
		return
	}

	events, unsubscribe := h.eventHub.Subscribe(submissionID)
	defer unsubscribe()

	snapshot, err := h.snapshotRepo.GetStreamSnapshot(c.Request.Context(), submissionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubmissionNotFound):
			response.HandleError(c, err)
			return
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			response.HandleError(c, err)
			return
		default:
			response.HandleError(c, domain.ErrInternalServer.Wrap(err))
			return
		}
	}
	if snapshot == nil {
		response.HandleError(c, domain.ErrInternalServer)
		return
	}
	if snapshot.UserID != claims.UserID {
		response.HandleError(c, domain.ErrSubmissionNotFound)
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.HandleError(c, domain.ErrSubmissionStreamUnsupported)
		return
	}

	headers := c.Writer.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache, no-transform")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")

	if err := writeRawSSE(c.Writer, "retry: 3000\n\n"); err != nil {
		h.logStreamWriteError(submissionID, err)
		return
	}
	flusher.Flush()

	currentAttemptID := snapshot.AttemptID
	snapshotEvent := snapshot.Event()
	snapshotName := "submission.snapshot"
	if entity.IsTerminalStatus(snapshot.Status) {
		snapshotName = "submission.completed"
	}
	if err := writeSSEEvent(c.Writer, snapshotName, snapshotEvent); err != nil {
		h.logStreamWriteError(submissionID, err)
		return
	}
	flusher.Flush()
	if entity.IsTerminalStatus(snapshot.Status) {
		return
	}

	heartbeat := time.NewTicker(h.heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			if err := writeRawSSE(c.Writer, ": heartbeat\n\n"); err != nil {
				h.logStreamWriteError(submissionID, err)
				return
			}
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.SubmissionID != submissionID || event.AttemptID != currentAttemptID {
				continue
			}
			eventName := "submission.updated"
			if status, ok := entity.ParseStatus(event.Status); ok && entity.IsTerminalStatus(status) {
				eventName = "submission.completed"
			}
			if err := writeSSEEvent(c.Writer, eventName, event); err != nil {
				h.logStreamWriteError(submissionID, err)
				return
			}
			flusher.Flush()
			if eventName == "submission.completed" {
				return
			}
		}
	}
}

func (h *SubmissionEventsHandler) applyCORS(c *gin.Context) bool {
	if h.allowedOrigin == "" {
		return true
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		return true
	}
	if origin != h.allowedOrigin {
		return false
	}
	c.Writer.Header().Set("Access-Control-Allow-Origin", h.allowedOrigin)
	c.Writer.Header().Set("Vary", "Origin")
	return true
}

func (h *SubmissionEventsHandler) logStreamWriteError(submissionID int64, err error) {
	if h.logger == nil {
		return
	}
	h.logger.Debug(
		"submission SSE stream write failed",
		zap.Int64("submission_id", submissionID),
		zap.Error(err),
	)
}

func parsePositiveInt64(raw string) (int64, bool) {
	value, err := strconv.ParseInt(raw, 10, 64)
	return value, err == nil && value > 0
}

func writeSSEEvent(w http.ResponseWriter, eventName string, event entity.SubmissionEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal submission SSE event: %w", err)
	}
	id := fmt.Sprintf("%d:%s:%s", event.SubmissionID, event.AttemptID, event.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err := writeRawSSE(w, "event: "+eventName+"\n"); err != nil {
		return err
	}
	if err := writeRawSSE(w, "id: "+id+"\n"); err != nil {
		return err
	}
	if err := writeRawSSE(w, "data: "+string(payload)+"\n\n"); err != nil {
		return err
	}
	return nil
}

func writeRawSSE(w http.ResponseWriter, data string) error {
	_, err := w.Write([]byte(data))
	return err
}
