package services

import (
	"fmt"
	"mwork_backend/internal/email"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(req *dto.RegisterRequest) error
	Login(req *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(refreshToken string) (*dto.LoginResponse, error)
	Logout(refreshToken string) error
	VerifyEmail(token string) error
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
	ChangePassword(userID, currentPassword, newPassword string) error
}

type AuthServiceImpl struct {
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	emailSender      email.Sender
}

func NewAuthService(
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	emailSender email.Sender,
) AuthService {
	return &AuthServiceImpl{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		subscriptionRepo: subscriptionRepo,
		emailSender:      emailSender,
	}
}

// Register - регистрация нового пользователя
func (s *AuthServiceImpl) Register(req *dto.RegisterRequest) error {
	// Валидация пароля
	if len(req.Password) < 6 {
		return appErrors.ErrWeakPassword
	}

	// Проверка роли
	if req.Role != models.UserRoleModel && req.Role != models.UserRoleEmployer {
		return appErrors.ErrInvalidUserRole
	}

	// Валидация полей в зависимости от роли
	if err := s.validateRegisterRequest(req); err != nil {
		return err
	}

	// Остальной код остается без изменений...
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.InternalError(err)
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

	if err := s.userRepo.Create(user); err != nil {
		if appErrors.Is(err, repositories.ErrUserAlreadyExists) {
			return appErrors.ErrEmailAlreadyExists
		}
		return appErrors.InternalError(err)
	}

	if err := s.createUserProfile(user, req); err != nil {
		s.userRepo.Delete(user.ID)
		return appErrors.InternalError(err)
	}

	if err := s.assignFreeSubscription(user.ID); err != nil {
		fmt.Printf("Failed to create free subscription: %v\n", err)
	}

	s.sendVerificationEmail(user.Email, verificationToken)

	return nil
}

// Login - аутентификация пользователя
func (s *AuthServiceImpl) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// Поиск пользователя
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		if appErrors.Is(err, repositories.ErrUserNotFound) {
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, appErrors.InternalError(err)
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, appErrors.ErrInvalidCredentials
	}

	// Проверка статуса пользователя
	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	// Генерация токенов
	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	refreshToken, err := s.createRefreshToken(user.ID)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	// Построение ответа с профилем
	userResponse, err := s.buildUserResponse(user)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userResponse,
	}, nil
}

// RefreshToken - обновление access token по refresh token
func (s *AuthServiceImpl) RefreshToken(refreshToken string) (*dto.LoginResponse, error) {
	// Поиск refresh token в БД
	token, err := s.userRepo.FindRefreshToken(refreshToken)
	if err != nil {
		return nil, appErrors.ErrInvalidToken
	}

	// Проверка срока действия
	if time.Now().After(token.ExpiresAt) {
		s.userRepo.DeleteRefreshToken(refreshToken)
		return nil, appErrors.ErrInvalidToken
	}

	// Получение пользователя
	user, err := s.userRepo.FindByID(token.UserID)
	if err != nil {
		return nil, appErrors.ErrInvalidToken
	}

	// Проверка статуса
	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	// Генерация нового access token
	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	// Ротация refresh token
	newRefreshToken, err := s.rotateRefreshToken(token.UserID, refreshToken)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	// Построение ответа
	userResponse, err := s.buildUserResponse(user)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         userResponse,
	}, nil
}

// Logout - выход (удаление refresh token)
func (s *AuthServiceImpl) Logout(refreshToken string) error {
	return s.userRepo.DeleteRefreshToken(refreshToken)
}

// VerifyEmail - подтверждение email
func (s *AuthServiceImpl) VerifyEmail(token string) error {
	// Поиск пользователя по токену верификации
	users, _, err := s.userRepo.FindWithFilter(repositories.UserFilter{
		Search:   token,
		Page:     1,
		PageSize: 1,
	})
	if err != nil || len(users) == 0 {
		return appErrors.ErrInvalidToken
	}

	user := &users[0]

	// Проверка токена
	if user.VerificationToken != token {
		return appErrors.ErrInvalidToken
	}

	// Верификация пользователя
	return s.userRepo.VerifyUser(user.ID)
}

// RequestPasswordReset - запрос сброса пароля
func (s *AuthServiceImpl) RequestPasswordReset(email string) error {
	// Поиск пользователя
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Не раскрываем существование email для безопасности
		return nil
	}

	// Генерация reset token
	resetToken := generateRandomToken()
	resetTokenExp := time.Now().Add(1 * time.Hour)

	user.ResetToken = resetToken
	user.ResetTokenExp = &resetTokenExp

	if err := s.userRepo.Update(user); err != nil {
		return appErrors.InternalError(err)
	}

	// Отправка email со ссылкой для сброса пароля
	s.sendPasswordResetEmail(user.Email, resetToken)

	return nil
}

// ResetPassword - сброс пароля по токену
func (s *AuthServiceImpl) ResetPassword(token, newPassword string) error {
	// Поиск пользователя по reset token
	users, _, err := s.userRepo.FindWithFilter(repositories.UserFilter{
		Search:   token,
		Page:     1,
		PageSize: 1,
	})
	if err != nil || len(users) == 0 {
		return appErrors.ErrInvalidToken
	}

	user := &users[0]

	// Проверка токена и срока действия
	if user.ResetToken != token || user.ResetTokenExp == nil || time.Now().After(*user.ResetTokenExp) {
		return appErrors.ErrInvalidToken
	}

	// Валидация нового пароля
	if len(newPassword) < 6 {
		return appErrors.ErrWeakPassword
	}

	// Хеширование нового пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)
	user.ResetToken = ""
	user.ResetTokenExp = nil

	if err := s.userRepo.Update(user); err != nil {
		return appErrors.InternalError(err)
	}

	// Удаляем все refresh токены для безопасности
	s.userRepo.DeleteUserRefreshTokens(user.ID)

	return nil
}

// ChangePassword - смена пароля (когда пользователь знает текущий)
func (s *AuthServiceImpl) ChangePassword(userID, currentPassword, newPassword string) error {
	// Получение пользователя
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	// Проверка текущего пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return appErrors.ErrInvalidCredentials
	}

	// Валидация нового пароля
	if len(newPassword) < 6 {
		return appErrors.ErrWeakPassword
	}

	// Хеширование нового пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)

	return s.userRepo.Update(user)
}

// --- Helper functions ---

// createUserProfile создает профиль в зависимости от роли
func (s *AuthServiceImpl) createUserProfile(user *models.User, req *dto.RegisterRequest) error {
	if user.Role == models.UserRoleModel {
		profile := &models.ModelProfile{
			UserID:   user.ID,
			Name:     req.Name,
			City:     req.City,
			Age:      0, // Можно сделать обязательным полем или оставить 0 по умолчанию
			IsPublic: true,
		}
		return s.profileRepo.CreateModelProfile(profile)
	} else if user.Role == models.UserRoleEmployer {
		profile := &models.EmployerProfile{
			UserID:      user.ID,
			CompanyName: req.CompanyName,
			City:        req.City,
			IsVerified:  false,
		}
		return s.profileRepo.CreateEmployerProfile(profile)
	}
	return nil
}

// assignFreeSubscription назначает бесплатную подписку новому пользователю
func (s *AuthServiceImpl) assignFreeSubscription(userID string) error {
	freePlan, err := s.subscriptionRepo.FindPlanByName("Free")
	if err != nil || freePlan == nil {
		return err
	}

	subscription := &models.UserSubscription{
		UserID:    userID,
		PlanID:    freePlan.ID,
		Status:    models.SubscriptionStatusActive,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(100, 0, 0), // 100 лет для free плана
		AutoRenew: false,
	}

	return s.subscriptionRepo.CreateUserSubscription(subscription)
}

// createRefreshToken создает и сохраняет refresh token
func (s *AuthServiceImpl) createRefreshToken(userID string) (string, error) {
	refreshToken := generateRandomToken()
	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour) // 7 дней

	refreshTokenModel := &models.RefreshToken{
		UserID:    userID,
		Token:     refreshToken,
		ExpiresAt: refreshTokenExp,
	}

	if err := s.userRepo.CreateRefreshToken(refreshTokenModel); err != nil {
		return "", err
	}

	return refreshToken, nil
}

// rotateRefreshToken удаляет старый и создает новый refresh token
func (s *AuthServiceImpl) rotateRefreshToken(userID, oldToken string) (string, error) {
	// Удаляем старый токен
	if err := s.userRepo.DeleteRefreshToken(oldToken); err != nil {
		return "", err
	}

	// Создаем новый
	return s.createRefreshToken(userID)
}

// checkUserStatus проверяет статус пользователя
func (s *AuthServiceImpl) checkUserStatus(user *models.User) error {
	switch user.Status {
	case models.UserStatusSuspended:
		return appErrors.ErrUserSuspended
	case models.UserStatusBanned:
		return appErrors.ErrUserBanned
	case models.UserStatusPending:
		if !user.IsVerified {
			return appErrors.ErrUserNotVerified
		}
	}
	return nil
}

// buildUserResponse строит ответ с данными пользователя и профилем
func (s *AuthServiceImpl) buildUserResponse(user *models.User) (*dto.UserResponse, error) {
	userResponse := &dto.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		IsVerified: user.IsVerified,
	}

	// Загрузка профиля
	if user.Role == models.UserRoleModel {
		if user.ModelProfile != nil {
			userResponse.Profile = user.ModelProfile
		} else {
			profile, err := s.profileRepo.FindModelProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	} else if user.Role == models.UserRoleEmployer {
		if user.EmployerProfile != nil {
			userResponse.Profile = user.EmployerProfile
		} else {
			profile, err := s.profileRepo.FindEmployerProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	}

	return userResponse, nil
}

// sendVerificationEmail отправляет email с токеном верификации
func (s *AuthServiceImpl) sendVerificationEmail(email, token string) {
	if s.emailSender == nil {
		return
	}

	go func() {
		if err := s.emailSender.SendVerification(email, token); err != nil {
			fmt.Printf("Failed to send verification email: %v\n", err)
		}
	}()
}

// sendPasswordResetEmail отправляет email со ссылкой для сброса пароля
func (s *AuthServiceImpl) sendPasswordResetEmail(email, token string) {
	if s.emailSender == nil {
		return
	}

	go func() {
		data := map[string]interface{}{
			"ResetURL": fmt.Sprintf("https://mwork.ru/reset-password?token=%s", token),
		}
		if err := s.emailSender.SendTemplate([]string{email}, "Сброс пароля", "password_reset", data); err != nil {
			fmt.Printf("Failed to send password reset email: %v\n", err)
		}
	}()
}

func (s *AuthServiceImpl) validateRegisterRequest(req *dto.RegisterRequest) error {
	if req.Role == models.UserRoleModel {
		if req.Name == "" {
			return appErrors.NewValidationError("name is required for model role")
		}
		if req.City == "" {
			return appErrors.NewValidationError("city is required for model role")
		}
	} else if req.Role == models.UserRoleEmployer {
		if req.CompanyName == "" {
			return appErrors.NewValidationError("company_name is required for employer role")
		}
		if req.City == "" {
			return appErrors.NewValidationError("city is required for employer role")
		}
	}
	return nil
}

// generateRandomToken генерирует случайный токен
func generateRandomToken() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
