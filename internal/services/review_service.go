package services

import (
	"errors"
	"fmt"
	"gorm.io/gorm"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

var (
	ErrInvalidReviewRating  = errors.New("invalid review rating")
	ErrSelfReviewNotAllowed = errors.New("self-review is not allowed")
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type ReviewService interface {
	// Review operations
	CreateReview(db *gorm.DB, userID string, req *dto.CreateReviewRequest) (*dto.ReviewResponse, error)
	GetReview(db *gorm.DB, reviewID string) (*dto.ReviewResponse, error)
	GetModelReviews(db *gorm.DB, modelID string, page, pageSize int) (*dto.ReviewListResponse, error)
	GetEmployerReviews(db *gorm.DB, employerID string, page, pageSize int) (*dto.ReviewListResponse, error)
	GetCastingReviews(db *gorm.DB, castingID string) ([]*dto.ReviewResponse, error)
	UpdateReview(db *gorm.DB, userID, reviewID string, req *dto.UpdateReviewRequest) error
	DeleteReview(db *gorm.DB, userID, reviewID string) error

	// Rating operations
	GetModelRating(db *gorm.DB, modelID string) (*dto.RatingResponse, error)
	GetEmployerRating(db *gorm.DB, employerID string) (*dto.RatingResponse, error)
	GetModelRatingStats(db *gorm.DB, modelID string) (*repositories.RatingStats, error)
	GetEmployerRatingStats(db *gorm.DB, employerID string) (*repositories.RatingStats, error)

	// Validation and business logic
	CanUserReview(db *gorm.DB, employerID, modelID, castingID string) (bool, error)
	IsUserReviewable(db *gorm.DB, userID string, userRole models.UserRole) (bool, error)

	// Admin operations
	GetAllReviews(db *gorm.DB, page, pageSize int) (*dto.ReviewListResponse, error)
	GetRecentReviews(db *gorm.DB, limit int) ([]*dto.ReviewResponse, error)
	GetPlatformReviewStats(db *gorm.DB) (*repositories.PlatformReviewStats, error)
	DeleteReviewByAdmin(db *gorm.DB, adminID, reviewID string) error

	// Additional features
	GetReviewSummary(db *gorm.DB, modelID string) (*repositories.ReviewSummary, error)
	SearchReviews(db *gorm.DB, criteria dto.ReviewSearchCriteria) (*dto.ReviewListResponse, error)
	GetUserReviewStats(db *gorm.DB, userID string, userRole models.UserRole) (*dto.UserReviewStats, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type reviewService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	reviewRepo       repositories.ReviewRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	notificationRepo repositories.NotificationRepository
}

// ✅ Конструктор обновлен (db убран)
func NewReviewService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	reviewRepo repositories.ReviewRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	notificationRepo repositories.NotificationRepository,
) ReviewService {
	return &reviewService{
		// ❌ 'db: db,' УДАЛЕНО
		reviewRepo:       reviewRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		notificationRepo: notificationRepo,
	}
}

// ---------------- Review Operations ----------------

// CreateReview - 'db' добавлен
func (s *reviewService) CreateReview(db *gorm.DB, userID string, req *dto.CreateReviewRequest) (*dto.ReviewResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// --- Встроенная логика ValidateReviewRequest ---
	if req.Rating < 1 || req.Rating > 5 {
		return nil, ErrInvalidReviewRating
	}
	if len(req.ReviewText) > 2000 {
		return nil, errors.New("review text too long")
	}
	if req.EmployerID == req.ModelID {
		return nil, ErrSelfReviewNotAllowed
	}

	// ✅ Передаем tx
	employer, err := s.userRepo.FindByID(tx, req.EmployerID)
	if err != nil {
		return nil, errors.New("employer not found")
	}
	// ✅ Передаем tx
	model, err := s.userRepo.FindByID(tx, req.ModelID)
	if err != nil {
		return nil, errors.New("model not found")
	}

	if employer.Role != models.UserRoleEmployer {
		return nil, errors.New("only employers can create reviews")
	}
	if model.Role != models.UserRoleModel {
		return nil, errors.New("can only review models")
	}
	// --- Конец валидации ---

	if userID != req.EmployerID {
		return nil, errors.New("only the employer can create reviews")
	}

	var castingID string
	if req.CastingID != nil {
		castingID = *req.CastingID
	}
	// ✅ Передаем tx
	canReview, err := s.CanUserReview(tx, req.EmployerID, req.ModelID, castingID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !canReview {
		return nil, errors.New("cannot create review for this casting")
	}

	review := &models.Review{
		ModelID:    req.ModelID,
		EmployerID: req.EmployerID,
		CastingID:  req.CastingID,
		Rating:     req.Rating,
		ReviewText: req.ReviewText,
	}

	// ✅ Передаем tx
	if err := s.reviewRepo.CreateReview(tx, review); err != nil {
		return nil, apperrors.InternalError(err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Отправляем уведомление *после* коммита, передаем 'db' (пул)
	go s.sendReviewNotification(db, req.ModelID, req.CastingID)

	// ✅ Возвращаем полный DTO, используя GetReview (и передавая 'db')
	return s.GetReview(db, review.ID)
}

// GetReview - 'db' добавлен
func (s *reviewService) GetReview(db *gorm.DB, reviewID string) (*dto.ReviewResponse, error) {
	// ✅ Используем 'db' из параметра
	review, err := s.reviewRepo.FindReviewByID(db, reviewID)
	if err != nil {
		return nil, handleReviewError(err)
	}
	return s.buildReviewResponse(review), nil
}

// GetModelReviews - 'db' добавлен
func (s *reviewService) GetModelReviews(db *gorm.DB, modelID string, page, pageSize int) (*dto.ReviewListResponse, error) {
	// ✅ Используем 'db' из параметра
	reviews, total, err := s.reviewRepo.FindReviewsWithPagination(db, modelID, page, pageSize)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}

	return &dto.ReviewListResponse{
		Reviews:    reviewResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calculateTotalPages(total, pageSize),
	}, nil
}

// GetEmployerReviews - 'db' добавлен
func (s *reviewService) GetEmployerReviews(db *gorm.DB, employerID string, page, pageSize int) (*dto.ReviewListResponse, error) {
	// ✅ Используем 'db' из параметра
	reviews, err := s.reviewRepo.FindReviewsByEmployer(db, employerID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// (Логика пагинации в памяти)
	total := int64(len(reviews))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(reviews) {
		start = len(reviews)
	}
	if end > len(reviews) {
		end = len(reviews)
	}
	paginated := reviews[start:end]

	var reviewResponses []*dto.ReviewResponse
	for _, review := range paginated {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}

	return &dto.ReviewListResponse{
		Reviews:    reviewResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calculateTotalPages(total, pageSize),
	}, nil
}

// GetCastingReviews - 'db' добавлен
func (s *reviewService) GetCastingReviews(db *gorm.DB, castingID string) ([]*dto.ReviewResponse, error) {
	// ✅ Используем 'db' из параметра
	reviews, err := s.reviewRepo.FindReviewsByCasting(db, castingID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}
	return reviewResponses, nil
}

// UpdateReview - 'db' добавлен
func (s *reviewService) UpdateReview(db *gorm.DB, userID, reviewID string, req *dto.UpdateReviewRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	review, err := s.reviewRepo.FindReviewByID(tx, reviewID)
	if err != nil {
		return handleReviewError(err)
	}

	if review.EmployerID != userID {
		return errors.New("only review author can update the review")
	}

	if req.Rating != nil {
		review.Rating = *req.Rating
	}
	if req.ReviewText != nil {
		review.ReviewText = *req.ReviewText
	}

	// ✅ Передаем tx
	if err := s.reviewRepo.UpdateReview(tx, review); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteReview - 'db' добавлен
func (s *reviewService) DeleteReview(db *gorm.DB, userID, reviewID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	review, err := s.reviewRepo.FindReviewByID(tx, reviewID)
	if err != nil {
		return handleReviewError(err)
	}

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, userID)
	if err != nil {
		return handleReviewError(err)
	}

	if review.EmployerID != userID && user.Role != models.UserRoleAdmin {
		return errors.New("only review author or admin can delete the review")
	}

	// ✅ Передаем tx
	if err := s.reviewRepo.DeleteReview(tx, reviewID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ---------------- Rating Operations ----------------

// GetModelRating - 'db' добавлен
func (s *reviewService) GetModelRating(db *gorm.DB, modelID string) (*dto.RatingResponse, error) {
	// ✅ Используем 'db' из параметра
	stats, err := s.reviewRepo.GetModelRatingStats(db, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildRatingResponse(stats), nil
}

// GetEmployerRating - 'db' добавлен
func (s *reviewService) GetEmployerRating(db *gorm.DB, employerID string) (*dto.RatingResponse, error) {
	// ✅ Используем 'db' из параметра
	stats, err := s.reviewRepo.GetEmployerRatingStats(db, employerID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildRatingResponse(stats), nil
}

// GetModelRatingStats - 'db' добавлен
func (s *reviewService) GetModelRatingStats(db *gorm.DB, modelID string) (*repositories.RatingStats, error) {
	// ✅ Используем 'db' из параметра
	return s.reviewRepo.GetModelRatingStats(db, modelID)
}

// GetEmployerRatingStats - 'db' добавлен
func (s *reviewService) GetEmployerRatingStats(db *gorm.DB, employerID string) (*repositories.RatingStats, error) {
	// ✅ Используем 'db' из параметра
	return s.reviewRepo.GetEmployerRatingStats(db, employerID)
}

// ---------------- Validation ----------------

// CanUserReview - (уже был 'db')
func (s *reviewService) CanUserReview(db *gorm.DB, employerID, modelID, castingID string) (bool, error) {
	// ✅ Передаем db
	return s.reviewRepo.CanCreateReview(db, employerID, modelID, castingID)
}

// IsUserReviewable - 'db' добавлен
func (s *reviewService) IsUserReviewable(db *gorm.DB, userID string, userRole models.UserRole) (bool, error) {
	// ✅ Используем 'db' из параметра
	return s.reviewRepo.IsUserReviewable(db, userID, userRole)
}

// ---------------- Admin Operations ----------------

// GetAllReviews - 'db' добавлен
func (s *reviewService) GetAllReviews(db *gorm.DB, page, pageSize int) (*dto.ReviewListResponse, error) {
	offset := (page - 1) * pageSize
	// ✅ Используем 'db' из параметра
	reviews, err := s.reviewRepo.FindAllReviews(db, pageSize, offset)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	// ✅ Используем 'db' из параметра
	total, err := s.reviewRepo.CountAllReviews(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}

	return &dto.ReviewListResponse{
		Reviews:    reviewResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calculateTotalPages(total, pageSize),
	}, nil
}

// GetRecentReviews - 'db' добавлен
func (s *reviewService) GetRecentReviews(db *gorm.DB, limit int) ([]*dto.ReviewResponse, error) {
	// ✅ Используем 'db' из параметра
	reviews, err := s.reviewRepo.FindRecentReviews(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}

	return reviewResponses, nil
}

// GetPlatformReviewStats - 'db' добавлен
func (s *reviewService) GetPlatformReviewStats(db *gorm.DB) (*repositories.PlatformReviewStats, error) {
	// ✅ Используем 'db' из параметра
	return s.reviewRepo.GetPlatformReviewStats(db)
}

// DeleteReviewByAdmin - 'db' добавлен
func (s *reviewService) DeleteReviewByAdmin(db *gorm.DB, adminID, reviewID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleReviewError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	// ✅ Передаем tx
	if err := s.reviewRepo.DeleteReview(tx, reviewID); err != nil {
		return handleReviewError(err)
	}
	return tx.Commit().Error
}

// ---------------- Additional Features ----------------

// GetReviewSummary - 'db' добавлен
func (s *reviewService) GetReviewSummary(db *gorm.DB, modelID string) (*repositories.ReviewSummary, error) {
	// ✅ Используем 'db' из параметра
	return s.reviewRepo.GetReviewSummary(db, modelID)
}

// SearchReviews - 'db' добавлен
func (s *reviewService) SearchReviews(db *gorm.DB, criteria dto.ReviewSearchCriteria) (*dto.ReviewListResponse, error) {
	var reviews []models.Review
	var err error

	// ✅ Используем 'db' из параметра
	if criteria.UserRole == string(models.UserRoleModel) {
		reviews, err = s.reviewRepo.FindReviewsByModel(db, criteria.UserID)
	} else if criteria.UserRole == string(models.UserRoleEmployer) {
		reviews, err = s.reviewRepo.FindReviewsByEmployer(db, criteria.UserID)
	} else {
		// ✅ Используем 'db' из параметра
		allReviews, totalErr := s.reviewRepo.FindAllReviews(db, 1000, 0)
		if totalErr != nil {
			return nil, totalErr
		}
		reviews = allReviews
	}

	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// (Фильтрация в памяти)
	filtered := s.applyReviewFilters(reviews, criteria)

	total := int64(len(filtered))
	start := (criteria.Page - 1) * criteria.PageSize
	end := start + criteria.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	if criteria.Page <= 0 {
		start = 0
	}
	if criteria.PageSize <= 0 {
		end = 0
	}

	var paginated []*dto.ReviewResponse
	if start < end {
		for _, review := range filtered[start:end] {
			paginated = append(paginated, s.buildReviewResponse(&review))
		}
	}

	return &dto.ReviewListResponse{
		Reviews:    paginated,
		Total:      total,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: calculateTotalPages(total, criteria.PageSize),
	}, nil
}

// GetUserReviewStats - 'db' добавлен
func (s *reviewService) GetUserReviewStats(db *gorm.DB, userID string, userRole models.UserRole) (*dto.UserReviewStats, error) {
	var stats *repositories.RatingStats
	var err error

	// ✅ Используем 'db' из параметра
	if userRole == models.UserRoleModel {
		stats, err = s.reviewRepo.GetModelRatingStats(db, userID)
	} else if userRole == models.UserRoleEmployer {
		stats, err = s.reviewRepo.GetEmployerRatingStats(db, userID)
	} else {
		return nil, errors.New("invalid user role for review stats")
	}
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	positive := int64(0)
	for rating, count := range stats.RatingCounts {
		if rating >= 4 {
			positive += count
		}
	}

	return &dto.UserReviewStats{
		TotalReviews:    stats.TotalReviews,
		AverageRating:   stats.AverageRating,
		PositiveReviews: positive,
		ResponseRate:    0,
		Ranking:         0,
	}, nil
}

// ---------------- Helper Methods ----------------

// (buildReviewResponse - чистая функция, без изменений)
func (s *reviewService) buildReviewResponse(review *models.Review) *dto.ReviewResponse {
	resp := &dto.ReviewResponse{
		ID:         review.ID,
		ModelID:    review.ModelID,
		EmployerID: review.EmployerID,
		CastingID:  review.CastingID,
		Rating:     review.Rating,
		ReviewText: review.ReviewText,
		CreatedAt:  review.CreatedAt,
		UpdatedAt:  review.UpdatedAt,
	}
	if review.Model.ID != "" {
		resp.Model = &dto.ModelInfo{
			ID:   review.Model.ID,
			Name: review.Model.Name,
			City: review.Model.City,
		}
	}
	if review.Employer.ID != "" {
		resp.Employer = &dto.EmployerInfo{
			ID:          review.Employer.ID,
			CompanyName: review.Employer.CompanyName,
			City:        review.Employer.City,
			IsVerified:  review.Employer.IsVerified,
		}
	}
	if review.Casting != nil && review.Casting.ID != "" {
		resp.Casting = &dto.CastingInfo{
			ID:    review.Casting.ID,
			Title: review.Casting.Title,
			City:  review.Casting.City,
		}
	}
	return resp
}

// (buildRatingResponse - чистая функция, без изменений)
func (s *reviewService) buildRatingResponse(stats *repositories.RatingStats) *dto.RatingResponse {
	breakdown := make(map[int]int)
	for rating, count := range stats.RatingCounts {
		breakdown[rating] = int(count)
	}
	return &dto.RatingResponse{
		AverageRating:   stats.AverageRating,
		TotalReviews:    stats.TotalReviews,
		RatingBreakdown: breakdown,
		RecentReviews:   stats.RecentReviews,
	}
}

// (applyReviewFilters - чистая функция, без изменений)
func (s *reviewService) applyReviewFilters(reviews []models.Review, criteria dto.ReviewSearchCriteria) []models.Review {
	var filtered []models.Review
	for _, review := range reviews {
		if criteria.MinRating > 0 && review.Rating < criteria.MinRating {
			continue
		}
		if criteria.MaxRating > 0 && review.Rating > criteria.MaxRating {
			continue
		}
		if !criteria.DateFrom.IsZero() && review.CreatedAt.Before(criteria.DateFrom) {
			continue
		}
		if !criteria.DateTo.IsZero() && review.CreatedAt.After(criteria.DateTo) {
			continue
		}
		if criteria.HasText != nil {
			hasText := len(review.ReviewText) > 0
			if *criteria.HasText != hasText {
				continue
			}
		}
		filtered = append(filtered, review)
	}
	return filtered
}

// (sendReviewNotification - 'db' уже был)
func (s *reviewService) sendReviewNotification(db *gorm.DB, modelID string, castingID *string) {
	title := "кастинг"
	if castingID != nil {
		// ✅ Передаем db
		casting, err := s.castingRepo.FindCastingByID(db, *castingID)
		if err == nil {
			title = casting.Title
		}
	}

	// ✅ Передаем db
	err := s.notificationRepo.CreateResponseStatusNotification(db,
		modelID,
		title,
		models.ResponseStatusAccepted,
	)
	if err != nil {
		fmt.Printf("Failed to send review notification: %v\n", err)
	}
}

// (handleReviewError - хелпер, без изменений)
func handleReviewError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrReviewNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}
