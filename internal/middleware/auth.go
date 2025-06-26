package middleware

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Missing authorization header")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(authHeader)
		if err != nil {
			utils.Unauthorized(c, "Invalid token: "+err.Error())
			c.Abort()
			return
		}

		// 将用户信息存储到context中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.Forbidden(c, "No role information")
			c.Abort()
			return
		}

		if role != "admin" {
			utils.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
