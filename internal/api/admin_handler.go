package api

import (
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *services.AdminService
	userService  *services.UserService
	eventService *services.EventService
	newsService  *services.NewsService
	rssService   *services.RSSService
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		adminService: services.NewAdminService(),
		userService:  services.NewUserService(),
		eventService: services.NewEventService(),
		newsService:  services.NewNewsService(),
		rssService:   services.NewRSSService(),
	}
}

func (h *AdminHandler) AdminLogin(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	user, token, err := h.adminService.AdminLogin(&req)
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

// GetSystemStats
func (h *AdminHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.adminService.GetSystemStats()
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, stats)
}

// user
// GetAllUsers
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
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

	// 绑定过滤参数
	var filter services.AdminUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "Invalid filter parameters")
		return
	}

	users, total, err := h.adminService.GetAllUsers(page, size, filter)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	// 转换为响应格式
	var userResponses []interface{}
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}

	utils.SuccessWithPagination(c, userResponses, total, page, size)
}

// GetUserByID 获取指定用户详细信息
func (h *AdminHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.adminService.GetUserByID(uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	utils.Success(c, user.ToResponse())
}

// UpdateUser 更新用户信息
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	var req services.AdminUserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	if err := h.adminService.UpdateUserInfo(uint(id), req); err != nil {
		if err.Error() == "user not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.BadRequest(c, err.Error())
		}
		return
	}

	utils.Success(c, gin.H{"message": "User updated successfully"})
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.adminService.DeleteUser(uint(id)); err != nil {
		if err.Error() == "user not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	utils.Success(c, gin.H{"message": "User deleted successfully"})
}

// ===== 事件管理 =====

// GetAllEvents 获取所有事件（管理员视图）
func (h *AdminHandler) GetAllEvents(c *gin.Context) {
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

	// 绑定过滤参数
	var filter services.AdminEventFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "Invalid filter parameters")
		return
	}

	events, total, err := h.adminService.GetAllEvents(page, size, filter)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	// 转换为响应格式
	var eventResponses []interface{}
	for _, event := range events {
		eventResponses = append(eventResponses, event)
	}

	utils.SuccessWithPagination(c, eventResponses, total, page, size)
}

// UpdateEvent 更新事件（管理员）
func (h *AdminHandler) UpdateEvent(c *gin.Context) {
	// 复用现有的 EventHandler 的 UpdateEvent 方法
	eventHandler := NewEventHandler()
	eventHandler.UpdateEvent(c)
}

// DeleteEvent 删除事件（管理员）
func (h *AdminHandler) DeleteEvent(c *gin.Context) {
	eventHandler := NewEventHandler()
	eventHandler.DeleteEvent(c)
}

// ===== 新闻管理 =====

// GetAllNews 获取所有新闻（管理员视图）
func (h *AdminHandler) GetAllNews(c *gin.Context) {
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

	// 绑定过滤参数
	var filter services.AdminNewsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "Invalid filter parameters")
		return
	}

	news, total, err := h.adminService.GetAllNews(page, size, filter)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	// 转换为响应格式
	var newsResponses []interface{}
	for _, item := range news {
		newsResponses = append(newsResponses, item.ToResponse())
	}

	utils.SuccessWithPagination(c, newsResponses, total, page, size)
}

// UpdateNews 更新新闻（管理员）
func (h *AdminHandler) UpdateNews(c *gin.Context) {
	// 复用现有的新闻更新逻辑
	newsHandler := NewNewsHandler()
	newsHandler.UpdateNews(c)
}

// DeleteNews 删除新闻（管理员）
func (h *AdminHandler) DeleteNews(c *gin.Context) {
	// 复用现有的新闻删除逻辑
	newsHandler := NewNewsHandler()
	newsHandler.DeleteNews(c)
}
