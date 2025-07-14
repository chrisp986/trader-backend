package main

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// setupRoutes configures all the API routes
func (app *application) setupRoutes() {

	server := &Server{
		router:    chi.NewRouter(),
		startTime: time.Now(),
		version:   getVersion(),
	}

	// Add built-in Chi middleware
	server.router.Use(middleware.RequestID)
	server.router.Use(middleware.RealIP)
	server.router.Use(middleware.Recoverer)

	// Add custom logging middleware
	server.router.Use(server.loggingMiddleware)

	// Health check endpoint
	server.router.Get("/health", app.healthCheckHandler)
	server.router.Post("/create_user", app.createUserHandler)

	// Add a catch-all for 404s
	server.router.NotFound(app.notFoundHandler)
}
