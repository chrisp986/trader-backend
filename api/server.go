package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Server holds the server configuration and dependencies
type Server struct {
	router    chi.Router
	logger    *zap.Logger
	startTime time.Time
	version   string
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

// NewServer creates a new server instance
func NewServer(logger *zap.Logger) *Server {

	server := &Server{
		router:    chi.NewRouter(),
		logger:    logger,
		startTime: time.Now(),
		version:   getVersion(),
	}

	server.setupRoutes()

	logger.Info("Trader backend version:", zap.String("version", server.version))

	return server
}

// getVersion returns the application version from environment or default
func getVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return "1.0.0"
	}
	return version
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
			// zap.String("user_agent", r.UserAgent()),
			// zap.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
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
