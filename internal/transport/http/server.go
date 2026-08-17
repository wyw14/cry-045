package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/wyw14/cry045/internal/application"
	"github.com/wyw14/cry045/internal/domain"
	"github.com/wyw14/cry045/internal/middleware"
	"github.com/wyw14/cry045/internal/repository"
)

type Server struct {
	Engine   *gin.Engine
	service  *application.ComplianceService
	log      *zap.Logger
	validate *validator.Validate
}

type listPolicy struct {
	AllowedSorts    map[string]bool
	AllowedFilters  map[string]bool
	DefaultPageSize int
	MaxPageSize     int
}
type pageEnvelope struct {
	Items    []domain.Material `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int               `json:"total"`
	Sort     string            `json:"sort"`
	Filter   string            `json:"filter"`
}

var materialListPolicy = listPolicy{AllowedSorts: map[string]bool{"code": true, "name": true, "risk": true}, AllowedFilters: map[string]bool{"risk": true, "process": true}, DefaultPageSize: 20, MaxPageSize: 100}

func NewServer(service *application.ComplianceService, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	s := &Server{Engine: gin.New(), service: service, log: log, validate: validator.New()}
	s.Engine.Use(gin.Recovery(), middleware.RequestID())
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	s.Engine.GET("/readyz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ready"}) })
	v1 := s.Engine.Group("/api/v1")
	v1.GET("/materials", s.listMaterials)
	v1.GET("/selections/:id", s.getSelection)
	v1.POST("/selections/:id/validate", s.validateSelection)
	v1.POST("/selections/:id/submit", s.submit)
	v1.POST("/selections/:id/review", s.review)
	v1.POST("/selections/:id/return", s.returnSelection)
	v1.POST("/selections/:id/approve", s.approve)
	v1.GET("/selections/:id/timeline", s.timeline)
	v1.GET("/selections/:id/report", s.report)
}

func (s *Server) listMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(materialListPolicy.DefaultPageSize)))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > materialListPolicy.MaxPageSize {
		size = materialListPolicy.DefaultPageSize
	}
	sortKey, filterKey := c.DefaultQuery("sort", "code"), c.Query("filter")
	if !materialListPolicy.AllowedSorts[sortKey] {
		writeStableError(c, http.StatusBadRequest, "invalid_sort", "sort field is not allow-listed")
		return
	}
	if filterKey != "" && !materialListPolicy.AllowedFilters[filterKey] {
		writeStableError(c, http.StatusBadRequest, "invalid_filter", "filter field is not allow-listed")
		return
	}
	// The demo API intentionally returns a stable empty page; PostgreSQL adapters can supply the same envelope.
	c.JSON(http.StatusOK, pageEnvelope{Items: []domain.Material{}, Page: page, PageSize: size, Total: 0, Sort: sortKey, Filter: filterKey})
}

func (s *Server) getSelection(c *gin.Context) {
	sel, err := s.service.ValidateSelection(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, sel)
}

func (s *Server) validateSelection(c *gin.Context) {
	sel, err := s.service.ValidateSelection(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"findings": sel.Findings})
}
func (s *Server) submit(c *gin.Context) {
	if err := s.service.Submit(c.Request.Context(), c.Param("id"), c.GetHeader("X-Actor-ID")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": domain.StatusSubmitted})
}
func (s *Server) review(c *gin.Context) {
	second := c.DefaultQuery("stage", "first") == "second"
	if err := s.service.BeginReview(c.Request.Context(), c.Param("id"), c.GetHeader("X-Actor-ID"), second); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reviewing"})
}
func (s *Server) returnSelection(c *gin.Context) {
	if err := s.service.ReturnForRevision(c.Request.Context(), c.Param("id"), c.GetHeader("X-Actor-ID"), c.Query("comment")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": domain.StatusReturned})
}
func (s *Server) approve(c *gin.Context) {
	if err := s.service.Approve(c.Request.Context(), c.Param("id"), c.GetHeader("X-Actor-ID")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": domain.StatusApproved})
}
func (s *Server) timeline(c *gin.Context) {
	events, err := s.service.Timeline(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events})
}
func (s *Server) report(c *gin.Context) {
	body, err := s.service.ExportReport(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

func writeError(c *gin.Context, err error) {
	code, status := "internal_error", http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		code, status = "not_found", http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidState):
		code, status = "invalid_state", http.StatusConflict
	case errors.Is(err, domain.ErrExpiredEvidence), errors.Is(err, domain.ErrMissingEvidence):
		code, status = "validation_failed", http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": err.Error(), "fields": map[string]string{}, "request_id": c.GetString(middleware.RequestIDKey)}})
}

func writeStableError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "fields": map[string]string{}, "request_id": c.GetString(middleware.RequestIDKey)}})
}

func parsePositiveInt(raw string, fallback, maximum int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

var _ repository.Store

type TimelineView struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Before    string    `json:"before,omitempty"`
	After     string    `json:"after,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func prepareTimelineView(events []domain.AuditEvent, viewerRole string, from, to time.Time) []TimelineView {
	result := make([]TimelineView, 0, len(events))
	for _, event := range events {
		result = append(result, TimelineView{ID: event.ID, Action: event.Action, Actor: event.ActorID, Before: event.Before, After: event.After, CreatedAt: event.CreatedAt})
	}
	return result
}
