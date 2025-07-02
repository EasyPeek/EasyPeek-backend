package api

import (
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// CommentHandler 结构体，用于封装与评论相关的 HTTP 请求处理逻辑
type CommentHandler struct {
	commentService *services.CommentService
}

// NewCommentHandler 创建并返回一个新的 CommentHandler 实例
func NewCommentHandler() *CommentHandler {
	return &CommentHandler{
		commentService: services.NewCommentService(),
	}
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req models.CommentCreateRequest
	// 将请求的 JSON 主体绑定到 CommentCreateRequest 结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	// 从 Gin 上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}
	// 将 userID 转换为 uint 类型
	creatorID, ok := userID.(uint)
	if !ok {
		utils.InternalServerError(c, "Failed to get user ID from context")
		return
	}

	// 调用 CommentService 的 CreateComment 方法来创建评论
	comment, err := h.commentService.CreateComment(&req, creatorID)
	if err != nil {
		// 根据错误类型返回不同的 HTTP 状态码
		if err.Error() == "database connection not initialized" {
			utils.InternalServerError(c, err.Error())
		} else if err.Error() == "news not found" {
			utils.NotFound(c, err.Error())
		} else if err.Error() == "user not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.BadRequest(c, err.Error())
		}
		return
	}

	// 成功创建，返回评论的响应格式
	utils.Success(c, comment.ToResponse())
}

// GetCommentByID 根据ID获取单条评论
func (h *CommentHandler) GetCommentByID(c *gin.Context) {
	// 从 URL 参数中获取评论ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid comment ID")
		return
	}

	// 调用 CommentService 的 GetCommentByID 方法
	comment, err := h.commentService.GetCommentByID(uint(id))
	if err != nil {
		if err.Error() == "comment not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	// 成功获取，返回评论的响应格式
	utils.Success(c, comment.ToResponse())
}

// GetCommentsByNewsID 根据新闻ID获取评论列表
func (h *CommentHandler) GetCommentsByNewsID(c *gin.Context) {
	// 从 URL 参数中获取新闻ID
	newsIDStr := c.Param("news_id")
	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		utils.BadRequest(c, "Invalid news ID")
		return
	}

	// 获取查询参数中的页码和每页大小，并设置默认值
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	// 转换页码和每页大小为整数，并处理无效值
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	// 调用 CommentService 的 GetCommentsByNewsID 方法
	comments, total, err := h.commentService.GetCommentsByNewsID(uint(newsID), page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var commentResponses []models.CommentResponse
	for _, comment := range comments {
		commentResponses = append(commentResponses, comment.ToResponse())
	}

	// 返回带分页信息成功的响应
	utils.SuccessWithPagination(c, commentResponses, total, page, size)
}

// GetCommentsByUserID 根据用户ID获取评论列表
func (h *CommentHandler) GetCommentsByUserID(c *gin.Context) {
	// 从 URL 参数中获取用户ID
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	// 获取查询参数中的页码和每页大小，并设置默认值
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	// 转换页码和每页大小为整数，并处理无效值
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	// 调用 CommentService 的 GetCommentsByUserID 方法
	comments, total, err := h.commentService.GetCommentsByUserID(uint(userID), page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var commentResponses []models.CommentResponse
	for _, comment := range comments {
		commentResponses = append(commentResponses, comment.ToResponse())
	}

	// 返回带分页信息成功的响应
	utils.SuccessWithPagination(c, commentResponses, total, page, size)
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	// 从 URL 参数中获取评论ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid comment ID")
		return
	}

	// 从 Gin 上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated")
		return
	}
	// 将 userID 转换为 uint 类型
	deleterID, ok := userID.(uint)
	if !ok {
		utils.InternalServerError(c, "Failed to get user ID from context")
		return
	}

	// 调用 CommentService 的 DeleteComment 方法进行软删除
	if err := h.commentService.DeleteComment(uint(id), deleterID); err != nil {
		if err.Error() == "comment not found or already deleted" {
			utils.NotFound(c, err.Error())
		} else if err.Error() == "permission denied: only comment author can delete" {
			utils.Forbidden(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	// 成功删除，返回成功消息
	utils.Success(c, gin.H{"message": "Comment deleted successfully"})
}

// AdminDeleteComment 管理员删除评论（硬删除）
func (h *CommentHandler) AdminDeleteComment(c *gin.Context) {
	// TODO: 管理员功能暂未实现，等待前端需求
	utils.InternalServerError(c, "Admin functionality not implemented yet")
}

// GetAllComments 获取所有评论（管理员功能）
func (h *CommentHandler) GetAllComments(c *gin.Context) {
	// TODO: 管理员功能暂未实现，等待前端需求
	utils.InternalServerError(c, "Admin functionality not implemented yet")
}
