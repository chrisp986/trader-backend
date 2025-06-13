package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// Server holds the server configuration and dependencies
type Server struct {
	router    chi.Router
	logger    *zap.Logger
	startTime time.Time
	version   string
}

// NewServer creates a new server instance
func NewServer() *Server {
	logger := newLogger()

	server := &Server{
		router:    chi.NewRouter(),
		logger:    logger,
		startTime: time.Now(),
		version:   getVersion(),
	}

	server.setupRoutes()
	return server
}

// newLogger creates a new zap logger with structured JSON output
func newLogger() *zap.Logger {
	// Get log level from environment variable or default to INFO
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		// Create a temporary logger to log the warning
		tempLogger, _ := zap.NewProduction()
		tempLogger.Warn("Invalid log level, defaulting to INFO", zap.String("provided_level", logLevel), zap.Error(err))
		tempLogger.Sync()
		level = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	return logger
}

// getVersion returns the application version from environment or default
func getVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return "1.0.0"
	}
	return version
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() {
	// Add built-in Chi middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)

	// Add custom logging middleware
	s.router.Use(s.loggingMiddleware)

	// Health check endpoint
	s.router.Get("/health", s.healthCheckHandler)

	// Add a catch-all for 404s
	s.router.NotFound(s.notFoundHandler)
}

// loggingMiddleware logs all incoming requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		s.logger.Info("HTTP request processed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status_code", wrapped.statusCode),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// healthCheckHandler handles the health check endpoint
func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   s.version,
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode health check response", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Health check requested",
		zap.String("status", response.Status),
		zap.String("version", response.Version),
		zap.String("uptime", response.Uptime),
	)
}

// notFoundHandler handles 404 errors
func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Warn("Route not found",
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

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

func main() {
	server := NewServer()

	// Ensure logger is properly closed on exit
	defer server.logger.Sync()

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port

	server.logger.Info("Application starting",
		zap.String("version", server.version),
		zap.String("port", port),
	)

	if err := server.Start(addr); err != nil {
		server.logger.Fatal("Failed to start server", zap.Error(err))
	}
}
