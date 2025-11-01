package services

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mwork_backend/internal/email"
	"time"

	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"

	"golang.org/x/crypto/bcrypt"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type AuthService interface {
	Register(db *gorm.DB, req *dto.RegisterRequest) error
	Login(db *gorm.DB, req *dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(db *gorm.DB, refreshToken string) (*dto.AuthResponse, error)
	Logout(db *gorm.DB, refreshToken string) error
	VerifyEmail(db *gorm.DB, token string) error
	RequestPasswordReset(db *gorm.DB, email string) error
	ResetPassword(db *gorm.DB, token, newPassword string) error
	ChangePassword(db *gorm.DB, userID, currentPassword, newPassword string) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type AuthServiceImpl struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	emailProvider    email.Provider
	refreshTokenRepo repositories.RefreshTokenRepository
}

// ✅ Конструктор обновлен (db убран)
func NewAuthService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	emailProvider email.Provider,
	refreshTokenRepo repositories.RefreshTokenRepository,
) AuthService {
	return &AuthServiceImpl{
		// ❌ 'db: db,' УДАЛЕНО
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		subscriptionRepo: subscriptionRepo,
		emailProvider:    emailProvider,
		refreshTokenRepo: refreshTokenRepo,
	}
}

// Register - 'db' добавлен
func (s *AuthServiceImpl) Register(db *gorm.DB, req *dto.RegisterRequest) error {
	if len(req.Password) < 6 {
		return apperrors.ErrWeakPassword
	}
	if req.Role != models.UserRoleModel && req.Role != models.UserRoleEmployer {
		return apperrors.ErrInvalidUserRole
	}
	if err := s.validateRegisterRequest(req); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.InternalError(err)
	}

	verificationToken := generateRandomToken()
	user := &models.User{
		Email:             req.Email,
		PasswordHash:      string(hashedPassword),
		Role:              req.Role,
		Status:            models.UserStatusPending,
		IsVerified:        false,
		VerificationToken: verificationToken,
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.userRepo.Create(tx, user); err != nil {
		if errors.Is(err, repositories.ErrUserAlreadyExists) {
			return apperrors.ErrEmailAlreadyExists
		}
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.createUserProfile(tx, user, req); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.assignFreeSubscription(tx, user.ID); err != nil {
		fmt.Printf("Failed to create free subscription: %v\n", err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	s.sendVerificationEmail(user.Email, verificationToken)
	return nil
}

// Login - 'db' добавлен
func (s *AuthServiceImpl) Login(db *gorm.DB, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	// ✅ Используем 'db' из параметра
	user, err := s.userRepo.FindByEmail(db, req.Email)
	if err != nil {
		return nil, handleRepositoryError(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	refreshToken, err := s.createRefreshToken(tx, user.ID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	userDto := buildUserDTO(user)

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *userDto,
	}, nil
}

// RefreshToken - 'db' добавлен
func (s *AuthServiceImpl) RefreshToken(db *gorm.DB, refreshToken string) (*dto.AuthResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	token, err := s.refreshTokenRepo.FindByToken(tx, refreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		// ✅ Передаем tx
		s.refreshTokenRepo.DeleteByToken(tx, refreshToken)
		return nil, apperrors.ErrInvalidToken
	}

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, token.UserID)
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}

	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	newRefreshToken, err := s.rotateRefreshToken(tx, token.UserID, refreshToken)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	userDto := buildUserDTO(user)

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         *userDto,
	}, nil
}

// Logout - 'db' добавлен
func (s *AuthServiceImpl) Logout(db *gorm.DB, refreshToken string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.refreshTokenRepo.DeleteByToken(tx, refreshToken); err != nil {
		if errors.Is(err, repositories.ErrRefreshTokenNotFound) {
			return tx.Commit().Error
		}
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// VerifyEmail - 'db' добавлен
func (s *AuthServiceImpl) VerifyEmail(db *gorm.DB, token string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByVerificationToken(tx, token)
	if err != nil {
		return apperrors.ErrInvalidToken
	}

	// ✅ Передаем tx
	if err := s.userRepo.VerifyUser(tx, user.ID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// RequestPasswordReset - 'db' добавлен
func (s *AuthServiceImpl) RequestPasswordReset(db *gorm.DB, email string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByEmail(tx, email)
	if err != nil {
		tx.Rollback()
		return nil
	}

	resetToken := generateRandomToken()
	resetTokenExp := time.Now().Add(1 * time.Hour)
	user.ResetToken = resetToken
	user.ResetTokenExp = &resetTokenExp

	// ✅ Передаем tx
	if err := s.userRepo.Update(tx, user); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	s.sendPasswordResetEmail(user.Email, resetToken)
	return nil
}

// ResetPassword - 'db' добавлен
func (s *AuthServiceImpl) ResetPassword(db *gorm.DB, token, newPassword string) error {
	if len(newPassword) < 6 {
		return apperrors.ErrWeakPassword
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByResetToken(tx, token)
	if err != nil {
		return apperrors.ErrInvalidToken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)
	user.ResetToken = ""
	user.ResetTokenExp = nil

	// ✅ Передаем tx
	if err := s.userRepo.Update(tx, user); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.refreshTokenRepo.DeleteByUserID(tx, user.ID); err != nil {
		fmt.Printf("Failed to delete refresh tokens on reset password: %v\n", err)
	}
	return tx.Commit().Error
}

// ChangePassword - 'db' добавлен
func (s *AuthServiceImpl) ChangePassword(db *gorm.DB, userID, currentPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return apperrors.ErrWeakPassword
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, userID)
	if err != nil {
		return handleRepositoryError(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return apperrors.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)

	// ✅ Передаем tx
	if err := s.userRepo.Update(tx, user); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// --- Helper functions ---

// (createUserProfile - 'db' уже был)
func (s *AuthServiceImpl) createUserProfile(db *gorm.DB, user *models.User, req *dto.RegisterRequest) error {
	if user.Role == models.UserRoleModel {
		profile := &models.ModelProfile{
			UserID:   user.ID,
			Name:     req.Name,
			City:     req.City,
			Age:      0,
			IsPublic: true,
		}
		// ✅ Передаем db
		return s.profileRepo.CreateModelProfile(db, profile)
	} else if user.Role == models.UserRoleEmployer {
		profile := &models.EmployerProfile{
			UserID:      user.ID,
			CompanyName: req.CompanyName,
			City:        req.City,
			IsVerified:  false,
		}
		// ✅ Передаем db
		return s.profileRepo.CreateEmployerProfile(db, profile)
	}
	return nil
}

// (assignFreeSubscription - 'db' уже был)
func (s *AuthServiceImpl) assignFreeSubscription(db *gorm.DB, userID string) error {
	// ✅ Передаем db
	freePlan, err := s.subscriptionRepo.FindPlanByName(db, "Free")
	if err != nil || freePlan == nil {
		return fmt.Errorf("free plan not found: %w", err)
	}

	subscription := &models.UserSubscription{
		UserID:    userID,
		PlanID:    freePlan.ID,
		Status:    models.SubscriptionStatusActive,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(100, 0, 0),
		AutoRenew: false,
	}

	// ✅ Передаем db
	return s.subscriptionRepo.CreateUserSubscription(db, subscription)
}

// (createRefreshToken - 'db' уже был)
func (s *AuthServiceImpl) createRefreshToken(db *gorm.DB, userID string) (string, error) {
	refreshToken := generateRandomToken()
	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)

	refreshTokenModel := &models.RefreshToken{
		UserID:    userID,
		Token:     refreshToken,
		ExpiresAt: refreshTokenExp,
	}

	// ✅ Передаем db
	if err := s.refreshTokenRepo.Create(db, refreshTokenModel); err != nil {
		return "", err
	}

	return refreshToken, nil
}

// (rotateRefreshToken - 'db' уже был)
func (s *AuthServiceImpl) rotateRefreshToken(db *gorm.DB, userID, oldToken string) (string, error) {
	// ✅ Передаем db
	if err := s.refreshTokenRepo.DeleteByToken(db, oldToken); err != nil {
		return "", err
	}
	// ✅ Передаем db
	return s.createRefreshToken(db, userID)
}

// (checkUserStatus - чистая функция, без изменений)
func (s *AuthServiceImpl) checkUserStatus(user *models.User) error {
	switch user.Status {
	case models.UserStatusSuspended:
		return apperrors.ErrUserSuspended
	case models.UserStatusBanned:
		return apperrors.ErrUserBanned
	case models.UserStatusPending:
		if !user.IsVerified {
			return apperrors.ErrUserNotVerified
		}
	}
	return nil
}

// (buildUserDTO - чистая функция, без изменений)
func buildUserDTO(user *models.User) *dto.UserDTO {
	return &dto.UserDTO{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
	}
}

// (sendVerificationEmail - чистая функция, без изменений)
func (s *AuthServiceImpl) sendVerificationEmail(email, token string) {
	if s.emailProvider == nil {
		return
	}
	go func() {
		if err := s.emailProvider.SendVerification(email, token); err != nil {
			fmt.Printf("Failed to send verification email: %v\n", err)
		}
	}()
}

// (sendPasswordResetEmail - чистая функция, без изменений)
func (s *AuthServiceImpl) sendPasswordResetEmail(email, token string) {
	if s.emailProvider == nil {
		return
	}
	go func() {
		data := map[string]interface{}{
			"ResetURL": fmt.Sprintf("https://mwork.ru/reset-password?token=%s", token),
		}
		if err := s.emailProvider.SendTemplate([]string{email}, "Сброс пароля", "password_reset", data); err != nil {
			fmt.Printf("Failed to send password reset email: %v\n", err)
		}
	}()
}

// (validateRegisterRequest - чистая функция, без изменений)
func (s *AuthServiceImpl) validateRegisterRequest(req *dto.RegisterRequest) error {
	if req.Role == models.UserRoleModel {
		if req.Name == "" {
			return apperrors.ValidationError("name is required for model role")
		}
		if req.City == "" {
			return apperrors.ValidationError("city is required for model role")
		}
	} else if req.Role == models.UserRoleEmployer {
		if req.CompanyName == "" {
			return apperrors.ValidationError("company_name is required for employer role")
		}
		if req.City == "" {
			return apperrors.ValidationError("city is required for employer role")
		}
	}
	return nil
}

// (generateRandomToken - чистая функция, без изменений)
func generateRandomToken() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
