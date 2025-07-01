package services

import (
	"errors"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService() *AdminService {
	return &AdminService{
		db: database.GetDB(),
	}
}

// ===== 用户管理相关 =====

// GetAllUsers 获取所有用户（包括已删除的）
func (s *AdminService) GetAllUsers(page, pageSize int, filter AdminUserFilter) ([]models.User, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	// 应用过滤条件
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetUserByID 获取指定用户详细信息
func (s *AdminService) GetUserByID(userID uint) (*models.User, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUserInfo 更新用户信息
func (s *AdminService) UpdateUserInfo(userID uint, updateData AdminUserUpdateRequest) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 检查新的用户名和邮箱是否已被其他用户使用
	if updateData.Username != "" && updateData.Username != user.Username {
		var existingUser models.User
		if err := s.db.Where("username = ? AND id != ?", updateData.Username, userID).First(&existingUser).Error; err == nil {
			return errors.New("username already exists")
		}
	}

	if updateData.Email != "" && updateData.Email != user.Email {
		var existingUser models.User
		if err := s.db.Where("email = ? AND id != ?", updateData.Email, userID).First(&existingUser).Error; err == nil {
			return errors.New("email already exists")
		}
	}

	// 更新用户信息
	updates := make(map[string]interface{})
	if updateData.Username != "" {
		updates["username"] = updateData.Username
	}
	if updateData.Email != "" {
		updates["email"] = updateData.Email
	}
	if updateData.Avatar != "" {
		updates["avatar"] = updateData.Avatar
	}
	if updateData.Role != "" {
		updates["role"] = updateData.Role
	}
	if updateData.Status != "" {
		updates["status"] = updateData.Status
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// DeleteUser 硬删除用户
func (s *AdminService) DeleteUser(userID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	result := s.db.Unscoped().Delete(&models.User{}, userID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// GetSystemStats 获取系统统计信息
func (s *AdminService) GetSystemStats() (*SystemStats, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	stats := &SystemStats{}

	// 用户统计
	s.db.Model(&models.User{}).Where("status = ?", "active").Count(&stats.TotalActiveUsers)
	s.db.Model(&models.User{}).Where("status = ?", "deleted").Count(&stats.TotalDeletedUsers)
	s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&stats.TotalAdmins)

	// 事件统计
	s.db.Model(&models.Event{}).Count(&stats.TotalEvents)
	s.db.Model(&models.Event{}).Where("status = ?", "进行中").Count(&stats.ActiveEvents)

	// RSS源统计
	s.db.Model(&models.RSSSource{}).Count(&stats.TotalRSSSources)
	s.db.Model(&models.RSSSource{}).Where("is_active = ?", true).Count(&stats.ActiveRSSSources)

	// 新闻统计
	s.db.Model(&models.News{}).Count(&stats.TotalNews)
	s.db.Model(&models.News{}).Where("is_active = ?", true).Count(&stats.ActiveNews)

	return stats, nil
}

// ===== 内容管理相关 =====

// GetAllEvents 获取所有事件（管理员视图）
func (s *AdminService) GetAllEvents(page, pageSize int, filter AdminEventFilter) ([]models.Event, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var events []models.Event
	var total int64

	query := s.db.Model(&models.Event{})

	// 应用过滤条件
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.CreatedBy != 0 {
		query = query.Where("created_by = ?", filter.CreatedBy)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// GetAllNews 获取所有新闻（管理员视图）
func (s *AdminService) GetAllNews(page, pageSize int, filter AdminNewsFilter) ([]models.News, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var news []models.News
	var total int64

	query := s.db.Model(&models.News{})

	// 应用过滤条件
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.SourceType != "" {
		query = query.Where("source_type = ?", filter.SourceType)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ? OR content ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&news).Error; err != nil {
		return nil, 0, err
	}

	return news, total, nil
}

// ===== 数据类型定义 =====

type AdminUserFilter struct {
	Role   string `form:"role"`
	Status string `form:"status"`
	Search string `form:"search"`
}

type AdminUserUpdateRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=20"`
	Email    string `json:"email" binding:"omitempty,email"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role" binding:"omitempty,oneof=user admin system"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive suspended deleted"`
}

type AdminEventFilter struct {
	Status    string `form:"status"`
	Category  string `form:"category"`
	CreatedBy uint   `form:"created_by"`
	Search    string `form:"search"`
}

type AdminNewsFilter struct {
	Status     string `form:"status"`
	Category   string `form:"category"`
	SourceType string `form:"source_type"`
	Search     string `form:"search"`
}

type SystemStats struct {
	TotalActiveUsers  int64 `json:"total_active_users"`
	TotalDeletedUsers int64 `json:"total_deleted_users"`
	TotalAdmins       int64 `json:"total_admins"`
	TotalEvents       int64 `json:"total_events"`
	ActiveEvents      int64 `json:"active_events"`
	TotalRSSSources   int64 `json:"total_rss_sources"`
	ActiveRSSSources  int64 `json:"active_rss_sources"`
	TotalNews         int64 `json:"total_news"`
	ActiveNews        int64 `json:"active_news"`
}
