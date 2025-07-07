package services

import (
	"errors"
	"log"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
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

// admin login
func (s *AdminService) AdminLogin(req *models.LoginRequest) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, "", err
	}

	// check admin role
	if user.Role != "admin" {
		return nil, "", errors.New("only admin users can login")
	}

	// check user status
	if user.Status != "active" {
		return nil, "", errors.New("admin account is not active")
	}

	// check password
	if !user.CheckPassword(req.Password) {
		return nil, "", errors.New("invalid password")
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

// User management
// GetAllUsers
func (s *AdminService) GetAllUsers(page, pageSize int) ([]models.User, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id ASC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetUserByID
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

// GetUserByUsername
func (s *AdminService) GetUserByUsername(username string) (*models.User, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail
func (s *AdminService) GetUserByEmail(email string) (*models.User, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUserInfo
func (s *AdminService) UpdateUserInfo(userID uint, updateData AdminUserUpdateRequest) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

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

// delete user
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

// GetSystemStats
func (s *AdminService) GetSystemStats() (*SystemStats, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	stats := &SystemStats{}

	// User statistics
	s.db.Model(&models.User{}).Where("status = ?", "active").Count(&stats.TotalActiveUsers)
	s.db.Model(&models.User{}).Where("status = ?", "deleted").Count(&stats.TotalDeletedUsers)
	s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&stats.TotalAdmins)

	s.db.Model(&models.Event{}).Count(&stats.TotalEvents)
	s.db.Model(&models.Event{}).Where("status = ?", "进行中").Count(&stats.ActiveEvents)

	// RSS statistics
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

// ClearAllEvents 清空所有事件
func (s *AdminService) ClearAllEvents() (int64, error) {
	if s.db == nil {
		return 0, errors.New("database connection not initialized")
	}

	// 使用事务确保数据一致性
	var deletedCount int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 首先计算要删除的事件数量
		var count int64
		if err := tx.Model(&models.Event{}).Count(&count).Error; err != nil {
			log.Printf("[ADMIN CLEAR] 获取事件数量失败: %v", err)
			return err
		}
		deletedCount = count
		log.Printf("[ADMIN CLEAR] 准备清空 %d 个事件", deletedCount)

		// 先清理相关的新闻关联（将belonged_event_id设置为NULL）
		// 这必须在删除事件之前进行，避免外键约束冲突
		var newsUpdateCount int64
		result := tx.Model(&models.News{}).Where("belonged_event_id IS NOT NULL").Update("belonged_event_id", nil)
		if result.Error != nil {
			log.Printf("[ADMIN CLEAR] 清理新闻关联失败: %v", result.Error)
			return result.Error
		}
		newsUpdateCount = result.RowsAffected
		log.Printf("[ADMIN CLEAR] 成功清理 %d 条新闻的事件关联", newsUpdateCount)

		// 然后删除所有事件（硬删除）
		if err := tx.Unscoped().Delete(&models.Event{}, "1=1").Error; err != nil {
			log.Printf("[ADMIN CLEAR] 删除事件失败: %v", err)
			return err
		}
		log.Printf("[ADMIN CLEAR] 成功删除所有事件")

		return nil
	})

	if err != nil {
		log.Printf("[ADMIN CLEAR ERROR] 清空事件事务失败: %v", err)
		return 0, err
	}

	log.Printf("[ADMIN CLEAR SUCCESS] 成功清空 %d 个事件", deletedCount)
	return deletedCount, nil
}

// ===== 数据类型定义 =====

type AdminUserUpdateRequest struct {
	Username  string `json:"username" binding:"omitempty,min=3,max=20"`
	Email     string `json:"email" binding:"omitempty,email"`
	Password  string `json:"password" binding:"omitempty,min=8"`
	Avatar    string `json:"avatar"`
	Phone     string `json:"phone" binding:"omitempty"`
	Location  string `json:"location" binding:"omitempty"`
	Bio       string `json:"bio" binding:"omitempty"`
	Interests string `json:"interests" binding:"omitempty"`
	Role      string `json:"role" binding:"omitempty,oneof=user admin system"`
	Status    string `json:"status" binding:"omitempty,oneof=active inactive suspended deleted"`
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
