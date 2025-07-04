package middleware

import (
	"errors"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RoleType 定义角色类型
type RoleType string

const (
	RoleUser   RoleType = "user"
	RoleAdmin  RoleType = "admin"
	RoleSystem RoleType = "system"
)

const userContextKey = "user"

// AuthMiddleware 基础认证中间件
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

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminAuthMiddleware 管理员认证中间件
// 这个中间件必须在 AuthMiddleware 之后使用
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getUserFromContext(c)
		if err != nil {
			handleAuthError(c, err)
			return
		}

		// 检查用户状态
		if !validateUserStatus(user) {
			utils.Forbidden(c, "User account is not active")
			c.Abort()
			return
		}

		// 检查用户角色
		if !hasRole(user, RoleAdmin, RoleSystem) {
			utils.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}

		// 将完整的用户信息存储到上下文中，供后续使用
		setUserContext(c, user)
		c.Next()
	}
}

// SuperAdminMiddleware 超级管理员权限中间件（仅限system角色）
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getUserFromContext(c)
		if err != nil {
			handleAuthError(c, err)
			return
		}

		// 检查用户状态
		if !validateUserStatus(user) {
			utils.Forbidden(c, "User account is not active")
			c.Abort()
			return
		}

		// 只允许system角色访问
		if !hasRole(user, RoleSystem) {
			utils.Forbidden(c, "Super admin access required")
			c.Abort()
			return
		}

		// 将完整的用户信息存储到上下文中
		setUserContext(c, user)
		c.Next()
	}
}

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

// --- 辅助函数 ---

func getUserFromContext(c *gin.Context) (*models.User, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, errors.New("user not authenticated")
	}

	id, ok := userID.(uint)
	if !ok {
		return nil, errors.New("invalid user ID format in context")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func validateUserStatus(user *models.User) bool {
	return user.Status == "active"
}

func hasRole(user *models.User, roles ...RoleType) bool {
	for _, role := range roles {
		if RoleType(user.Role) == role {
			return true
		}
	}
	return false
}

func setUserContext(c *gin.Context, user *models.User) {
	c.Set(userContextKey, user)
}

func handleAuthError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Unauthorized(c, "User not authenticated")
	} else if errors.Is(err, gorm.ErrInvalidDB) {
		utils.InternalServerError(c, "Database connection not available")
	} else {
		utils.InternalServerError(c, "Failed to verify user: "+err.Error())
	}
	c.Abort()
}
