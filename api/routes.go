package api

import "github.com/go-chi/chi/v5/middleware"

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
	s.router.Post("/create_user", s.createUserHandler)

	// Add a catch-all for 404s
	s.router.NotFound(s.notFoundHandler)
}
