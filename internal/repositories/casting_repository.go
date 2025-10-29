package repositories

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrCastingNotFound = errors.New("casting not found")
)

type CastingRepository interface {
	// Casting operations
	CreateCasting(casting *models.Casting) error
	FindCastingByID(id string) (*models.Casting, error)
	FindCastingsByEmployer(employerID string) ([]models.Casting, error)
	UpdateCasting(casting *models.Casting) error
	UpdateCastingStatus(castingID string, status models.CastingStatus) error
	DeleteCasting(id string) error
	IncrementCastingViews(castingID string) error
	SearchCastings(criteria CastingSearchCriteria) ([]models.Casting, int64, error)
	FindActiveCastings(limit int) ([]models.Casting, error)
	FindCastingsByCity(city string, limit int) ([]models.Casting, error)
	FindExpiredCastings() ([]models.Casting, error)
	GetCastingStats(employerID string) (*CastingStats, error)

	// Matching operations
	FindCastingsForMatching(criteria MatchingCriteria) ([]models.Casting, error)
	// Analytics methods
	GetPlatformCastingStats(dateFrom, dateTo time.Time) (*PlatformCastingStats, error)
	GetMatchingStats(dateFrom, dateTo time.Time) (*MatchingStats, error)
	GetCastingDistributionByCity() (map[string]int64, error)
	GetActiveCastingsCount() (int64, error)
	GetPopularCategories(limit int) ([]CategoryCount, error)
}

type CastingRepositoryImpl struct {
	db *gorm.DB
}

// Search criteria for castings
type CastingSearchCriteria struct {
	Query      string     `form:"query"`
	City       string     `form:"city"`
	Categories []string   `form:"categories[]"`
	Gender     string     `form:"gender"`
	MinAge     *int       `form:"min_age"`
	MaxAge     *int       `form:"max_age"`
	MinHeight  *int       `form:"min_height"`
	MaxHeight  *int       `form:"max_height"`
	MinSalary  *int       `form:"min_salary"`
	MaxSalary  *int       `form:"max_salary"`
	JobType    string     `form:"job_type"`
	Status     string     `form:"status"`
	EmployerID string     `form:"employer_id"`
	DateFrom   *time.Time `form:"date_from"`
	DateTo     *time.Time `form:"date_to"`
	Page       int        `form:"page" binding:"min=1"`
	PageSize   int        `form:"page_size" binding:"min=1,max=100"`
	SortBy     string     `form:"sort_by"`    // created_at, salary, casting_date
	SortOrder  string     `form:"sort_order"` // asc, desc
}

// Criteria for matching algorithm
type MatchingCriteria struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
	Gender     string   `json:"gender"`
	MinAge     *int     `json:"min_age"`
	MaxAge     *int     `json:"max_age"`
	MinHeight  *int     `json:"min_height"`
	MaxHeight  *int     `json:"max_height"`
	JobType    string   `json:"job_type"`
	Limit      int      `json:"limit" binding:"min=1,max=100"`
}

// Statistics for castings
type CastingStats struct {
	TotalCastings    int64 `json:"total_castings"`
	ActiveCastings   int64 `json:"active_castings"`
	DraftCastings    int64 `json:"draft_castings"`
	ClosedCastings   int64 `json:"closed_castings"`
	TotalViews       int64 `json:"total_views"`
	TotalResponses   int64 `json:"total_responses"`
	PendingResponses int64 `json:"pending_responses"`
}

type PlatformCastingStats struct {
	TotalCastings   int64   `json:"totalCastings"`
	ActiveCastings  int64   `json:"activeCastings"`
	SuccessRate     float64 `json:"successRate"`
	AvgResponseRate float64 `json:"avgResponseRate"`
	AvgResponseTime float64 `json:"avgResponseTime"`
}

type MatchingStats struct {
	TotalMatches    int64   `json:"totalMatches"`
	AvgMatchScore   float64 `json:"avgMatchScore"`
	AvgSatisfaction float64 `json:"avgSatisfaction"`
	MatchRate       float64 `json:"matchRate"`
	ResponseRate    float64 `json:"responseRate"`
	TimeToMatch     float64 `json:"timeToMatch"` // in hours
}

type CategoryCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

func NewCastingRepository(db *gorm.DB) CastingRepository {
	return &CastingRepositoryImpl{db: db}
}

// Casting operations

func (r *CastingRepositoryImpl) CreateCasting(casting *models.Casting) error {
	return r.db.Create(casting).Error
}

func (r *CastingRepositoryImpl) FindCastingByID(id string) (*models.Casting, error) {
	var casting models.Casting
	err := r.db.Preload("Employer").Preload("Responses").Preload("Responses.Model").
		First(&casting, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCastingNotFound
		}
		return nil, err
	}
	return &casting, nil
}

func (r *CastingRepositoryImpl) FindCastingsByEmployer(employerID string) ([]models.Casting, error) {
	var castings []models.Casting
	err := r.db.Preload("Responses", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Model")
	}).Where("employer_id = ?", employerID).
		Order("created_at DESC").
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) UpdateCasting(casting *models.Casting) error {
	result := r.db.Model(casting).Updates(map[string]interface{}{
		"title":            casting.Title,
		"description":      casting.Description,
		"payment_min":      casting.PaymentMin,
		"payment_max":      casting.PaymentMax,
		"casting_date":     casting.CastingDate,
		"casting_time":     casting.CastingTime,
		"address":          casting.Address,
		"city":             casting.City,
		"categories":       casting.Categories,
		"gender":           casting.Gender,
		"age_min":          casting.AgeMin,
		"age_max":          casting.AgeMax,
		"height_min":       casting.HeightMin,
		"height_max":       casting.HeightMax,
		"weight_min":       casting.WeightMin,
		"weight_max":       casting.WeightMax,
		"clothing_size":    casting.ClothingSize,
		"shoe_size":        casting.ShoeSize,
		"experience_level": casting.ExperienceLevel,
		"languages":        casting.Languages,
		"job_type":         casting.JobType,
		"status":           casting.Status,
		"updated_at":       time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCastingNotFound
	}
	return nil
}

func (r *CastingRepositoryImpl) UpdateCastingStatus(castingID string, status models.CastingStatus) error {
	result := r.db.Model(&models.Casting{}).Where("id = ?", castingID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCastingNotFound
	}
	return nil
}

func (r *CastingRepositoryImpl) DeleteCasting(id string) error {
	// Use transaction to delete casting and related responses
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete responses first
		if err := tx.Where("casting_id = ?", id).Delete(&models.CastingResponse{}).Error; err != nil {
			return err
		}

		// Delete casting
		result := tx.Where("id = ?", id).Delete(&models.Casting{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrCastingNotFound
		}
		return nil
	})
}

func (r *CastingRepositoryImpl) IncrementCastingViews(castingID string) error {
	return r.db.Model(&models.Casting{}).Where("id = ?", castingID).
		Update("views", gorm.Expr("views + ?", 1)).Error
}

func (r *CastingRepositoryImpl) SearchCastings(criteria CastingSearchCriteria) ([]models.Casting, int64, error) {
	var castings []models.Casting
	query := r.db.Model(&models.Casting{}).Preload("Employer")

	// Apply filters based on TZ requirements
	if criteria.Query != "" {
		search := "%" + criteria.Query + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", search, search)
	}

	if criteria.City != "" {
		query = query.Where("city = ?", criteria.City)
	}

	if criteria.Gender != "" {
		query = query.Where("gender = ?", criteria.Gender)
	}

	if criteria.JobType != "" {
		query = query.Where("job_type = ?", criteria.JobType)
	}

	if criteria.Status != "" {
		query = query.Where("status = ?", criteria.Status)
	} else {
		// По умолчанию показываем только активные кастинги
		query = query.Where("status = ?", models.CastingStatusActive)
	}

	if criteria.EmployerID != "" {
		query = query.Where("employer_id = ?", criteria.EmployerID)
	}

	if criteria.MinAge != nil {
		query = query.Where("age_min >= ?", criteria.MinAge)
	}

	if criteria.MaxAge != nil {
		query = query.Where("age_max <= ?", criteria.MaxAge)
	}

	if criteria.MinHeight != nil {
		query = query.Where("height_min >= ?", criteria.MinHeight)
	}

	if criteria.MaxHeight != nil {
		query = query.Where("height_max <= ?", criteria.MaxHeight)
	}

	if criteria.MinSalary != nil {
		query = query.Where("payment_min >= ?", criteria.MinSalary)
	}

	if criteria.MaxSalary != nil {
		query = query.Where("payment_max <= ?", criteria.MaxSalary)
	}

	if criteria.DateFrom != nil {
		query = query.Where("casting_date >= ?", criteria.DateFrom)
	}

	if criteria.DateTo != nil {
		query = query.Where("casting_date <= ?", criteria.DateTo)
	}

	// ✅ PostgreSQL JSONB operations для категорий
	if len(criteria.Categories) > 0 {
		categoryConditions := []string{}
		categoryArgs := []interface{}{}

		for _, category := range criteria.Categories {
			categoryConditions = append(categoryConditions, "categories::jsonb @> ?")
			categoryArgs = append(categoryArgs, datatypes.JSON(`["`+category+`"]`))
		}

		query = query.Where("("+strings.Join(categoryConditions, " OR ")+")", categoryArgs...)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortField := getCastingSortField(criteria.SortBy)
	sortOrder := getSortOrder(criteria.SortOrder)
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Limit(limit).Offset(offset).Find(&castings).Error
	return castings, total, err
}

func (r *CastingRepositoryImpl) FindActiveCastings(limit int) ([]models.Casting, error) {
	var castings []models.Casting
	err := r.db.Preload("Employer").
		Where("status = ?", models.CastingStatusActive).
		Where("casting_date IS NULL OR casting_date >= ?", time.Now()).
		Order("created_at DESC").
		Limit(limit).
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) FindCastingsByCity(city string, limit int) ([]models.Casting, error) {
	var castings []models.Casting
	err := r.db.Preload("Employer").
		Where("city = ? AND status = ?", city, models.CastingStatusActive).
		Where("casting_date IS NULL OR casting_date >= ?", time.Now()).
		Order("created_at DESC").
		Limit(limit).
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) FindExpiredCastings() ([]models.Casting, error) {
	var castings []models.Casting
	err := r.db.Where("status = ? AND casting_date < ?",
		models.CastingStatusActive, time.Now()).Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) GetCastingStats(employerID string) (*CastingStats, error) {
	var stats CastingStats

	// Total castings
	if err := r.db.Model(&models.Casting{}).Where("employer_id = ?", employerID).
		Count(&stats.TotalCastings).Error; err != nil {
		return nil, err
	}

	// Active castings
	if err := r.db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusActive).Count(&stats.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// Draft castings
	if err := r.db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusDraft).Count(&stats.DraftCastings).Error; err != nil {
		return nil, err
	}

	// Closed castings
	if err := r.db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusClosed).Count(&stats.ClosedCastings).Error; err != nil {
		return nil, err
	}

	// Total views
	if err := r.db.Model(&models.Casting{}).Where("employer_id = ?", employerID).
		Select("COALESCE(SUM(views), 0)").Scan(&stats.TotalViews).Error; err != nil {
		return nil, err
	}

	// Total responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id IN (?)",
		r.db.Model(&models.Casting{}).Select("id").Where("employer_id = ?", employerID)).
		Count(&stats.TotalResponses).Error; err != nil {
		return nil, err
	}

	// Pending responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id IN (?) AND status = ?",
		r.db.Model(&models.Casting{}).Select("id").Where("employer_id = ?", employerID),
		models.ResponseStatusPending).Count(&stats.PendingResponses).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Matching operations

func (r *CastingRepositoryImpl) FindCastingsForMatching(criteria MatchingCriteria) ([]models.Casting, error) {
	var castings []models.Casting
	query := r.db.Model(&models.Casting{}).Where("status = ?", models.CastingStatusActive)

	// Apply matching criteria
	if criteria.City != "" {
		query = query.Where("city = ?", criteria.City)
	}

	if criteria.Gender != "" {
		query = query.Where("gender = ?", criteria.Gender)
	}

	if criteria.JobType != "" {
		query = query.Where("job_type = ?", criteria.JobType)
	}

	if criteria.MinAge != nil {
		query = query.Where("age_min >= ?", criteria.MinAge)
	}

	if criteria.MaxAge != nil {
		query = query.Where("age_max <= ?", criteria.MaxAge)
	}

	if criteria.MinHeight != nil {
		query = query.Where("height_min >= ?", criteria.MinHeight)
	}

	if criteria.MaxHeight != nil {
		query = query.Where("height_max <= ?", criteria.MaxHeight)
	}

	// ✅ PostgreSQL JSONB operations для категорий
	if len(criteria.Categories) > 0 {
		categoryConditions := []string{}
		categoryArgs := []interface{}{}

		for _, category := range criteria.Categories {
			categoryConditions = append(categoryConditions, "categories::jsonb @> ?")
			categoryArgs = append(categoryArgs, datatypes.JSON(`["`+category+`"]`))
		}

		query = query.Where("("+strings.Join(categoryConditions, " OR ")+")", categoryArgs...)
	}

	err := query.Order("created_at DESC").Limit(criteria.Limit).Find(&castings).Error
	return castings, err
}

// Analytics methods

func (r *CastingRepositoryImpl) GetPlatformCastingStats(dateFrom, dateTo time.Time) (*PlatformCastingStats, error) {
	var stats PlatformCastingStats

	// Total castings in period
	if err := r.db.Model(&models.Casting{}).
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
		Count(&stats.TotalCastings).Error; err != nil {
		return nil, err
	}

	// Active castings
	if err := r.db.Model(&models.Casting{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", models.CastingStatusActive, dateFrom, dateTo).
		Count(&stats.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// Calculate success rate (castings with responses)
	var castingsWithResponses int64
	subquery := r.db.Model(&models.CastingResponse{}).
		Select("DISTINCT casting_id").
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo)

	if err := r.db.Model(&models.Casting{}).
		Where("id IN (?) AND created_at BETWEEN ? AND ?", subquery, dateFrom, dateTo).
		Count(&castingsWithResponses).Error; err != nil {
		return nil, err
	}

	if stats.TotalCastings > 0 {
		stats.SuccessRate = float64(castingsWithResponses) / float64(stats.TotalCastings) * 100
	}

	// Average response rate calculation would require more complex business logic
	// For now, setting placeholder values
	stats.AvgResponseRate = 0.0
	stats.AvgResponseTime = 0.0

	return &stats, nil
}

func (r *CastingRepositoryImpl) GetMatchingStats(dateFrom, dateTo time.Time) (*MatchingStats, error) {
	var stats MatchingStats

	// Total matches (responses in period)
	if err := r.db.Model(&models.CastingResponse{}).
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
		Count(&stats.TotalMatches).Error; err != nil {
		return nil, err
	}

	// Placeholder values for complex calculations
	// These would require additional tables/fields for match scoring and satisfaction
	stats.AvgMatchScore = 0.0
	stats.AvgSatisfaction = 0.0
	stats.MatchRate = 0.0
	stats.ResponseRate = 0.0
	stats.TimeToMatch = 0.0

	return &stats, nil
}

func (r *CastingRepositoryImpl) GetCastingDistributionByCity() (map[string]int64, error) {
	type CityCount struct {
		City  string
		Count int64
	}

	var cityCounts []CityCount
	result := make(map[string]int64)

	err := r.db.Model(&models.Casting{}).
		Select("city, COUNT(*) as count").
		Where("status = ?", models.CastingStatusActive).
		Group("city").
		Find(&cityCounts).Error

	if err != nil {
		return nil, err
	}

	for _, cc := range cityCounts {
		result[cc.City] = cc.Count
	}

	return result, nil
}

func (r *CastingRepositoryImpl) GetActiveCastingsCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Casting{}).
		Where("status = ?", models.CastingStatusActive).
		Count(&count).Error
	return count, err
}

func (r *CastingRepositoryImpl) GetPopularCategories(limit int) ([]CategoryCount, error) {
	// This is a simplified implementation
	// In a real scenario, you'd need to extract categories from JSONB field
	var categories []CategoryCount

	// Using raw SQL for JSONB array extraction (PostgreSQL specific)
	query := `
		SELECT category as name, COUNT(*) as count
		FROM (
			SELECT jsonb_array_elements_text(categories) as category
			FROM castings
			WHERE status = ? AND categories IS NOT NULL
		) as extracted_categories
		GROUP BY category
		ORDER BY count DESC
		LIMIT ?
	`

	err := r.db.Raw(query, models.CastingStatusActive, limit).Scan(&categories).Error
	return categories, err
}

// Helper functions

func getCastingSortField(sortBy string) string {
	switch sortBy {
	case "salary":
		return "payment_max"
	case "casting_date":
		return "casting_date"
	case "created_at":
		return "created_at"
	case "views":
		return "views"
	default:
		return "created_at"
	}
}
