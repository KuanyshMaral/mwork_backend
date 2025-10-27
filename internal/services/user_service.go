package services

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"mwork_backend/internal/pkg/email"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

type UserService struct {
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	emailSender      email.Sender
}

func NewUserService(
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	emailSender email.Sender,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		subscriptionRepo: subscriptionRepo,
		emailSender:      emailSender,
	}
}

// =======================
// Auth operations
// =======================

func (s *UserService) Register(req *dto.RegisterRequest) error {
	if len(req.Password) < 6 {
		return appErrors.ErrWeakPassword
	}

	if req.Role != models.UserRoleModel && req.Role != models.UserRoleEmployer {
		return appErrors.ErrInvalidUserRole
	}

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

	if req.Role == models.UserRoleModel {
		profile := &models.ModelProfile{
			UserID:   user.ID,
			Name:     req.Name,
			City:     req.City,
			Age:      0,
			IsPublic: true,
		}
		if err := s.profileRepo.CreateModelProfile(profile); err != nil {
			s.userRepo.Delete(user.ID)
			return appErrors.InternalError(err)
		}
	} else if req.Role == models.UserRoleEmployer {
		profile := &models.EmployerProfile{
			UserID:      user.ID,
			CompanyName: req.CompanyName,
			City:        req.City,
			IsVerified:  false,
		}
		if err := s.profileRepo.CreateEmployerProfile(profile); err != nil {
			s.userRepo.Delete(user.ID)
			return appErrors.InternalError(err)
		}
	}

	freePlan, err := s.subscriptionRepo.FindPlanByName("Free")
	if err == nil && freePlan != nil {
		subscription := &models.UserSubscription{
			UserID:    user.ID,
			PlanID:    freePlan.ID,
			Status:    models.SubscriptionStatusActive,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(100, 0, 0),
			AutoRenew: false,
		}
		if err := s.subscriptionRepo.CreateUserSubscription(subscription); err != nil {
			fmt.Printf("Failed to create free subscription: %v\n", err)
		}
	}

	if s.emailSender != nil {
		go func() {
			if err := s.emailSender.SendVerification(user.Email, verificationToken); err != nil {
				fmt.Printf("Failed to send verification email: %v\n", err)
			}
		}()
	}

	return nil
}

func (s *UserService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		if appErrors.Is(err, repositories.ErrUserNotFound) {
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, appErrors.InternalError(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, appErrors.ErrInvalidCredentials
	}

	switch user.Status {
	case models.UserStatusSuspended:
		return nil, appErrors.ErrUserSuspended
	case models.UserStatusBanned:
		return nil, appErrors.ErrUserBanned
	case models.UserStatusPending:
		if !user.IsVerified {
			return nil, appErrors.ErrUserNotVerified
		}
	}

	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	refreshToken := generateRandomToken()
	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)

	refreshTokenModel := &models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: refreshTokenExp,
	}
	if err := s.userRepo.CreateRefreshToken(refreshTokenModel); err != nil {
		return nil, appErrors.InternalError(err)
	}

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

func (s *UserService) RefreshToken(refreshToken string) (*dto.LoginResponse, error) {
	token, err := s.userRepo.FindRefreshToken(refreshToken)
	if err != nil {
		return nil, appErrors.ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		s.userRepo.DeleteRefreshToken(refreshToken)
		return nil, appErrors.ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(token.UserID)
	if err != nil {
		return nil, appErrors.ErrInvalidToken
	}

	switch user.Status {
	case models.UserStatusSuspended, models.UserStatusBanned:
		return nil, appErrors.ErrUserSuspended
	}

	accessToken, err := auth.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	newRefreshToken := generateRandomToken()
	newRefreshTokenExp := time.Now().Add(7 * 24 * time.Hour)

	s.userRepo.DeleteRefreshToken(refreshToken)

	newRefreshTokenModel := &models.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: newRefreshTokenExp,
	}
	if err := s.userRepo.CreateRefreshToken(newRefreshTokenModel); err != nil {
		return nil, appErrors.InternalError(err)
	}

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

func (s *UserService) Logout(refreshToken string) error {
	return s.userRepo.DeleteRefreshToken(refreshToken)
}

func (s *UserService) VerifyEmail(token string) error {
	users, _, err := s.userRepo.FindWithFilter(repositories.UserFilter{
		Search:   token,
		Page:     1,
		PageSize: 1,
	})
	if err != nil || len(users) == 0 {
		return appErrors.ErrInvalidToken
	}

	user := &users[0]

	if user.VerificationToken != token {
		return appErrors.ErrInvalidToken
	}

	return s.userRepo.VerifyUser(user.ID)
}

// =======================
// Profile operations
// =======================

func (s *UserService) GetProfile(userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	return s.buildUserResponse(user)
}

func (s *UserService) UpdateProfile(userID string, req *dto.UpdateProfileRequestUser) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if user.Role == models.UserRoleModel {
		profile, err := s.profileRepo.FindModelProfileByUserID(userID)
		if err != nil {
			return appErrors.InternalError(err)
		}

		updateModelProfile(profile, req)
		return s.profileRepo.UpdateModelProfile(profile)
	} else if user.Role == models.UserRoleEmployer {
		profile, err := s.profileRepo.FindEmployerProfileByUserID(userID)
		if err != nil {
			return appErrors.InternalError(err)
		}

		updateEmployerProfile(profile, req)
		return s.profileRepo.UpdateEmployerProfile(profile)
	}

	return appErrors.ErrInvalidUserRole
}

// =======================
// Admin operations
// =======================

func (s *UserService) GetUsers(filter dto.AdminUserFilter) ([]*dto.UserResponse, int64, error) {
	repoFilter := repositories.UserFilter{
		Role:       filter.Role,
		Status:     filter.Status,
		IsVerified: filter.IsVerified,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
		Search:     filter.Search,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}

	users, total, err := s.userRepo.FindWithFilter(repoFilter)
	if err != nil {
		return nil, 0, appErrors.InternalError(err)
	}

	var userResponses []*dto.UserResponse
	for i := range users {
		userResponse, err := s.buildUserResponse(&users[i])
		if err != nil {
			continue
		}
		userResponses = append(userResponses, userResponse)
	}

	return userResponses, total, nil
}

func (s *UserService) UpdateUserStatus(adminID, userID string, status models.UserStatus) error {
	if adminID == userID {
		return appErrors.ErrCannotModifySelf
	}

	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	return s.userRepo.UpdateStatus(userID, status)
}

func (s *UserService) VerifyEmployer(adminID, employerID string) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	return s.profileRepo.VerifyEmployerProfile(employerID)
}

func (s *UserService) GetRegistrationStats(days int) (*repositories.RegistrationStats, error) {
	stats, err := s.userRepo.GetRegistrationStats(days)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}
	return stats, nil
}

// =======================
// Password operations
// =======================

func (s *UserService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return appErrors.ErrInvalidCredentials
	}

	if len(newPassword) < 6 {
		return appErrors.ErrWeakPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)
	return s.userRepo.Update(user)
}

func (s *UserService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil
	}

	resetToken := generateRandomToken()
	resetTokenExp := time.Now().Add(1 * time.Hour)

	user.ResetToken = resetToken
	user.ResetTokenExp = &resetTokenExp

	if err := s.userRepo.Update(user); err != nil {
		return appErrors.InternalError(err)
	}

	if s.emailSender != nil {
		go func() {
			data := map[string]interface{}{
				"ResetURL": fmt.Sprintf("https://mwork.ru/reset-password?token=%s", resetToken),
			}
			if err := s.emailSender.SendTemplate([]string{user.Email}, "Сброс пароля", "password_reset", data); err != nil {
				fmt.Printf("Failed to send password reset email: %v\n", err)
			}
		}()
	}

	return nil
}

func (s *UserService) ResetPassword(token, newPassword string) error {
	users, _, err := s.userRepo.FindWithFilter(repositories.UserFilter{
		Search:   token,
		Page:     1,
		PageSize: 1,
	})
	if err != nil || len(users) == 0 {
		return appErrors.ErrInvalidToken
	}

	user := &users[0]

	if user.ResetToken != token || user.ResetTokenExp == nil || time.Now().After(*user.ResetTokenExp) {
		return appErrors.ErrInvalidToken
	}

	if len(newPassword) < 6 {
		return appErrors.ErrWeakPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.InternalError(err)
	}

	user.PasswordHash = string(hashedPassword)
	user.ResetToken = ""
	user.ResetTokenExp = nil

	return s.userRepo.Update(user)
}

// =======================
// Helper methods
// =======================

func (s *UserService) buildUserResponse(user *models.User) (*dto.UserResponse, error) {
	userResponse := &dto.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		IsVerified: user.IsVerified,
	}

	if user.Role == models.UserRoleModel && user.ModelProfile != nil {
		userResponse.Profile = user.ModelProfile
	} else if user.Role == models.UserRoleEmployer && user.EmployerProfile != nil {
		userResponse.Profile = user.EmployerProfile
	} else {
		if user.Role == models.UserRoleModel {
			profile, err := s.profileRepo.FindModelProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		} else if user.Role == models.UserRoleEmployer {
			profile, err := s.profileRepo.FindEmployerProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	}

	return userResponse, nil
}

func updateFieldWithConversion(dst interface{}, src interface{}) {
	if src == nil {
		return
	}
	switch d := dst.(type) {
	case *string:
		*d = *(src.(*string))
	case *int:
		val, _ := strconv.Atoi(*(src.(*string))) // конвертация string -> int
		*d = val
	case *float64:
		*d = *(src.(*float64))
	case *bool:
		*d = *(src.(*bool))
	}
}

func updateModelProfile(profile *models.ModelProfile, req *dto.UpdateProfileRequestUser) {
	updateFieldWithConversion(&profile.Name, req.Name)
	updateFieldWithConversion(&profile.City, req.City)
	updateFieldWithConversion(&profile.Age, req.Age)
	updateFieldWithConversion(&profile.Height, req.Height)
	updateFieldWithConversion(&profile.Weight, req.Weight)
	updateFieldWithConversion(&profile.Gender, req.Gender)
	updateFieldWithConversion(&profile.Experience, req.Experience)
	updateFieldWithConversion(&profile.HourlyRate, req.HourlyRate)
	updateFieldWithConversion(&profile.Description, req.Description)
	updateFieldWithConversion(&profile.ClothingSize, req.ClothingSize)
	updateFieldWithConversion(&profile.ShoeSize, req.ShoeSize)
	updateFieldWithConversion(&profile.BarterAccepted, req.BarterAccepted)
	updateFieldWithConversion(&profile.IsPublic, req.IsPublic)
}

func updateEmployerProfile(profile *models.EmployerProfile, req *dto.UpdateProfileRequestUser) {
	updateFieldWithConversion(&profile.CompanyName, req.CompanyName)
	updateFieldWithConversion(&profile.ContactPerson, req.ContactPerson)
	updateFieldWithConversion(&profile.Phone, req.Phone)
	updateFieldWithConversion(&profile.Website, req.Website)
	updateFieldWithConversion(&profile.City, req.City)
	updateFieldWithConversion(&profile.CompanyType, req.CompanyType)
	updateFieldWithConversion(&profile.Description, req.Description)
}

func generateRandomToken() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
