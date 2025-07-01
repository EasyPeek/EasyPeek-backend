package middleware

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// RoleType 定义角色类型
type RoleType string

const (
	RoleUser   RoleType = "user"
	RoleAdmin  RoleType = "admin"
	RoleSystem RoleType = "system"
)

// RoleMiddleware 角色权限中间件
func RoleMiddleware(allowedRoles ...RoleType) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.Forbidden(c, "No role information found")
			c.Abort()
			return
		}

		userRole := RoleType(role.(string))

		// 检查用户角色是否在允许的角色列表中
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		utils.Forbidden(c, "Insufficient permissions for this operation")
		c.Abort()
	}
}

// RequireAdmin 需要管理员权限的中间件（向后兼容）
func RequireAdmin() gin.HandlerFunc {
	return RoleMiddleware(RoleAdmin)
}

// RequireSystemOrAdmin 需要系统权限或管理员权限的中间件
func RequireSystemOrAdmin() gin.HandlerFunc {
	return RoleMiddleware(RoleSystem, RoleAdmin)
}

// RequireAnyRole 需要任意用户权限的中间件
func RequireAnyRole() gin.HandlerFunc {
	return RoleMiddleware(RoleUser, RoleAdmin, RoleSystem)
}

// PermissionMiddleware 更灵活的权限中间件，支持自定义权限验证函数
func PermissionMiddleware(permissionCheck func(userRole string, c *gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.Forbidden(c, "No role information found")
			c.Abort()
			return
		}

		userRole := role.(string)

		if !permissionCheck(userRole, c) {
			utils.Forbidden(c, "Insufficient permissions for this operation")
			c.Abort()
			return
		}

		c.Next()
	}
}

// ResourceOwnerOrAdmin 资源所有者或管理员权限中间件
// 检查用户是否为资源所有者或管理员
func ResourceOwnerOrAdmin(getResourceOwnerID func(*gin.Context) (uint, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.Unauthorized(c, "User not authenticated")
			c.Abort()
			return
		}

		role, exists := c.Get("role")
		if !exists {
			utils.Forbidden(c, "No role information found")
			c.Abort()
			return
		}

		// 如果是管理员，直接通过
		if role.(string) == string(RoleAdmin) || role.(string) == string(RoleSystem) {
			c.Next()
			return
		}

		// 检查是否为资源所有者
		resourceOwnerID, err := getResourceOwnerID(c)
		if err != nil {
			utils.BadRequest(c, "Unable to verify resource ownership")
			c.Abort()
			return
		}

		if userID.(uint) != resourceOwnerID {
			utils.Forbidden(c, "You can only access your own resources")
			c.Abort()
			return
		}

		c.Next()
	}
}
