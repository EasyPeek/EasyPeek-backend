package api

import (
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(),
	}
}

// user register
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, err := h.userService.CreateUser(&req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, user.ToResponse())
}

// user login
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, token, err := h.userService.Login(&req)
	if err != nil {
		utils.Unauthorized(c, err.Error())
		return
	}

	response := gin.H{
		"user":  user.ToResponse(),
		"token": token,
	}

	utils.Success(c, response)
}

// GetProfile 获取当前用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, user.ToResponse())
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	// 更新用户信息
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := h.userService.UpdateUser(user); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, user.ToResponse())
}

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	// 验证旧密码
	if !user.CheckPassword(req.OldPassword) {
		utils.BadRequest(c, "Old password is incorrect")
		return
	}

	// 验证新密码格式
	if !utils.IsValidPassword(req.NewPassword) {
		utils.BadRequest(c, "New password must contain at least one letter and one number")
		return
	}

	// 更新密码
	user.Password = req.NewPassword
	if err := h.userService.UpdateUser(user); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "Password changed successfully"})
}

// GetUsers 获取用户列表（管理员功能）
func (h *UserHandler) GetUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	users, total, err := h.userService.GetAllUsers(page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	// 转换为响应格式
	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}

	utils.SuccessWithPagination(c, userResponses, total, page, size)
}

// GetUser 获取指定用户信息（管理员功能）
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, user.ToResponse())
}

// DeleteUser 删除用户（管理员功能）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "User deleted successfully"})
}
