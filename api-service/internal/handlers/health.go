package handlers

import (
	"azlo-goboiler/internal/database"
	"context"
	"net/http"
	"time"
)

// Health handles health check requests with enhanced diagnostics
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	var dbLatency time.Duration
	dbStart := time.Now()
	if err := h.app.DB.Ping(healthCtx); err != nil {
		dbStatus = "disconnected"
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Database health check failed")
	} else {
		dbLatency = time.Since(dbStart)
	}

	redisStatus := "connected"
	var redisLatency time.Duration
	redisStart := time.Now()
	if _, err := h.app.Redis.Ping(healthCtx).Result(); err != nil {
		redisStatus = "disconnected"
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Redis health check failed")
	} else {
		redisLatency = time.Since(redisStart)
	}

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      time.Since(startTime).String(),
		"version":     "1.4.0",
		"environment": h.app.Config.App_Env,
		"request_id":  requestID,
		"services": map[string]interface{}{
			"database": map[string]interface{}{
				"status":  dbStatus,
				"latency": dbLatency.String(),
			},
			"redis": map[string]interface{}{
				"status":  redisStatus,
				"latency": redisLatency.String(),
			},
		},
	}

	if dbStatus == "disconnected" || redisStatus == "disconnected" {
		health["status"] = "degraded"
		writeResponse(w, h.app, http.StatusServiceUnavailable, false, health, "Service is degraded")
		return
	}

	writeSuccess(w, h.app, health, "Service is healthy")
}

// HealthDetailed provides detailed health information including database stats
func (h *Handlers) HealthDetailed(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	healthCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      time.Since(startTime).String(),
		"version":     "1.4.0",
		"environment": h.app.Config.App_Env,
		"request_id":  requestID,
	}

	// Database health
	dbHealth := make(map[string]interface{})
	dbStart := time.Now()
	if err := database.HealthCheck(h.app.DB); err != nil {
		dbHealth["status"] = "unhealthy"
		dbHealth["error"] = err.Error()
		health["status"] = "degraded"
	} else {
		dbHealth["status"] = "healthy"
		dbHealth["latency"] = time.Since(dbStart).String()
		dbHealth["stats"] = database.GetConnectionStats(h.app.DB)
	}
	health["database"] = dbHealth

	// Redis health
	redisHealth := make(map[string]interface{})
	redisStart := time.Now()
	if _, err := h.app.Redis.Ping(healthCtx).Result(); err != nil {
		redisHealth["status"] = "unhealthy"
		redisHealth["error"] = err.Error()
		health["status"] = "degraded"
	} else {
		redisHealth["status"] = "healthy"
		redisHealth["latency"] = time.Since(redisStart).String()
	}
	health["redis"] = redisHealth

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	writeResponse(w, h.app, statusCode, health["status"] == "healthy", health, "Detailed health check complete")
}

// GetDatabaseStats retrieves DB connection info
// @Summary      Database Statistics
// @Description  Get internal database connection pool stats
// @Tags         admin
// @Security     Bearer
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/admin/db-stats [get]
func (h *Handlers) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	// In production, you might want to add admin role checking here
	stats := database.GetConnectionStats(h.app.DB)
	writeSuccess(w, h.app, stats, "Database statistics retrieved")
}
