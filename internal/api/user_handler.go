package api

import (
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

// user logout
func (h *UserHandler) Logout(c *gin.Context) {
	utils.Success(c, gin.H{"message": "User logged out successfully"})
}

// user getProfile
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

// user updateProfile
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

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Location != "" {
		user.Location = req.Location
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}
	if req.Interests != "" {
		user.Interests = req.Interests
	}

	if err := h.userService.UpdateUser(user); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, user.ToResponse())
}

// user changePassword
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

	if !user.CheckPassword(req.OldPassword) {
		utils.BadRequest(c, "Old password is incorrect")
		return
	}

	if !utils.IsValidPassword(req.NewPassword) {
		utils.BadRequest(c, "New password must contain at least one letter and one number")
		return
	}

	user.Password = req.NewPassword
	if err := h.userService.UpdateUser(user); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "Password changed successfully"})
}

// DeleteSelf
func (h *UserHandler) DeleteSelf(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	if !user.CheckPassword(req.Password) {
		utils.BadRequest(c, "Password is incorrect")
		return
	}

	if err := h.userService.SoftDeleteUser(userID.(uint)); err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message": "Account deleted successfully",
	})
}
