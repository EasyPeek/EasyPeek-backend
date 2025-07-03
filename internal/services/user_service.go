package services

import (
	"errors"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

// create user service instance
func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

// user register
func (s *UserService) CreateUser(req *models.RegisterRequest) (*models.User, error) {
	// check if username already exists
	var existingUser models.User
	if err := s.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	// check if email already exists
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already exists")
	}

	// validate input
	if !utils.IsValidUsername(req.Username) {
		return nil, errors.New("invalid username format")
	}

	if !utils.IsValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}

	if !utils.IsValidPassword(req.Password) {
		return nil, errors.New("password must contain at least one letter and one number")
	}

	// create user
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     "user",
		Status:   "active",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// user login
func (s *UserService) Login(req *models.LoginRequest) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, "", err
	}

	// check user role
	if user.Role != "user" {
		return nil, "", errors.New("only regular users can login")
	}

	// check user status
	if user.Status != "active" {
		return nil, "", errors.New("user account is not active")
	}

	// check password
	if !user.CheckPassword(req.Password) {
		return nil, "", errors.New("invalid password")
	}

	// generate jwt token
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

// get user by id
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	// check if database connection is initialized
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// get user by id
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// get user by username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	// check if database connection is initialized
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// get user by username
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// get user by email
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	// check if database connection is initialized
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// get user by email
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// update user
func (s *UserService) UpdateUser(user *models.User) error {
	// check if database connection is initialized
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// update user
	return s.db.Save(user).Error
}

// delete user
func (s *UserService) DeleteUser(id uint) error {
	// check if database connection is initialized
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// delete user
	result := s.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// get all users
func (s *UserService) GetAllUsers(page, pageSize int) ([]models.User, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var users []models.User
	var total int64

	// calculate total
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// paginate query
	offset := (page - 1) * pageSize
	if err := s.db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// SoftDeleteUser 软删除用户账户（用户自删除）
func (s *UserService) SoftDeleteUser(userID uint) error {
	// check if database connection is initialized
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 检查用户是否已经被删除
	if user.Status == "deleted" {
		return errors.New("user account is already deleted")
	}

	// 软删除：更新状态为 deleted
	user.Status = "deleted"
	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

// GetActiveUsers 获取所有活跃用户（排除已删除的用户）
func (s *UserService) GetActiveUsers(page, pageSize int) ([]models.User, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var users []models.User
	var total int64

	// 只统计活跃用户
	if err := s.db.Model(&models.User{}).Where("status != ?", "deleted").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询活跃用户
	offset := (page - 1) * pageSize
	if err := s.db.Where("status != ?", "deleted").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUserRole 更新用户角色（管理员功能）
func (s *UserService) UpdateUserRole(userID uint, newRole string) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证角色是否有效
	validRoles := []string{"user", "admin", "system"}
	isValidRole := false
	for _, role := range validRoles {
		if role == newRole {
			isValidRole = true
			break
		}
	}

	if !isValidRole {
		return errors.New("invalid role. Valid roles are: user, admin, system")
	}

	// 更新用户角色
	result := s.db.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// UpdateUserStatus 更新用户状态（管理员功能）
func (s *UserService) UpdateUserStatus(userID uint, newStatus string) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证状态是否有效
	validStatuses := []string{"active", "inactive", "suspended", "deleted"}
	isValidStatus := false
	for _, status := range validStatuses {
		if status == newStatus {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		return errors.New("invalid status. Valid statuses are: active, inactive, suspended, deleted")
	}

	// 更新用户状态
	result := s.db.Model(&models.User{}).Where("id = ?", userID).Update("status", newStatus)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}
