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
	CreateCasting(db *gorm.DB, casting *models.Casting) error
	FindCastingByID(db *gorm.DB, id string) (*models.Casting, error)
	FindCastingsByEmployer(db *gorm.DB, employerID string) ([]models.Casting, error)
	UpdateCasting(db *gorm.DB, casting *models.Casting) error
	UpdateCastingStatus(db *gorm.DB, castingID string, status models.CastingStatus) error
	DeleteCasting(db *gorm.DB, id string) error
	IncrementCastingViews(db *gorm.DB, castingID string) error
	SearchCastings(db *gorm.DB, criteria CastingSearchCriteria) ([]models.Casting, int64, error)
	FindActiveCastings(db *gorm.DB, limit int) ([]models.Casting, error)
	FindCastingsByCity(db *gorm.DB, city string, limit int) ([]models.Casting, error)
	FindExpiredCastings(db *gorm.DB) ([]models.Casting, error)
	GetCastingStats(db *gorm.DB, employerID string) (*CastingStats, error)

	// Matching operations
	FindCastingsForMatching(db *gorm.DB, criteria MatchingCriteria) ([]models.Casting, error)

	// GetPlatformCastingStats (для analyticsService.GetCastingAnalytics)
	GetPlatformCastingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*PlatformCastingStats, error)

	// GetMatchingStats (для analyticsService.GetMatchingAnalytics)
	GetMatchingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*MatchingStats, error)

	// GetActiveCastingsCount (для analyticsService.GetRealTimeMetrics)
	GetActiveCastingsCount(db *gorm.DB) (int64, error)

	// GetPopularCategories (для analyticsService.GetPopularCategories)
	GetPopularCategories(db *gorm.DB, limit int) ([]PopularCategoryStat, error)

	GetCastingDistributionByCity(db *gorm.DB) ([]CityDistributionStat, error)
}

type CastingRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
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

// PopularCategoryStat - DTO для GetPopularCategories
type PopularCategoryStat struct {
	Name  string
	Count int64
}

type CityDistributionStat struct {
	City  string `json:"city"`
	Count int64  `json:"count"`
}

// ✅ Конструктор не принимает db
func NewCastingRepository() CastingRepository {
	return &CastingRepositoryImpl{}
}

// Casting operations

func (r *CastingRepositoryImpl) CreateCasting(db *gorm.DB, casting *models.Casting) error {
	// ✅ Используем 'db' из параметра
	return db.Create(casting).Error
}

func (r *CastingRepositoryImpl) FindCastingByID(db *gorm.DB, id string) (*models.Casting, error) {
	var casting models.Casting
	// ✅ Используем 'db' из параметра
	err := db.Preload("Employer").Preload("Responses").Preload("Responses.Model").
		First(&casting, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCastingNotFound
		}
		return nil, err
	}
	return &casting, nil
}

func (r *CastingRepositoryImpl) FindCastingsByEmployer(db *gorm.DB, employerID string) ([]models.Casting, error) {
	var castings []models.Casting
	// ✅ Используем 'db' из параметра
	err := db.Preload("Responses", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Model")
	}).Where("employer_id = ?", employerID).
		Order("created_at DESC").
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) UpdateCasting(db *gorm.DB, casting *models.Casting) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(casting).Updates(map[string]interface{}{
		"title":            casting.Title,
		"description":      casting.Description,
		"payment_min":      casting.PaymentMin,
		"payment_max":      casting.PaymentMax,
		"event_date":       casting.CastingDate,
		"event_time":       casting.CastingTime,
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

func (r *CastingRepositoryImpl) UpdateCastingStatus(db *gorm.DB, castingID string, status models.CastingStatus) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.Casting{}).Where("id = ?", castingID).Updates(map[string]interface{}{
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

func (r *CastingRepositoryImpl) DeleteCasting(db *gorm.DB, id string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Delete responses first
	// ✅ Используем 'db' из параметра
	if err := db.Where("casting_id = ?", id).Delete(&models.CastingResponse{}).Error; err != nil {
		return err
	}

	// Delete casting
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.Casting{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCastingNotFound
	}
	return nil
}

func (r *CastingRepositoryImpl) IncrementCastingViews(db *gorm.DB, castingID string) error {
	// ✅ Используем 'db' из параметра
	return db.Model(&models.Casting{}).Where("id = ?", castingID).
		Update("views", gorm.Expr("views + ?", 1)).Error
}

func (r *CastingRepositoryImpl) SearchCastings(db *gorm.DB, criteria CastingSearchCriteria) ([]models.Casting, int64, error) {
	var castings []models.Casting
	query := db.Model(&models.Casting{}).Preload("Employer")

	// ... (фильтры по query, city, gender, job_type, status, employer_id, age, height, salary)
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

	// =======================
	// 2. ✅ ИСПРАВЛЕНА ОШИБКА
	// =======================
	if criteria.DateFrom != nil {
		query = query.Where("event_date >= ?", criteria.DateFrom) // ❌ БЫЛО: "casting_date"
	}
	if criteria.DateTo != nil {
		query = query.Where("event_date <= ?", criteria.DateTo) // ❌ БЫЛО: "casting_date"
	}
	// ... (фильтр по categories)
	if len(criteria.Categories) > 0 {
		categoryConditions := []string{}
		categoryArgs := []interface{}{}

		for _, category := range criteria.Categories {
			categoryConditions = append(categoryConditions, "categories::jsonb @> ?")
			categoryArgs = append(categoryArgs, datatypes.JSON(`["`+category+`"]`))
		}

		query = query.Where("("+strings.Join(categoryConditions, " OR ")+")", categoryArgs...)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// =======================
	// 3. ✅ ИСПРАВЛЕНА ОШИБКА (в getCastingSortField)
	// =======================
	sortField := getCastingSortField(criteria.SortBy)
	sortOrder := getSortOrder(criteria.SortOrder)
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Limit(limit).Offset(offset).Find(&castings).Error
	return castings, total, err
}

func (r *CastingRepositoryImpl) FindActiveCastings(db *gorm.DB, limit int) ([]models.Casting, error) {
	var castings []models.Casting
	// =======================
	// 4. ✅ ИСПРАВЛЕНА ОШИБКА
	// =======================
	err := db.Preload("Employer").
		Where("status = ?", models.CastingStatusActive).
		Where("event_date IS NULL OR event_date >= ?", time.Now()). // ❌ БЫЛО: "casting_date"
		Order("created_at DESC").
		Limit(limit).
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) FindCastingsByCity(db *gorm.DB, city string, limit int) ([]models.Casting, error) {
	var castings []models.Casting
	// =======================
	// 5. ✅ ИСПРАВЛЕНА ОШИБКА
	// =======================
	err := db.Preload("Employer").
		Where("city = ? AND status = ?", city, models.CastingStatusActive).
		Where("event_date IS NULL OR event_date >= ?", time.Now()). // ❌ БЫЛО: "casting_date"
		Order("created_at DESC").
		Limit(limit).
		Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) FindExpiredCastings(db *gorm.DB) ([]models.Casting, error) {
	var castings []models.Casting
	// =======================
	// 6. ✅ ИСПРАВЛЕНА ОШИБКА (которая вызвала ваш сбой)
	// =======================
	err := db.Where("status = ? AND event_date < ?", // ❌ БЫЛО: "casting_date"
		models.CastingStatusActive, time.Now()).Find(&castings).Error
	return castings, err
}

func (r *CastingRepositoryImpl) GetCastingStats(db *gorm.DB, employerID string) (*CastingStats, error) {
	var stats CastingStats

	// Total castings
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Casting{}).Where("employer_id = ?", employerID).
		Count(&stats.TotalCastings).Error; err != nil {
		return nil, err
	}

	// Active castings
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusActive).Count(&stats.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// Draft castings
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusDraft).Count(&stats.DraftCastings).Error; err != nil {
		return nil, err
	}

	// Closed castings
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Casting{}).Where("employer_id = ? AND status = ?",
		employerID, models.CastingStatusClosed).Count(&stats.ClosedCastings).Error; err != nil {
		return nil, err
	}

	// Total views
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Casting{}).Where("employer_id = ?", employerID).
		Select("COALESCE(SUM(views), 0)").Scan(&stats.TotalViews).Error; err != nil {
		return nil, err
	}

	// Total responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id IN (?)",
		db.Model(&models.Casting{}).Select("id").Where("employer_id = ?", employerID)).
		Count(&stats.TotalResponses).Error; err != nil {
		return nil, err
	}

	// Pending responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id IN (?) AND status = ?",
		db.Model(&models.Casting{}).Select("id").Where("employer_id = ?", employerID),
		models.ResponseStatusPending).Count(&stats.PendingResponses).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Matching operations

func (r *CastingRepositoryImpl) FindCastingsForMatching(db *gorm.DB, criteria MatchingCriteria) ([]models.Casting, error) {
	var castings []models.Casting
	// ✅ Используем 'db' из параметра
	query := db.Model(&models.Casting{}).Where("status = ?", models.CastingStatusActive)

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

	// ✅ Используем 'db' (query)
	err := query.Order("created_at DESC").Limit(criteria.Limit).Find(&castings).Error
	return castings, err
}

// Helper functions

// === 🚀 РЕАЛИЗАЦИИ-ЗАГЛУШКИ ===

func (r *CastingRepositoryImpl) GetPlatformCastingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*PlatformCastingStats, error) {
	var stats PlatformCastingStats

	// 1. TotalCastings
	if err := db.Model(&models.Casting{}).Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).Count(&stats.TotalCastings).Error; err != nil {
		return nil, err
	}

	// 2. ActiveCastings (созданные в этот период И активные СЕЙЧАС)
	if err := db.Model(&models.Casting{}).Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).Where("status = ?", models.CastingStatusActive).Count(&stats.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// 3. ClosedCastings (созданные в этот период И закрытые СЕЙЧАС)
	if err := db.Model(&models.Casting{}).Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).Where("status = ?", models.CastingStatusClosed).Count(&stats.ClosedCastings).Error; err != nil {
		return nil, err
	}

	// 4. AcceptedResponses (для кастингов, созданных в этот период)
	if err := db.Model(&models.CastingResponse{}).
		Where("casting_id IN (?)",
			db.Model(&models.Casting{}).Select("id").Where("created_at BETWEEN ? AND ?", dateFrom, dateTo),
		).
		Where("status = ?", models.ResponseStatusAccepted).
		Count(&stats.AcceptedResponses).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *CastingRepositoryImpl) GetMatchingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*MatchingStats, error) {
	// TODO: Реализовать SQL-запрос для сбора статистики по мэтчингу
	stats := &MatchingStats{
		TotalMatches:    0,
		AvgMatchScore:   0.0,
		AvgSatisfaction: 0.0,
		MatchRate:       0.0,
		TimeToMatch:     0.0,
		ResponseRate:    0.0,
	}
	// ...
	return stats, nil
}

func (r *CastingRepositoryImpl) GetActiveCastingsCount(db *gorm.DB) (int64, error) {
	var count int64
	// Исправлена логика: Активный = статус 'active' И дата еще не прошла
	err := db.Model(&models.Casting{}).
		Where("status = ?", models.CastingStatusActive).
		Where("event_date IS NULL OR event_date >= ?", time.Now()).
		Count(&count).Error
	return count, err
}

func (r *CastingRepositoryImpl) GetPopularCategories(db *gorm.DB, limit int) ([]PopularCategoryStat, error) {
	var results []PopularCategoryStat

	// Реализуем запрос, который был в TODO.
	// Он "разворачивает" JSON-массив 'categories' и считает вхождения
	err := db.Model(&models.Casting{}).
		Select("jsonb_array_elements_text(categories) as name, COUNT(*) as count").
		Where("status = ?", models.CastingStatusActive). // Считаем только активные
		Group("name").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error

	return results, err
}

func (r *CastingRepositoryImpl) GetCastingDistributionByCity(db *gorm.DB) ([]CityDistributionStat, error) {
	var results []CityDistributionStat

	err := db.Model(&models.Casting{}).
		Select("city, COUNT(*) as count").
		Where("status = ?", models.CastingStatusActive). // Считаем только активные
		Group("city").
		Order("count DESC").
		Limit(20). // Ограничим топ-20 городами
		Scan(&results).Error

	return results, err
}

func getCastingSortField(sortBy string) string {
	switch sortBy {
	case "salary":
		return "payment_max"
	case "casting_date":
		return "event_date"
	case "created_at":
		return "created_at"
	case "views":
		return "views"
	default:
		return "created_at"
	}
}
