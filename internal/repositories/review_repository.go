package repositories

import (
	"errors"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrReviewNotFound       = errors.New("review not found")
	ErrReviewAlreadyExists  = errors.New("review already exists for this casting")
	ErrSelfReviewNotAllowed = errors.New("cannot review yourself")
	ErrInvalidReviewRating  = errors.New("rating must be between 1 and 5")
)

type ReviewRepository interface {
	// Review operations
	CreateReview(review *models.Review) error
	FindReviewByID(id string) (*models.Review, error)
	FindReviewsByModel(modelID string) ([]models.Review, error)
	FindReviewsByEmployer(employerID string) ([]models.Review, error)
	FindReviewsByCasting(castingID string) ([]models.Review, error)
	FindReviewByCastingAndAuthor(castingID, authorID string) (*models.Review, error)
	UpdateReview(review *models.Review) error
	DeleteReview(id string) error

	// Rating operations
	CalculateModelRating(modelID string) (float64, error)
	CalculateEmployerRating(employerID string) (float64, error)
	GetModelRatingStats(modelID string) (*RatingStats, error)
	GetEmployerRatingStats(employerID string) (*RatingStats, error)

	// Review validation
	CanCreateReview(employerID, modelID, castingID string) (bool, error)
	ValidateReview(review *models.Review) error

	// Admin operations
	FindAllReviews(limit, offset int) ([]models.Review, error)
	CountAllReviews() (int64, error)
	FindRecentReviews(limit int) ([]models.Review, error)
	GetPlatformReviewStats() (*PlatformReviewStats, error)

	// Additional methods
	UpdateModelRating(modelID string) error
	UpdateEmployerRating(employerID string) error
	FindReviewsWithPagination(modelID string, page, pageSize int) ([]models.Review, int64, error)
	GetReviewSummary(modelID string) (*ReviewSummary, error)
	IsUserReviewable(userID string, userRole models.UserRole) (bool, error)
}

type ReviewRepositoryImpl struct {
	db *gorm.DB
}

// Rating statistics
type RatingStats struct {
	AverageRating float64       `json:"average_rating"`
	TotalReviews  int64         `json:"total_reviews"`
	RatingCounts  map[int]int64 `json:"rating_counts"`  // 1-5 stars count
	RecentReviews int64         `json:"recent_reviews"` // Last 30 days
}

// Platform review statistics
type PlatformReviewStats struct {
	TotalReviews          int64   `json:"total_reviews"`
	TotalModelReviews     int64   `json:"total_model_reviews"`
	TotalEmployerReviews  int64   `json:"total_employer_reviews"`
	AverageModelRating    float64 `json:"average_model_rating"`
	AverageEmployerRating float64 `json:"average_employer_rating"`
	RecentReviews         int64   `json:"recent_reviews"`   // Last 30 days
	PositiveReviews       int64   `json:"positive_reviews"` // 4-5 stars
}

// Review summary structure
type ReviewSummary struct {
	AverageRating      float64       `json:"average_rating"`
	TotalReviews       int64         `json:"total_reviews"`
	RecentReviews      int64         `json:"recent_reviews"`
	RatingDistribution map[int]int64 `json:"rating_distribution"`
	ResponseRate       float64       `json:"response_rate,omitempty"`
}

func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &ReviewRepositoryImpl{db: db}
}

// Review operations

func (r *ReviewRepositoryImpl) CreateReview(review *models.Review) error {
	// Validate review before creation
	if err := r.ValidateReview(review); err != nil {
		return err
	}

	// Check if review already exists for this casting
	if review.CastingID != nil {
		var existing models.Review
		if err := r.db.Where("casting_id = ? AND employer_id = ?", *review.CastingID, review.EmployerID).
			First(&existing).Error; err == nil {
			return ErrReviewAlreadyExists
		}
	}

	// Используем транзакцию для создания отзыва и обновления рейтингов
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Создаем отзыв
		if err := tx.Create(review).Error; err != nil {
			return err
		}

		// Обновляем рейтинг модели
		if err := r.updateModelRatingInTransaction(tx, review.ModelID); err != nil {
			return err
		}

		// Обновляем рейтинг работодателя
		if err := r.updateEmployerRatingInTransaction(tx, review.EmployerID); err != nil {
			return err
		}

		return nil
	})
}

func (r *ReviewRepositoryImpl) FindReviewByID(id string) (*models.Review, error) {
	var review models.Review
	err := r.db.Preload("Model").Preload("Employer").Preload("Casting").
		First(&review, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepositoryImpl) FindReviewsByModel(modelID string) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Employer").Preload("Casting").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewsByEmployer(employerID string) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Model").Preload("Casting").
		Where("employer_id = ?", employerID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewsByCasting(castingID string) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Model").Preload("Employer").
		Where("casting_id = ?", castingID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewByCastingAndAuthor(castingID, authorID string) (*models.Review, error) {
	var review models.Review
	err := r.db.Preload("Model").Preload("Employer").Preload("Casting").
		Where("casting_id = ? AND employer_id = ?", castingID, authorID).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepositoryImpl) UpdateReview(review *models.Review) error {
	// Validate review before update
	if err := r.ValidateReview(review); err != nil {
		return err
	}

	// Используем транзакцию для обновления отзыва и рейтингов
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Получаем старый отзыв для сравнения
		var oldReview models.Review
		if err := tx.First(&oldReview, "id = ?", review.ID).Error; err != nil {
			return err
		}

		// Обновляем отзыв
		result := tx.Model(review).Updates(map[string]interface{}{
			"rating":      review.Rating,
			"review_text": review.ReviewText,
			"updated_at":  time.Now(),
		})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrReviewNotFound
		}

		// Если изменился рейтинг, обновляем рейтинги пользователей
		if oldReview.Rating != review.Rating {
			// Обновляем рейтинг модели
			if err := r.updateModelRatingInTransaction(tx, review.ModelID); err != nil {
				return err
			}

			// Обновляем рейтинг работодателя
			if err := r.updateEmployerRatingInTransaction(tx, review.EmployerID); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ReviewRepositoryImpl) DeleteReview(id string) error {
	// Используем транзакцию для удаления отзыва и обновления рейтингов
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Получаем отзыв перед удалением
		var review models.Review
		if err := tx.First(&review, "id = ?", id).Error; err != nil {
			return ErrReviewNotFound
		}

		// Удаляем отзыв
		result := tx.Where("id = ?", id).Delete(&models.Review{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrReviewNotFound
		}

		// Обновляем рейтинг модели
		if err := r.updateModelRatingInTransaction(tx, review.ModelID); err != nil {
			return err
		}

		// Обновляем рейтинг работодателя
		if err := r.updateEmployerRatingInTransaction(tx, review.EmployerID); err != nil {
			return err
		}

		return nil
	})
}

// Rating operations

func (r *ReviewRepositoryImpl) CalculateModelRating(modelID string) (float64, error) {
	var avgRating float64
	err := r.db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

func (r *ReviewRepositoryImpl) CalculateEmployerRating(employerID string) (float64, error) {
	var avgRating float64
	err := r.db.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

func (r *ReviewRepositoryImpl) GetModelRatingStats(modelID string) (*RatingStats, error) {
	var stats RatingStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews and average rating
	if err := r.db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COUNT(*) as total_reviews, COALESCE(AVG(rating), 0) as average_rating").
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	// Rating counts (1-5 stars)
	stats.RatingCounts = make(map[int]int64)
	for i := 1; i <= 5; i++ {
		var count int64
		if err := r.db.Model(&models.Review{}).Where("model_id = ? AND rating = ?", modelID, i).
			Count(&count).Error; err != nil {
			return nil, err
		}
		stats.RatingCounts[i] = count
	}

	// Recent reviews (last 30 days)
	if err := r.db.Model(&models.Review{}).Where("model_id = ? AND created_at >= ?", modelID, monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *ReviewRepositoryImpl) GetEmployerRatingStats(employerID string) (*RatingStats, error) {
	var stats RatingStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews and average rating
	if err := r.db.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COUNT(*) as total_reviews, COALESCE(AVG(rating), 0) as average_rating").
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	// Rating counts (1-5 stars)
	stats.RatingCounts = make(map[int]int64)
	for i := 1; i <= 5; i++ {
		var count int64
		if err := r.db.Model(&models.Review{}).Where("employer_id = ? AND rating = ?", employerID, i).
			Count(&count).Error; err != nil {
			return nil, err
		}
		stats.RatingCounts[i] = count
	}

	// Recent reviews (last 30 days)
	if err := r.db.Model(&models.Review{}).Where("employer_id = ? AND created_at >= ?", employerID, monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Review validation

func (r *ReviewRepositoryImpl) CanCreateReview(employerID, modelID, castingID string) (bool, error) {
	// Check if users are different
	if employerID == modelID {
		return false, ErrSelfReviewNotAllowed
	}

	// Check if casting exists and was completed
	if castingID != "" {
		var casting models.Casting
		if err := r.db.First(&casting, "id = ?", castingID).Error; err != nil {
			return false, errors.New("casting not found")
		}

		// Check if casting is completed
		if casting.Status != models.CastingStatusClosed {
			return false, errors.New("can only review completed castings")
		}

		// Check if employer was the owner of the casting
		if casting.EmployerID != employerID {
			return false, errors.New("only casting owner can leave review")
		}

		// Check if model participated in the casting
		var response models.CastingResponse
		if err := r.db.Where("casting_id = ? AND model_id = ? AND status = ?",
			castingID, modelID, models.ResponseStatusAccepted).First(&response).Error; err != nil {
			return false, errors.New("model did not participate in this casting")
		}
	}

	return true, nil
}

func (r *ReviewRepositoryImpl) ValidateReview(review *models.Review) error {
	// Validate rating
	if review.Rating < 1 || review.Rating > 5 {
		return ErrInvalidReviewRating
	}

	// Validate review text length
	if len(review.ReviewText) > 2000 {
		return errors.New("review text too long")
	}

	// Check if employer and model are different
	if review.EmployerID == review.ModelID {
		return ErrSelfReviewNotAllowed
	}

	return nil
}

// Admin operations

func (r *ReviewRepositoryImpl) FindAllReviews(limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Model").Preload("Employer").Preload("Casting").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) CountAllReviews() (int64, error) {
	var count int64
	err := r.db.Model(&models.Review{}).Count(&count).Error
	return count, err
}

func (r *ReviewRepositoryImpl) FindRecentReviews(limit int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Model").Preload("Employer").Preload("Casting").
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) GetPlatformReviewStats() (*PlatformReviewStats, error) {
	var stats PlatformReviewStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews
	if err := r.db.Model(&models.Review{}).Count(&stats.TotalReviews).Error; err != nil {
		return nil, err
	}

	// Model reviews
	if err := r.db.Model(&models.Review{}).Where("model_id IS NOT NULL").Count(&stats.TotalModelReviews).Error; err != nil {
		return nil, err
	}

	// Employer reviews
	if err := r.db.Model(&models.Review{}).Where("employer_id IS NOT NULL").Count(&stats.TotalEmployerReviews).Error; err != nil {
		return nil, err
	}

	// Average model rating
	if err := r.db.Model(&models.Review{}).Where("model_id IS NOT NULL").
		Select("COALESCE(AVG(rating), 0)").Scan(&stats.AverageModelRating).Error; err != nil {
		return nil, err
	}

	// Average employer rating
	if err := r.db.Model(&models.Review{}).Where("employer_id IS NOT NULL").
		Select("COALESCE(AVG(rating), 0)").Scan(&stats.AverageEmployerRating).Error; err != nil {
		return nil, err
	}

	// Recent reviews
	if err := r.db.Model(&models.Review{}).Where("created_at >= ?", monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	// Positive reviews (4-5 stars)
	if err := r.db.Model(&models.Review{}).Where("rating >= ?", 4).
		Count(&stats.PositiveReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Additional methods for business logic

func (r *ReviewRepositoryImpl) UpdateModelRating(modelID string) error {
	newRating, err := r.CalculateModelRating(modelID)
	if err != nil {
		return err
	}

	// Update model profile rating
	return r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) UpdateEmployerRating(employerID string) error {
	newRating, err := r.CalculateEmployerRating(employerID)
	if err != nil {
		return err
	}

	// Update employer profile rating
	return r.db.Model(&models.EmployerProfile{}).Where("user_id = ?", employerID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) FindReviewsWithPagination(modelID string, page, pageSize int) ([]models.Review, int64, error) {
	var reviews []models.Review

	// Get total count
	var total int64
	if err := r.db.Model(&models.Review{}).Where("model_id = ?", modelID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	err := r.db.Preload("Employer").Preload("Casting").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&reviews).Error

	return reviews, total, err
}

func (r *ReviewRepositoryImpl) GetReviewSummary(modelID string) (*ReviewSummary, error) {
	var summary ReviewSummary

	// Get rating stats
	ratingStats, err := r.GetModelRatingStats(modelID)
	if err != nil {
		return nil, err
	}

	summary.AverageRating = ratingStats.AverageRating
	summary.TotalReviews = ratingStats.TotalReviews
	summary.RatingDistribution = ratingStats.RatingCounts

	// Get recent activity
	monthAgo := time.Now().AddDate(0, -1, 0)
	if err := r.db.Model(&models.Review{}).Where("model_id = ? AND created_at >= ?", modelID, monthAgo).
		Count(&summary.RecentReviews).Error; err != nil {
		return nil, err
	}

	// Get response rate (if we track response to reviews)
	// This would require additional fields in the review model
	summary.ResponseRate = 0.0 // Placeholder for future implementation

	return &summary, nil
}

// Method to check if user can be reviewed
func (r *ReviewRepositoryImpl) IsUserReviewable(userID string, userRole models.UserRole) (bool, error) {
	// For models, check if they have completed castings
	if userRole == models.UserRoleModel {
		var completedCastings int64
		err := r.db.Model(&models.CastingResponse{}).
			Joins("LEFT JOIN castings ON casting_responses.casting_id = castings.id").
			Where("casting_responses.model_id = ? AND casting_responses.status = ? AND castings.status = ?",
				userID, models.ResponseStatusAccepted, models.CastingStatusClosed).
			Count(&completedCastings).Error

		if err != nil {
			return false, err
		}

		return completedCastings > 0, nil
	}

	// For employers, check if they have completed castings
	if userRole == models.UserRoleEmployer {
		var completedCastings int64
		err := r.db.Model(&models.Casting{}).
			Where("employer_id = ? AND status = ?", userID, models.CastingStatusClosed).
			Count(&completedCastings).Error

		if err != nil {
			return false, err
		}

		return completedCastings > 0, nil
	}

	return false, nil
}

// Helper methods for transaction operations

func (r *ReviewRepositoryImpl) updateModelRatingInTransaction(tx *gorm.DB, modelID string) error {
	newRating, err := r.calculateModelRatingInTransaction(tx, modelID)
	if err != nil {
		return err
	}

	return tx.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) updateEmployerRatingInTransaction(tx *gorm.DB, employerID string) error {
	newRating, err := r.calculateEmployerRatingInTransaction(tx, employerID)
	if err != nil {
		return err
	}

	return tx.Model(&models.EmployerProfile{}).Where("user_id = ?", employerID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) calculateModelRatingInTransaction(tx *gorm.DB, modelID string) (float64, error) {
	var avgRating float64
	err := tx.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

func (r *ReviewRepositoryImpl) calculateEmployerRatingInTransaction(tx *gorm.DB, employerID string) (float64, error) {
	var avgRating float64
	err := tx.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}
