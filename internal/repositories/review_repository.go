package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"time"

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
	CreateReview(db *gorm.DB, review *models.Review) error
	FindReviewByID(db *gorm.DB, id string) (*models.Review, error)
	FindReviewsByModel(db *gorm.DB, modelID string) ([]models.Review, error)
	FindReviewsByEmployer(db *gorm.DB, employerID string) ([]models.Review, error)
	FindReviewsByCasting(db *gorm.DB, castingID string) ([]models.Review, error)
	FindReviewByCastingAndAuthor(db *gorm.DB, castingID, authorID string) (*models.Review, error)
	UpdateReview(db *gorm.DB, review *models.Review) error
	DeleteReview(db *gorm.DB, id string) error

	// Rating operations
	CalculateModelRating(db *gorm.DB, modelID string) (float64, error)
	CalculateEmployerRating(db *gorm.DB, employerID string) (float64, error)
	GetModelRatingStats(db *gorm.DB, modelID string) (*RatingStats, error)
	GetEmployerRatingStats(db *gorm.DB, employerID string) (*RatingStats, error)

	// Review validation
	CanCreateReview(db *gorm.DB, employerID, modelID, castingID string) (bool, error)
	ValidateReview(db *gorm.DB, review *models.Review) error

	// Admin operations
	FindAllReviews(db *gorm.DB, limit, offset int) ([]models.Review, error)
	CountAllReviews(db *gorm.DB) (int64, error)
	FindRecentReviews(db *gorm.DB, limit int) ([]models.Review, error)
	GetPlatformReviewStats(db *gorm.DB) (*PlatformReviewStats, error)

	// Additional methods
	UpdateModelRating(db *gorm.DB, modelID string) error
	UpdateEmployerRating(db *gorm.DB, employerID string) error
	FindReviewsWithPagination(db *gorm.DB, modelID string, page, pageSize int) ([]models.Review, int64, error)
	GetReviewSummary(db *gorm.DB, modelID string) (*ReviewSummary, error)
	IsUserReviewable(db *gorm.DB, userID string, userRole models.UserRole) (bool, error)
}

type ReviewRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
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

// ✅ Конструктор не принимает db
func NewReviewRepository() ReviewRepository {
	return &ReviewRepositoryImpl{}
}

// Review operations

func (r *ReviewRepositoryImpl) CreateReview(db *gorm.DB, review *models.Review) error {
	// Validate review before creation
	// ✅ Передаем db, хотя он и не используется (для согласованности интерфейса)
	if err := r.ValidateReview(db, review); err != nil {
		return err
	}

	// Check if review already exists for this casting
	if review.CastingID != nil {
		var existing models.Review
		// ✅ Используем 'db' из параметра
		if err := db.Where("casting_id = ? AND employer_id = ?", *review.CastingID, review.EmployerID).
			First(&existing).Error; err == nil {
			return ErrReviewAlreadyExists
		}
	}

	// ✅ Вложенная транзакция удалена.
	// Создаем отзыв
	// ✅ Используем 'db' из параметра
	if err := db.Create(review).Error; err != nil {
		return err
	}

	// Обновляем рейтинг модели
	// ✅ Используем 'db' из параметра и переименованный хелпер
	if err := r.updateModelRatingInternal(db, review.ModelID); err != nil {
		return err
	}

	// Обновляем рейтинг работодателя
	// ✅ Используем 'db' из параметра и переименованный хелпер
	if err := r.updateEmployerRatingInternal(db, review.EmployerID); err != nil {
		return err
	}

	return nil
}

func (r *ReviewRepositoryImpl) FindReviewByID(db *gorm.DB, id string) (*models.Review, error) {
	var review models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Employer").Preload("Casting").
		First(&review, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepositoryImpl) FindReviewsByModel(db *gorm.DB, modelID string) ([]models.Review, error) {
	var reviews []models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Employer").Preload("Casting").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewsByEmployer(db *gorm.DB, employerID string) ([]models.Review, error) {
	var reviews []models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Casting").
		Where("employer_id = ?", employerID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewsByCasting(db *gorm.DB, castingID string) ([]models.Review, error) {
	var reviews []models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Employer").
		Where("casting_id = ?", castingID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) FindReviewByCastingAndAuthor(db *gorm.DB, castingID, authorID string) (*models.Review, error) {
	var review models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Employer").Preload("Casting").
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

func (r *ReviewRepositoryImpl) UpdateReview(db *gorm.DB, review *models.Review) error {
	// Validate review before update
	// ✅ Передаем db
	if err := r.ValidateReview(db, review); err != nil {
		return err
	}

	// ✅ Вложенная транзакция удалена.
	// Получаем старый отзыв для сравнения
	var oldReview models.Review
	// ✅ Используем 'db' из параметра
	if err := db.First(&oldReview, "id = ?", review.ID).Error; err != nil {
		return err
	}

	// Обновляем отзыв
	// ✅ Используем 'db' из параметра
	result := db.Model(review).Updates(map[string]interface{}{
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
		// ✅ Используем 'db' из параметра и переименованный хелпер
		if err := r.updateModelRatingInternal(db, review.ModelID); err != nil {
			return err
		}

		// Обновляем рейтинг работодателя
		// ✅ Используем 'db' из параметра и переименованный хелпер
		if err := r.updateEmployerRatingInternal(db, review.EmployerID); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReviewRepositoryImpl) DeleteReview(db *gorm.DB, id string) error {
	// ✅ Вложенная транзакция удалена.
	// Получаем отзыв перед удалением
	var review models.Review
	// ✅ Используем 'db' из параметра
	if err := db.First(&review, "id = ?", id).Error; err != nil {
		return ErrReviewNotFound
	}

	// Удаляем отзыв
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.Review{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrReviewNotFound
	}

	// Обновляем рейтинг модели
	// ✅ Используем 'db' из параметра и переименованный хелпер
	if err := r.updateModelRatingInternal(db, review.ModelID); err != nil {
		return err
	}

	// Обновляем рейтинг работодателя
	// ✅ Используем 'db' из параметра и переименованный хелпер
	if err := r.updateEmployerRatingInternal(db, review.EmployerID); err != nil {
		return err
	}

	return nil
}

// Rating operations

func (r *ReviewRepositoryImpl) CalculateModelRating(db *gorm.DB, modelID string) (float64, error) {
	var avgRating float64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

func (r *ReviewRepositoryImpl) CalculateEmployerRating(db *gorm.DB, employerID string) (float64, error) {
	var avgRating float64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

func (r *ReviewRepositoryImpl) GetModelRatingStats(db *gorm.DB, modelID string) (*RatingStats, error) {
	var stats RatingStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews and average rating
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COUNT(*) as total_reviews, COALESCE(AVG(rating), 0) as average_rating").
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	// Rating counts (1-5 stars)
	stats.RatingCounts = make(map[int]int64)
	for i := 1; i <= 5; i++ {
		var count int64
		// ✅ Используем 'db' из параметра
		if err := db.Model(&models.Review{}).Where("model_id = ? AND rating = ?", modelID, i).
			Count(&count).Error; err != nil {
			return nil, err
		}
		stats.RatingCounts[i] = count
	}

	// Recent reviews (last 30 days)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id = ? AND created_at >= ?", modelID, monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *ReviewRepositoryImpl) GetEmployerRatingStats(db *gorm.DB, employerID string) (*RatingStats, error) {
	var stats RatingStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews and average rating
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COUNT(*) as total_reviews, COALESCE(AVG(rating), 0) as average_rating").
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	// Rating counts (1-5 stars)
	stats.RatingCounts = make(map[int]int64)
	for i := 1; i <= 5; i++ {
		var count int64
		// ✅ Используем 'db' из параметра
		if err := db.Model(&models.Review{}).Where("employer_id = ? AND rating = ?", employerID, i).
			Count(&count).Error; err != nil {
			return nil, err
		}
		stats.RatingCounts[i] = count
	}

	// Recent reviews (last 30 days)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("employer_id = ? AND created_at >= ?", employerID, monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Review validation

func (r *ReviewRepositoryImpl) CanCreateReview(db *gorm.DB, employerID, modelID, castingID string) (bool, error) {
	// Check if users are different
	if employerID == modelID {
		return false, ErrSelfReviewNotAllowed
	}

	// Check if casting exists and was completed
	if castingID != "" {
		var casting models.Casting
		// ✅ Используем 'db' из параметра
		if err := db.First(&casting, "id = ?", castingID).Error; err != nil {
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
		// ✅ Используем 'db' из параметра
		if err := db.Where("casting_id = ? AND model_id = ? AND status = ?",
			castingID, modelID, models.ResponseStatusAccepted).First(&response).Error; err != nil {
			return false, errors.New("model did not participate in this casting")
		}
	}

	return true, nil
}

// ✅ Метод теперь принимает 'db' (хотя и не использует), для согласованности интерфейса
func (r *ReviewRepositoryImpl) ValidateReview(db *gorm.DB, review *models.Review) error {
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

func (r *ReviewRepositoryImpl) FindAllReviews(db *gorm.DB, limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Employer").Preload("Casting").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) CountAllReviews(db *gorm.DB) (int64, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Review{}).Count(&count).Error
	return count, err
}

func (r *ReviewRepositoryImpl) FindRecentReviews(db *gorm.DB, limit int) ([]models.Review, error) {
	var reviews []models.Review
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Employer").Preload("Casting").
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews).Error
	return reviews, err
}

func (r *ReviewRepositoryImpl) GetPlatformReviewStats(db *gorm.DB) (*PlatformReviewStats, error) {
	var stats PlatformReviewStats
	monthAgo := time.Now().AddDate(0, -1, 0)

	// Total reviews
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Count(&stats.TotalReviews).Error; err != nil {
		return nil, err
	}

	// Model reviews
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id IS NOT NULL").Count(&stats.TotalModelReviews).Error; err != nil {
		return nil, err
	}

	// Employer reviews
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("employer_id IS NOT NULL").Count(&stats.TotalEmployerReviews).Error; err != nil {
		return nil, err
	}

	// Average model rating
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id IS NOT NULL").
		Select("COALESCE(AVG(rating), 0)").Scan(&stats.AverageModelRating).Error; err != nil {
		return nil, err
	}

	// Average employer rating
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("employer_id IS NOT NULL").
		Select("COALESCE(AVG(rating), 0)").Scan(&stats.AverageEmployerRating).Error; err != nil {
		return nil, err
	}

	// Recent reviews
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("created_at >= ?", monthAgo).
		Count(&stats.RecentReviews).Error; err != nil {
		return nil, err
	}

	// Positive reviews (4-5 stars)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("rating >= ?", 4).
		Count(&stats.PositiveReviews).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Additional methods for business logic

func (r *ReviewRepositoryImpl) UpdateModelRating(db *gorm.DB, modelID string) error {
	// ✅ Передаем 'db' в CalculateModelRating
	newRating, err := r.CalculateModelRating(db, modelID)
	if err != nil {
		return err
	}

	// Update model profile rating
	// ✅ Используем 'db' из параметра
	return db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) UpdateEmployerRating(db *gorm.DB, employerID string) error {
	// ✅ Передаем 'db' в CalculateEmployerRating
	newRating, err := r.CalculateEmployerRating(db, employerID)
	if err != nil {
		return err
	}

	// Update employer profile rating
	// ✅ Используем 'db' из параметра
	return db.Model(&models.EmployerProfile{}).Where("user_id = ?", employerID).
		Update("rating", newRating).Error
}

func (r *ReviewRepositoryImpl) FindReviewsWithPagination(db *gorm.DB, modelID string, page, pageSize int) ([]models.Review, int64, error) {
	var reviews []models.Review

	// Get total count
	var total int64
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id = ?", modelID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	// ✅ Используем 'db' из параметра
	err := db.Preload("Employer").Preload("Casting").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&reviews).Error

	return reviews, total, err
}

func (r *ReviewRepositoryImpl) GetReviewSummary(db *gorm.DB, modelID string) (*ReviewSummary, error) {
	var summary ReviewSummary

	// Get rating stats
	// ✅ Передаем 'db' в GetModelRatingStats
	ratingStats, err := r.GetModelRatingStats(db, modelID)
	if err != nil {
		return nil, err
	}

	summary.AverageRating = ratingStats.AverageRating
	summary.TotalReviews = ratingStats.TotalReviews
	summary.RatingDistribution = ratingStats.RatingCounts

	// Get recent activity
	monthAgo := time.Now().AddDate(0, -1, 0)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Review{}).Where("model_id = ? AND created_at >= ?", modelID, monthAgo).
		Count(&summary.RecentReviews).Error; err != nil {
		return nil, err
	}

	// Get response rate (if we track response to reviews)
	// This would require additional fields in the review model
	summary.ResponseRate = 0.0 // Placeholder for future implementation

	return &summary, nil
}

// Method to check if user can be reviewed
func (r *ReviewRepositoryImpl) IsUserReviewable(db *gorm.DB, userID string, userRole models.UserRole) (bool, error) {
	// For models, check if they have completed castings
	if userRole == models.UserRoleModel {
		var completedCastings int64
		// ✅ Используем 'db' из параметра
		err := db.Model(&models.CastingResponse{}).
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
		// ✅ Используем 'db' из параметра
		err := db.Model(&models.Casting{}).
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

// ✅ Переименован: updateModelRatingInTransaction -> updateModelRatingInternal
func (r *ReviewRepositoryImpl) updateModelRatingInternal(db *gorm.DB, modelID string) error {
	// ✅ Передаем 'db', хелпер переименован
	newRating, err := r.calculateModelRatingInternal(db, modelID)
	if err != nil {
		return err
	}

	// ✅ Используем 'db' из параметра
	return db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("rating", newRating).Error
}

// ✅ Переименован: updateEmployerRatingInTransaction -> updateEmployerRatingInternal
func (r *ReviewRepositoryImpl) updateEmployerRatingInternal(db *gorm.DB, employerID string) error {
	// ✅ Передаем 'db', хелпер переименован
	newRating, err := r.calculateEmployerRatingInternal(db, employerID)
	if err != nil {
		return err
	}

	// ✅ Используем 'db' из параметра
	return db.Model(&models.EmployerProfile{}).Where("user_id = ?", employerID).
		Update("rating", newRating).Error
}

// ✅ Переименован: calculateModelRatingInTransaction -> calculateModelRatingInternal
func (r *ReviewRepositoryImpl) calculateModelRatingInternal(db *gorm.DB, modelID string) (float64, error) {
	var avgRating float64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}

// ✅ Переименован: calculateEmployerRatingInTransaction -> calculateEmployerRatingInternal
func (r *ReviewRepositoryImpl) calculateEmployerRatingInternal(db *gorm.DB, employerID string) (float64, error) {
	var avgRating float64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Review{}).Where("employer_id = ?", employerID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error
	return avgRating, err
}
