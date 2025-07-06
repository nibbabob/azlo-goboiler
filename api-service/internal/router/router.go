// File: internal/router/router.go
package router

import (
	"net/http"
	"time"

	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/handlers"
	"azlo-goboiler/internal/middleware"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func Setup(app *config.Application) http.Handler {
	router := mux.NewRouter()

	// Create instances of handlers and middleware
	h := handlers.New(app)
	mw := middleware.New(app)

	// Apply global middleware in order of execution
	router.Use(mw.RequestID)                 // First: Add request ID
	router.Use(mw.Recovery)                  // Second: Catch panics
	router.Use(mw.Logging)                   // Third: Log requests
	router.Use(middleware.Security)          // Fourth: Security headers
	router.Use(mw.Timeout(30 * time.Second)) // Fifth: Request timeout
	router.Use(mw.RateLimit)                 // Sixth: Rate limiting

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   app.Config.CORS_Allowed_Origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})
	router.Use(c.Handler)

	// Health and monitoring routes (no authentication required)
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/health/detailed", h.HealthDetailed).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Public authentication routes
	auth := router.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", h.Register).Methods("POST")
	auth.HandleFunc("/login", h.Auth).Methods("POST")

	// Protected API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(mw.JWT) // JWT authentication required for all /api/v1 routes

	// User management routes
	api.HandleFunc("/profile", h.GetProfile).Methods("GET")
	api.HandleFunc("/profile", h.UpdateProfile).Methods("PUT")
	api.HandleFunc("/password", h.ChangePassword).Methods("PUT")
	api.HandleFunc("/users", h.GetUsers).Methods("GET")

	// Example protected route
	api.HandleFunc("/protected", h.Protected).Methods("GET")

	// Database statistics route (admin only in production)
	api.HandleFunc("/admin/db-stats", h.GetDatabaseStats).Methods("GET")

	return router
}
