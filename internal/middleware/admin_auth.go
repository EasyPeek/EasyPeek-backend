package middleware

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminAuthMiddleware 管理员认证中间件
// 这个中间件必须在 AuthMiddleware 之后使用
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先检查用户是否已通过基础认证
		userID, exists := c.Get("user_id")
		if !exists {
			utils.Unauthorized(c, "User not authenticated")
			c.Abort()
			return
		}

		// 从数据库查询用户信息以验证角色
		db := database.GetDB()
		if db == nil {
			utils.InternalServerError(c, "Database connection not available")
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, userID.(uint)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Unauthorized(c, "User not found")
			} else {
				utils.InternalServerError(c, "Failed to verify user")
			}
			c.Abort()
			return
		}

		// 检查用户状态
		if user.Status != "active" {
			utils.Forbidden(c, "User account is not active")
			c.Abort()
			return
		}

		// 检查用户角色
		if user.Role != "admin" && user.Role != "system" {
			utils.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}

		// 将完整的用户信息存储到上下文中，供后续使用
		c.Set("admin_user", user)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

// SuperAdminMiddleware 超级管理员权限中间件（仅限system角色）
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先检查用户是否已通过基础认证
		userID, exists := c.Get("user_id")
		if !exists {
			utils.Unauthorized(c, "User not authenticated")
			c.Abort()
			return
		}

		// 从数据库查询用户信息以验证角色
		db := database.GetDB()
		if db == nil {
			utils.InternalServerError(c, "Database connection not available")
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, userID.(uint)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Unauthorized(c, "User not found")
			} else {
				utils.InternalServerError(c, "Failed to verify user")
			}
			c.Abort()
			return
		}

		// 检查用户状态
		if user.Status != "active" {
			utils.Forbidden(c, "User account is not active")
			c.Abort()
			return
		}

		// 只允许system角色访问
		if user.Role != "system" {
			utils.Forbidden(c, "Super admin access required")
			c.Abort()
			return
		}

		// 将完整的用户信息存储到上下文中
		c.Set("admin_user", user)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

// OwnerOrAdminMiddleware 资源所有者或管理员权限中间件
// 允许资源所有者或管理员访问
func OwnerOrAdminMiddleware(getOwnerID func(*gin.Context) (uint, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.Unauthorized(c, "User not authenticated")
			c.Abort()
			return
		}

		// 获取资源所有者ID
		ownerID, err := getOwnerID(c)
		if err != nil {
			utils.BadRequest(c, "Unable to determine resource owner")
			c.Abort()
			return
		}

		// 如果是资源所有者，直接允许访问
		if userID.(uint) == ownerID {
			c.Next()
			return
		}

		// 如果不是所有者，检查是否为管理员
		db := database.GetDB()
		if db == nil {
			utils.InternalServerError(c, "Database connection not available")
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, userID.(uint)).Error; err != nil {
			utils.InternalServerError(c, "Failed to verify user")
			c.Abort()
			return
		}

		// 检查是否为管理员或系统管理员
		if user.Role != "admin" && user.Role != "system" {
			utils.Forbidden(c, "Access denied: not owner or admin")
			c.Abort()
			return
		}

		// 检查用户状态
		if user.Status != "active" {
			utils.Forbidden(c, "User account is not active")
			c.Abort()
			return
		}

		c.Set("admin_user", user)
		c.Set("user_role", user.Role)
		c.Next()
	}
}
