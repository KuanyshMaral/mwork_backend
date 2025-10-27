package services

import (
	"errors"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

var (
	ErrInvalidReviewRating  = errors.New("invalid review rating")
	ErrSelfReviewNotAllowed = errors.New("self-review is not allowed")
)

type ReviewService interface {
	// Review operations
	CreateReview(userID string, req *dto.CreateReviewRequest) (*dto.ReviewResponse, error)
	GetReview(reviewID string) (*dto.ReviewResponse, error)
	GetModelReviews(modelID string, page, pageSize int) (*dto.ReviewListResponse, error)
	GetEmployerReviews(employerID string, page, pageSize int) (*dto.ReviewListResponse, error)
	GetCastingReviews(castingID string) ([]*dto.ReviewResponse, error)
	UpdateReview(userID, reviewID string, req *dto.UpdateReviewRequest) error
	DeleteReview(userID, reviewID string) error

	// Rating operations
	GetModelRating(modelID string) (*dto.RatingResponse, error)
	GetEmployerRating(employerID string) (*dto.RatingResponse, error)
	GetModelRatingStats(modelID string) (*repositories.RatingStats, error)
	GetEmployerRatingStats(employerID string) (*repositories.RatingStats, error)

	// Validation and business logic
	CanUserReview(employerID, modelID, castingID string) (bool, error)
	IsUserReviewable(userID string, userRole models.UserRole) (bool, error)
	ValidateReviewRequest(req *dto.CreateReviewRequest) error

	// Admin operations
	GetAllReviews(page, pageSize int) (*dto.ReviewListResponse, error)
	GetRecentReviews(limit int) ([]*dto.ReviewResponse, error)
	GetPlatformReviewStats() (*repositories.PlatformReviewStats, error)
	DeleteReviewByAdmin(adminID, reviewID string) error

	// Additional features
	GetReviewSummary(modelID string) (*repositories.ReviewSummary, error)
	SearchReviews(criteria dto.ReviewSearchCriteria) (*dto.ReviewListResponse, error)
	GetUserReviewStats(userID string, userRole models.UserRole) (*dto.UserReviewStats, error)
}

type reviewService struct {
	reviewRepo       repositories.ReviewRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	notificationRepo repositories.NotificationRepository
}

func NewReviewService(
	reviewRepo repositories.ReviewRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	notificationRepo repositories.NotificationRepository,
) ReviewService {
	return &reviewService{
		reviewRepo:       reviewRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		notificationRepo: notificationRepo,
	}
}

// ---------------- Review Operations ----------------

func (s *reviewService) CreateReview(userID string, req *dto.CreateReviewRequest) (*dto.ReviewResponse, error) {
	if err := s.ValidateReviewRequest(req); err != nil {
		return nil, err
	}

	if userID != req.EmployerID {
		return nil, errors.New("only the employer can create reviews")
	}

	var castingID string
	if req.CastingID != nil {
		castingID = *req.CastingID
	}
	canReview, err := s.CanUserReview(req.EmployerID, req.ModelID, castingID)
	if err != nil {
		return nil, err
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

	if err := s.reviewRepo.CreateReview(review); err != nil {
		return nil, err
	}

	go s.notificationRepo.CreateResponseStatusNotification(
		req.ModelID,
		getCastingTitle(req.CastingID),
		models.ResponseStatusAccepted,
	)

	return s.buildReviewResponse(review), nil
}

func (s *reviewService) GetReview(reviewID string) (*dto.ReviewResponse, error) {
	review, err := s.reviewRepo.FindReviewByID(reviewID)
	if err != nil {
		return nil, err
	}
	return s.buildReviewResponse(review), nil
}

func (s *reviewService) GetModelReviews(modelID string, page, pageSize int) (*dto.ReviewListResponse, error) {
	reviews, total, err := s.reviewRepo.FindReviewsWithPagination(modelID, page, pageSize)
	if err != nil {
		return nil, err
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

func (s *reviewService) GetEmployerReviews(employerID string, page, pageSize int) (*dto.ReviewListResponse, error) {
	reviews, err := s.reviewRepo.FindReviewsByEmployer(employerID)
	if err != nil {
		return nil, err
	}

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

func (s *reviewService) GetCastingReviews(castingID string) ([]*dto.ReviewResponse, error) {
	reviews, err := s.reviewRepo.FindReviewsByCasting(castingID)
	if err != nil {
		return nil, err
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}
	return reviewResponses, nil
}

func (s *reviewService) UpdateReview(userID, reviewID string, req *dto.UpdateReviewRequest) error {
	review, err := s.reviewRepo.FindReviewByID(reviewID)
	if err != nil {
		return err
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

	return s.reviewRepo.UpdateReview(review)
}

func (s *reviewService) DeleteReview(userID, reviewID string) error {
	review, err := s.reviewRepo.FindReviewByID(reviewID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	if review.EmployerID != userID && user.Role != models.UserRoleAdmin {
		return errors.New("only review author or admin can delete the review")
	}

	return s.reviewRepo.DeleteReview(reviewID)
}

// ---------------- Rating Operations ----------------

func (s *reviewService) GetModelRating(modelID string) (*dto.RatingResponse, error) {
	stats, err := s.reviewRepo.GetModelRatingStats(modelID)
	if err != nil {
		return nil, err
	}
	return s.buildRatingResponse(stats), nil
}

func (s *reviewService) GetEmployerRating(employerID string) (*dto.RatingResponse, error) {
	stats, err := s.reviewRepo.GetEmployerRatingStats(employerID)
	if err != nil {
		return nil, err
	}
	return s.buildRatingResponse(stats), nil
}

func (s *reviewService) GetModelRatingStats(modelID string) (*repositories.RatingStats, error) {
	return s.reviewRepo.GetModelRatingStats(modelID)
}

func (s *reviewService) GetEmployerRatingStats(employerID string) (*repositories.RatingStats, error) {
	return s.reviewRepo.GetEmployerRatingStats(employerID)
}

// ---------------- Validation ----------------

func (s *reviewService) CanUserReview(employerID, modelID, castingID string) (bool, error) {
	return s.reviewRepo.CanCreateReview(employerID, modelID, castingID)
}

func (s *reviewService) IsUserReviewable(userID string, userRole models.UserRole) (bool, error) {
	return s.reviewRepo.IsUserReviewable(userID, userRole)
}

func (s *reviewService) ValidateReviewRequest(req *dto.CreateReviewRequest) error {
	if req.Rating < 1 || req.Rating > 5 {
		return ErrInvalidReviewRating
	}
	if len(req.ReviewText) > 2000 {
		return errors.New("review text too long")
	}
	if req.EmployerID == req.ModelID {
		return ErrSelfReviewNotAllowed
	}

	employer, err := s.userRepo.FindByID(req.EmployerID)
	if err != nil {
		return errors.New("employer not found")
	}
	model, err := s.userRepo.FindByID(req.ModelID)
	if err != nil {
		return errors.New("model not found")
	}

	if employer.Role != models.UserRoleEmployer {
		return errors.New("only employers can create reviews")
	}
	if model.Role != models.UserRoleModel {
		return errors.New("can only review models")
	}
	return nil
}

// ---------------- Admin Operations ----------------

func (s *reviewService) GetAllReviews(page, pageSize int) (*dto.ReviewListResponse, error) {
	offset := (page - 1) * pageSize
	reviews, err := s.reviewRepo.FindAllReviews(pageSize, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.reviewRepo.CountAllReviews()
	if err != nil {
		return nil, err
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

func (s *reviewService) GetRecentReviews(limit int) ([]*dto.ReviewResponse, error) {
	reviews, err := s.reviewRepo.FindRecentReviews(limit)
	if err != nil {
		return nil, err
	}

	var reviewResponses []*dto.ReviewResponse
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, s.buildReviewResponse(&review))
	}

	return reviewResponses, nil
}

func (s *reviewService) GetPlatformReviewStats() (*repositories.PlatformReviewStats, error) {
	return s.reviewRepo.GetPlatformReviewStats()
}

func (s *reviewService) DeleteReviewByAdmin(adminID, reviewID string) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	return s.reviewRepo.DeleteReview(reviewID)
}

// ---------------- Additional Features ----------------

func (s *reviewService) GetReviewSummary(modelID string) (*repositories.ReviewSummary, error) {
	return s.reviewRepo.GetReviewSummary(modelID)
}

func (s *reviewService) SearchReviews(criteria dto.ReviewSearchCriteria) (*dto.ReviewListResponse, error) {
	var reviews []models.Review
	var err error

	if criteria.UserRole == string(models.UserRoleModel) {
		reviews, err = s.reviewRepo.FindReviewsByModel(criteria.UserID)
	} else if criteria.UserRole == string(models.UserRoleEmployer) {
		reviews, err = s.reviewRepo.FindReviewsByEmployer(criteria.UserID)
	} else {
		allReviews, totalErr := s.reviewRepo.FindAllReviews(1000, 0)
		if totalErr != nil {
			return nil, totalErr
		}
		reviews = allReviews
	}

	if err != nil {
		return nil, err
	}

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

	var paginated []*dto.ReviewResponse
	for _, review := range filtered[start:end] {
		paginated = append(paginated, s.buildReviewResponse(&review))
	}

	return &dto.ReviewListResponse{
		Reviews:    paginated,
		Total:      total,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: calculateTotalPages(total, criteria.PageSize),
	}, nil
}

func (s *reviewService) GetUserReviewStats(userID string, userRole models.UserRole) (*dto.UserReviewStats, error) {
	var stats *repositories.RatingStats
	var err error

	if userRole == models.UserRoleModel {
		stats, err = s.reviewRepo.GetModelRatingStats(userID)
	} else if userRole == models.UserRoleEmployer {
		stats, err = s.reviewRepo.GetEmployerRatingStats(userID)
	} else {
		return nil, errors.New("invalid user role for review stats")
	}
	if err != nil {
		return nil, err
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

func getCastingTitle(castingID *string) string {
	if castingID == nil {
		return "кастинг"
	}
	return "кастинг"
}
