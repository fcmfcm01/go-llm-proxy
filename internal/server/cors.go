package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"accept",
			"origin",
			"Cache-Control",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// StrictCORSConfig returns strict CORS configuration
func StrictCORSConfig(allowedOrigins []string) *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           1 * time.Hour,
	}
}

// SetupCORS sets up CORS middleware
func (s *Server) SetupCORS(config *CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Default()
	corsConfig.AllowOrigins = config.AllowOrigins
	corsConfig.AllowMethods = config.AllowMethods
	corsConfig.AllowHeaders = config.AllowHeaders
	corsConfig.ExposeHeaders = config.ExposeHeaders
	corsConfig.AllowCredentials = config.AllowCredentials
	corsConfig.MaxAge = config.MaxAge

	return corsConfig
}

// AddCORS adds CORS middleware to the router
func (s *Server) AddCORS(config *CORSConfig) {
	s.router.Use(s.SetupCORS(config))
	s.logger.Info("CORS middleware enabled")
}

// CORSOptions provides CORS configuration for different scenarios

// DevelopmentCORSConfig returns CORS config for development
func DevelopmentCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"http://localhost:5173",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"accept",
			"origin",
			"Cache-Control",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers",
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// ProductionCORSConfig returns CORS config for production
func ProductionCORSConfig(allowedOrigins []string) *CORSConfig {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"https://yourdomain.com"}
	}

	return &CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           1 * time.Hour,
	}
}

// AdminCORSConfig returns CORS config for admin endpoints
func AdminCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{
			"https://admin.yourdomain.com",
			"https://yourdomain.com/admin",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"X-Admin-Token",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Total-Count",
		},
		AllowCredentials: true,
		MaxAge:           30 * time.Minute,
	}
}
