// File: internal/router/router.go
package router

import (
	"net/http"

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

	// Apply global middleware
	router.Use(mw.Recovery)
	router.Use(mw.Logging)
	router.Use(middleware.Security) // Stateless middleware
	router.Use(mw.RateLimit)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   app.Config.CORS_Allowed_Origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	router.Use(c.Handler)

	// Public routes
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/auth", h.Auth).Methods("POST")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Protected routes
	protected := router.PathPrefix("/api/v1").Subrouter()
	protected.Use(mw.JWT)
	protected.HandleFunc("/protected", h.Protected).Methods("GET")

	return router
}
