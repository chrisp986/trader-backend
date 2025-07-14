package main

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthResponse represents the health check response structure
type HttpResponse struct {
	HttpStatusCode int       `json:"http_status_code"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
	Version        string    `json:"version"`
	Uptime         string    `json:"uptime"`
}

// healthCheckHandler handles the health check endpoint
func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	// uptime := time.Since(s.startTime)

	response := HttpResponse{
		HttpStatusCode: http.StatusOK,
		Status:         "New user created",
		Timestamp:      time.Now(),
		// Version:        s.version,
		// Uptime:         uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.logger.Error("Failed to encode health check response", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	app.logger.Debug("Create user route",
		zap.Int("status_code", response.HttpStatusCode),
		zap.String("status", response.Status),
		zap.String("version", response.Version),
		zap.String("uptime", response.Uptime),
	)
}

// healthCheckHandler handles the health check endpoint
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// uptime := time.Since(s.startTime)

	response := HttpResponse{
		HttpStatusCode: http.StatusOK,
		Status:         "healthy",
		Timestamp:      time.Now(),
		// Version:        s.version,
		// Uptime:         uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.logger.Error("Failed to encode health check response", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	app.logger.Debug("Health check requested",
		zap.Int("status_code", response.HttpStatusCode),
		zap.String("status", response.Status),
		zap.String("version", response.Version),
		zap.String("uptime", response.Uptime),
	)
}

// notFoundHandler handles 404 errors
func (app *application) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	app.logger.Warn("Route not found",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	response := map[string]string{
		"error":   "Not Found",
		"message": "The requested resource was not found",
	}

	json.NewEncoder(w).Encode(response)
}
