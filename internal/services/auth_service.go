package services

import (
	"context"
	"errors"
	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo       *repositories.UserRepository
	emailService   *EmailService
	refreshService *RefreshTokenService
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	emailService *EmailService,
	refreshService *RefreshTokenService,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		emailService:   emailService,
		refreshService: refreshService,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, role string) (*models.User, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	verificationToken := uuid.New().String()

	user := &models.User{
		Email:             email,
		PasswordHash:      string(hash),
		Role:              role,
		Status:            "pending",
		IsVerified:        false,
		VerificationToken: verificationToken,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.emailService.SendVerificationEmail(user.Email, verificationToken)

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *models.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, errors.New("invalid password")
	}

	// 1. Access token (15 мин)
	accessToken, err := auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", "", nil, err
	}

	// 2. Refresh token (30 дней)
	refreshToken := uuid.New().String()
	refreshExp := time.Now().Add(30 * 24 * time.Hour)

	err = s.refreshService.Create(ctx, user.ID, refreshToken, refreshExp)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

func (s *AuthService) VerifyEmailToken(ctx context.Context, token string) error {
	user, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil || user == nil {
		return errors.New("invalid token")
	}

	user.IsVerified = true
	user.VerificationToken = ""
	user.Status = "active"
	return s.userRepo.Update(ctx, user)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	token := uuid.New().String()
	expiration := time.Now().Add(30 * time.Minute)

	user.ResetToken = token
	user.ResetTokenExp = expiration
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return s.emailService.SendPasswordResetEmail(user.Email, token)
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.userRepo.GetByResetToken(ctx, token)
	if err != nil || user == nil {
		return errors.New("invalid token")
	}
	if time.Now().After(user.ResetTokenExp) {
		return errors.New("token expired")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	user.ResetToken = ""
	user.ResetTokenExp = time.Time{}

	return s.userRepo.Update(ctx, user)
}

func (s *AuthService) RefreshService() *RefreshTokenService {
	return s.refreshService
}
