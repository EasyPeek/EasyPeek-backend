package middleware

import (
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// cors middleware
// todo
func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
