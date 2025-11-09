package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/repository"
)

// AuditLogHandler handles audit log queries
type AuditLogHandler struct {
	logger    *logrus.Logger
	auditRepo *repository.AuditLogRepository
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(logger *logrus.Logger, auditRepo *repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{
		logger:    logger,
		auditRepo: auditRepo,
	}
}

// HandleList handles GET /admin/api/v1/audit-logs
func (h *AuditLogHandler) HandleList(c *gin.Context) {
	// Parse query parameters
	params, err := h.parseQueryParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_PARAMS",
				"message": err.Error(),
			},
		})
		return
	}

	// Query logs
	logs, total, err := h.auditRepo.Query(params)
	if err != nil {
		h.logger.Errorf("Failed to query audit logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "QUERY_FAILED",
				"message": "Failed to query audit logs",
			},
		})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"logs":      logs,
		"total":     total,
		"limit":     params.Limit,
		"offset":    params.Offset,
		"has_more":  params.Offset+params.Limit < total,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HandleGet handles GET /admin/api/v1/audit-logs/:id
func (h *AuditLogHandler) HandleGet(c *gin.Context) {
	id := c.Param("id")

	log, err := h.auditRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Audit log not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"log":       log,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HandleByCorrelation handles GET /admin/api/v1/audit-logs/correlation/:correlationId
func (h *AuditLogHandler) HandleByCorrelation(c *gin.Context) {
	correlationID := c.Param("correlationId")

	logs := h.auditRepo.GetByCorrelationID(correlationID)

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"correlation_id": correlationID,
		"logs":           logs,
		"count":          len(logs),
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

// HandleExport handles GET /admin/api/v1/audit-logs/export
func (h *AuditLogHandler) HandleExport(c *gin.Context) {
	// Parse query parameters
	params, err := h.parseQueryParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_PARAMS",
				"message": err.Error(),
			},
		})
		return
	}

	// Query logs
	logs, _, err := h.auditRepo.Query(params)
	if err != nil {
		h.logger.Errorf("Failed to query audit logs for export: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "QUERY_FAILED",
				"message": "Failed to query audit logs",
			},
		})
		return
	}

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=\"audit_logs.csv\"")

	// Write CSV header
	c.Writer.Write([]byte("ID,Timestamp,Action,Entity,Actor,Status,Message,Duration,IP Address\n"))

	// Write logs
	for _, log := range logs {
		line := h.logToCSV(log)
		c.Writer.Write([]byte(line + "\n"))
	}
}

// HandleStats handles GET /admin/api/v1/audit-logs/stats
func (h *AuditLogHandler) HandleStats(c *gin.Context) {
	stats, err := h.auditRepo.GetStats()
	if err != nil {
		h.logger.Errorf("Failed to get audit log stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "STATS_FAILED",
				"message": "Failed to get statistics",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"stats":     stats,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// parseQueryParams parses query parameters for audit log queries
func (h *AuditLogHandler) parseQueryParams(c *gin.Context) (repository.AuditLogQueryParams, error) {
	params := repository.AuditLogQueryParams{
		Limit:     50,
		Offset:    0,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err != nil {
			return params, fmt.Errorf("invalid start_time format: %s", startTimeStr)
		} else {
			params.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err != nil {
			return params, fmt.Errorf("invalid end_time format: %s", endTimeStr)
		} else {
			params.EndTime = endTime
		}
	}

	// Parse other filters
	params.Actor = c.Query("actor")
	params.Action = c.Query("action")
	params.Entity = c.Query("entity")
	params.Status = c.Query("status")
	params.Search = c.Query("search")

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err != nil {
			return params, fmt.Errorf("invalid limit: %s", limitStr)
		} else if limit < 0 || limit > 1000 {
			return params, fmt.Errorf("limit must be between 0 and 1000")
		} else {
			params.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err != nil {
			return params, fmt.Errorf("invalid offset: %s", offsetStr)
		} else if offset < 0 {
			return params, fmt.Errorf("offset must be non-negative")
		} else {
			params.Offset = offset
		}
	}

	// Parse sorting
	if sortBy := c.Query("sort_by"); sortBy != "" {
		validSortBy := map[string]bool{"timestamp": true, "action": true, "actor": true}
		if !validSortBy[sortBy] {
			return params, fmt.Errorf("invalid sort_by: %s", sortBy)
		}
		params.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		if sortOrder != "asc" && sortOrder != "desc" {
			return params, fmt.Errorf("invalid sort_order: %s (must be 'asc' or 'desc')", sortOrder)
		}
		params.SortOrder = sortOrder
	}

	return params, nil
}

// logToCSV converts an audit log to CSV format
func (h *AuditLogHandler) logToCSV(log repository.AuditLog) string {
	return h.csvEscape(log.ID) + "," +
		h.csvEscape(log.Timestamp.Format(time.RFC3339)) + "," +
		h.csvEscape(log.Action) + "," +
		h.csvEscape(log.Entity) + "," +
		h.csvEscape(log.Actor) + "," +
		h.csvEscape(log.Status) + "," +
		h.csvEscape(log.Message) + "," +
		h.csvEscape(strconv.FormatInt(log.Duration, 10)) + "," +
		h.csvEscape(log.IPAddress)
}

// csvEscape escapes a string for CSV output
func (h *AuditLogHandler) csvEscape(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
	}
	return value
}

// LogAction logs an action to the audit log
func (h *AuditLogHandler) LogAction(c *gin.Context, action, entity, entityID, status, message string, metadata map[string]interface{}) {
	log := repository.AuditLog{
		ID:            uuid.New().String(),
		Timestamp:     time.Now(),
		Level:         "info",
		Action:        action,
		Entity:        entity,
		EntityID:      entityID,
		Actor:         h.getActor(c),
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		Status:        status,
		Message:       message,
		Metadata:      metadata,
		CorrelationID: c.GetString("correlation_id"),
	}

	if err := h.auditRepo.Create(&log); err != nil {
		h.logger.Errorf("Failed to create audit log: %v", err)
	}
}

// getActor extracts the actor from the request context
func (h *AuditLogHandler) getActor(c *gin.Context) string {
	// Try to get user from context
	if user, exists := c.Get("user"); exists {
		if userMap, ok := user.(map[string]interface{}); ok {
			if username, ok := userMap["username"].(string); ok {
				return username
			}
		}
	}

	// Fall back to client IP
	return c.ClientIP()
}
