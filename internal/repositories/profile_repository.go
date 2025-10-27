package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists for this user")
)

// Добавляем интерфейс ProfileRepository
type ProfileRepository interface {
	// ModelProfile operations
	CreateModelProfile(profile *models.ModelProfile) error
	FindModelProfileByID(id string) (*models.ModelProfile, error)
	FindModelProfileByUserID(userID string) (*models.ModelProfile, error)
	UpdateModelProfile(profile *models.ModelProfile) error
	UpdateModelProfileRating(modelID string, newRating float64) error
	IncrementModelProfileViews(modelID string) error
	DeleteModelProfile(id string) error
	SearchModelProfiles(criteria ModelSearchCriteria) ([]models.ModelProfile, int64, error)
	FindFeaturedModels(limit int) ([]models.ModelProfile, error)
	FindModelsByCity(city string) ([]models.ModelProfile, error)
	GetModelStats(modelID string) (*ModelStats, error)

	// EmployerProfile operations
	CreateEmployerProfile(profile *models.EmployerProfile) error
	FindEmployerProfileByID(id string) (*models.EmployerProfile, error)
	FindEmployerProfileByUserID(userID string) (*models.EmployerProfile, error)
	UpdateEmployerProfile(profile *models.EmployerProfile) error
	VerifyEmployerProfile(employerID string) error
	DeleteEmployerProfile(id string) error
	SearchEmployerProfiles(criteria EmployerSearchCriteria) ([]models.EmployerProfile, int64, error)
	FindEmployersWithActiveCastings(limit int) ([]models.EmployerProfile, error)

	// Дополнительные методы
	FindModelsByExactCategory(category string) ([]models.ModelProfile, error)
	FindModelsByMultipleLanguages(languages []string) ([]models.ModelProfile, error)
	UpdateModelCategories(modelID string, categories []string) error
	UpdateModelLanguages(modelID string, languages []string) error
}

type ProfileRepositoryImpl struct {
	db *gorm.DB
}

// Исправленные search criteria с правильными типами
type ModelSearchCriteria struct {
	Query         string   `form:"query"`
	City          string   `form:"city"`
	Categories    []string `form:"categories[]"`
	Gender        string   `form:"gender"`
	MinAge        *int     `form:"min_age"`
	MaxAge        *int     `form:"max_age"`
	MinHeight     *int     `form:"min_height"`
	MaxHeight     *int     `form:"max_height"`
	MinWeight     *int     `form:"min_weight"`
	MaxWeight     *int     `form:"max_weight"`
	MinPrice      *int     `form:"min_price"`
	MaxPrice      *int     `form:"max_price"`
	MinExperience *int     `form:"min_experience"`
	Languages     []string `form:"languages[]"`
	AcceptsBarter *bool    `form:"accepts_barter"`
	MinRating     *float64 `form:"min_rating"`
	IsPublic      *bool    `form:"is_public"`
	Page          int      `form:"page" binding:"min=1"`
	PageSize      int      `form:"page_size" binding:"min=1,max=100"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order"`
}

// Добавляем EmployerSearchCriteria
type EmployerSearchCriteria struct {
	Query       string `form:"query"`
	City        string `form:"city"`
	CompanyType string `form:"company_type"`
	IsVerified  *bool  `form:"is_verified"`
	Page        int    `form:"page" binding:"min=1"`
	PageSize    int    `form:"page_size" binding:"min=1,max=100"`
}

// Statistics for model profile
type ModelStats struct {
	TotalViews      int64   `json:"total_views"`
	AverageRating   float64 `json:"average_rating"`
	TotalReviews    int64   `json:"total_reviews"`
	PortfolioItems  int64   `json:"portfolio_items"`
	ActiveResponses int64   `json:"active_responses"`
	CompletedJobs   int64   `json:"completed_jobs"`
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &ProfileRepositoryImpl{db: db}
}

// ModelProfile operations

func (r *ProfileRepositoryImpl) CreateModelProfile(profile *models.ModelProfile) error {
	// Check if profile already exists for this user
	var existing models.ModelProfile
	if err := r.db.Where("user_id = ?", profile.UserID).First(&existing).Error; err == nil {
		return ErrProfileAlreadyExists
	}

	return r.db.Create(profile).Error
}

func (r *ProfileRepositoryImpl) FindModelProfileByID(id string) (*models.ModelProfile, error) {
	var profile models.ModelProfile
	err := r.db.Preload("PortfolioItems").Preload("PortfolioItems.Upload").
		First(&profile, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepositoryImpl) FindModelProfileByUserID(userID string) (*models.ModelProfile, error) {
	var profile models.ModelProfile
	err := r.db.Preload("PortfolioItems").Preload("PortfolioItems.Upload").
		Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepositoryImpl) UpdateModelProfile(profile *models.ModelProfile) error {
	result := r.db.Model(profile).Updates(map[string]interface{}{
		"name":            profile.Name,
		"age":             profile.Age,
		"height":          profile.Height,
		"weight":          profile.Weight,
		"gender":          profile.Gender,
		"experience":      profile.Experience,
		"hourly_rate":     profile.HourlyRate,
		"description":     profile.Description,
		"clothing_size":   profile.ClothingSize,
		"shoe_size":       profile.ShoeSize,
		"city":            profile.City,
		"languages":       profile.Languages,
		"categories":      profile.Categories,
		"barter_accepted": profile.BarterAccepted,
		"is_public":       profile.IsPublic,
		"updated_at":      time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ProfileRepositoryImpl) UpdateModelProfileRating(modelID string, newRating float64) error {
	result := r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).Update("rating", newRating)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ProfileRepositoryImpl) IncrementModelProfileViews(modelID string) error {
	return r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("profile_views", gorm.Expr("profile_views + ?", 1)).Error
}

func (r *ProfileRepositoryImpl) DeleteModelProfile(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.ModelProfile{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// 🎯 ИСПРАВЛЕННЫЙ метод поиска с PostgreSQL-специфичными операциями
func (r *ProfileRepositoryImpl) SearchModelProfiles(criteria ModelSearchCriteria) ([]models.ModelProfile, int64, error) {
	var profiles []models.ModelProfile
	query := r.db.Model(&models.ModelProfile{})

	// Apply privacy filter - только публичные профили для поиска
	if criteria.IsPublic == nil || *criteria.IsPublic {
		query = query.Where("is_public = ?", true)
	}

	// Text search
	if criteria.Query != "" {
		search := "%" + criteria.Query + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", search, search)
	}

	// Basic filters
	if criteria.City != "" {
		query = query.Where("city = ?", criteria.City)
	}

	if criteria.Gender != "" {
		query = query.Where("gender = ?", criteria.Gender)
	}

	if criteria.MinAge != nil {
		query = query.Where("age >= ?", *criteria.MinAge)
	}

	if criteria.MaxAge != nil {
		query = query.Where("age <= ?", *criteria.MaxAge)
	}

	if criteria.MinHeight != nil {
		query = query.Where("height >= ?", *criteria.MinHeight)
	}

	if criteria.MaxHeight != nil {
		query = query.Where("height <= ?", *criteria.MaxHeight)
	}

	if criteria.MinWeight != nil {
		query = query.Where("weight >= ?", *criteria.MinWeight)
	}

	if criteria.MaxWeight != nil {
		query = query.Where("weight <= ?", *criteria.MaxWeight)
	}

	if criteria.MinPrice != nil {
		query = query.Where("hourly_rate >= ?", *criteria.MinPrice)
	}

	if criteria.MaxPrice != nil {
		query = query.Where("hourly_rate <= ?", *criteria.MaxPrice)
	}

	if criteria.MinExperience != nil {
		query = query.Where("experience >= ?", *criteria.MinExperience)
	}

	if criteria.AcceptsBarter != nil {
		query = query.Where("barter_accepted = ?", *criteria.AcceptsBarter)
	}

	if criteria.MinRating != nil {
		query = query.Where("rating >= ?", *criteria.MinRating)
	}

	// ✅ PostgreSQL JSONB operations - ИСПРАВЛЕНО
	if len(criteria.Categories) > 0 {
		// Для JSONB массива: проверяем что categories содержит любой из указанных элементов
		categoryConditions := []string{}
		categoryArgs := []interface{}{}

		for _, category := range criteria.Categories {
			categoryConditions = append(categoryConditions, "categories::jsonb @> ?")
			// ИСПРАВЛЕНИЕ: правильное создание JSON массива
			categoryJSON, _ := json.Marshal([]string{category})
			categoryArgs = append(categoryArgs, datatypes.JSON(categoryJSON))
		}

		query = query.Where("("+strings.Join(categoryConditions, " OR ")+")", categoryArgs...)
	}

	if len(criteria.Languages) > 0 {
		// Для JSONB массива: проверяем что languages содержит любой из указанных элементов
		languageConditions := []string{}
		languageArgs := []interface{}{}

		for _, language := range criteria.Languages {
			languageConditions = append(languageConditions, "languages::jsonb @> ?")
			// ИСПРАВЛЕНИЕ: правильное создание JSON массива
			languageJSON, _ := json.Marshal([]string{language})
			languageArgs = append(languageArgs, datatypes.JSON(languageJSON))
		}

		query = query.Where("("+strings.Join(languageConditions, " OR ")+")", languageArgs...)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortField := getModelSortField(criteria.SortBy)
	sortOrder := getSortOrder(criteria.SortOrder)
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	// Preload limited portfolio items for performance
	err := query.Preload("PortfolioItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC").Limit(3) // Load first 3 portfolio items
	}).Preload("PortfolioItems.Upload").
		Limit(limit).Offset(offset).Find(&profiles).Error

	return profiles, total, err
}

// 🎯 Дополнительные методы для работы с JSONB

// FindModelsByExactCategory - поиск по точному совпадению категории
func (r *ProfileRepositoryImpl) FindModelsByExactCategory(category string) ([]models.ModelProfile, error) {
	var profiles []models.ModelProfile
	categoryJSON, _ := json.Marshal([]string{category})

	err := r.db.Where("categories::jsonb @> ?", datatypes.JSON(categoryJSON)).
		Where("is_public = ?", true).
		Order("rating DESC").
		Find(&profiles).Error
	return profiles, err
}

// FindModelsByMultipleLanguages - поиск по нескольким языкам
func (r *ProfileRepositoryImpl) FindModelsByMultipleLanguages(languages []string) ([]models.ModelProfile, error) {
	var profiles []models.ModelProfile

	conditions := []string{}
	args := []interface{}{}

	for _, lang := range languages {
		conditions = append(conditions, "languages::jsonb @> ?")
		langJSON, _ := json.Marshal([]string{lang})
		args = append(args, datatypes.JSON(langJSON))
	}

	query := strings.Join(conditions, " OR ")
	args = append([]interface{}{query}, args...)

	err := r.db.Where("is_public = ?", true).
		Where(args[0], args[1:]...).
		Order("rating DESC").
		Find(&profiles).Error

	return profiles, err
}

// UpdateModelCategories - обновление категорий модели
func (r *ProfileRepositoryImpl) UpdateModelCategories(modelID string, categories []string) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	result := r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("categories", datatypes.JSON(categoriesJSON))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// UpdateModelLanguages - обновление языков модели
func (r *ProfileRepositoryImpl) UpdateModelLanguages(modelID string, languages []string) error {
	languagesJSON, err := json.Marshal(languages)
	if err != nil {
		return fmt.Errorf("failed to marshal languages: %w", err)
	}

	result := r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Update("languages", datatypes.JSON(languagesJSON))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// Остальные методы остаются без изменений...
func (r *ProfileRepositoryImpl) FindFeaturedModels(limit int) ([]models.ModelProfile, error) {
	var profiles []models.ModelProfile
	err := r.db.Where("is_public = ? AND rating >= ?", true, 4.0).
		Order("rating DESC, profile_views DESC").
		Limit(limit).
		Preload("PortfolioItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC").Limit(1)
		}).
		Preload("PortfolioItems.Upload").
		Find(&profiles).Error
	return profiles, err
}

func (r *ProfileRepositoryImpl) FindModelsByCity(city string) ([]models.ModelProfile, error) {
	var profiles []models.ModelProfile
	err := r.db.Where("city = ? AND is_public = ?", city, true).
		Order("rating DESC").
		Preload("PortfolioItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC").Limit(2)
		}).
		Preload("PortfolioItems.Upload").
		Find(&profiles).Error
	return profiles, err
}

func (r *ProfileRepositoryImpl) GetModelStats(modelID string) (*ModelStats, error) {
	var stats ModelStats

	// Get profile views
	if err := r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Pluck("profile_views", &stats.TotalViews).Error; err != nil {
		return nil, err
	}

	// Get rating
	if err := r.db.Model(&models.ModelProfile{}).Where("id = ?", modelID).
		Pluck("rating", &stats.AverageRating).Error; err != nil {
		return nil, err
	}

	// Get review count
	if err := r.db.Model(&models.Review{}).Where("model_id = ?", modelID).
		Count(&stats.TotalReviews).Error; err != nil {
		return nil, err
	}

	// Get portfolio items count
	if err := r.db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Count(&stats.PortfolioItems).Error; err != nil {
		return nil, err
	}

	// Get active responses count (simplified)
	if err := r.db.Model(&models.CastingResponse{}).Where("model_id = ? AND status = ?", modelID, "pending").
		Count(&stats.ActiveResponses).Error; err != nil {
		return nil, err
	}

	// Get completed jobs count (simplified)
	if err := r.db.Model(&models.CastingResponse{}).Where("model_id = ? AND status = ?", modelID, "accepted").
		Count(&stats.CompletedJobs).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// EmployerProfile methods
func (r *ProfileRepositoryImpl) CreateEmployerProfile(profile *models.EmployerProfile) error {
	var existing models.EmployerProfile
	if err := r.db.Where("user_id = ?", profile.UserID).First(&existing).Error; err == nil {
		return ErrProfileAlreadyExists
	}
	return r.db.Create(profile).Error
}

func (r *ProfileRepositoryImpl) FindEmployerProfileByID(id string) (*models.EmployerProfile, error) {
	var profile models.EmployerProfile
	err := r.db.First(&profile, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepositoryImpl) FindEmployerProfileByUserID(userID string) (*models.EmployerProfile, error) {
	var profile models.EmployerProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepositoryImpl) UpdateEmployerProfile(profile *models.EmployerProfile) error {
	result := r.db.Model(profile).Updates(map[string]interface{}{
		"company_name":   profile.CompanyName,
		"contact_person": profile.ContactPerson,
		"phone":          profile.Phone,
		"website":        profile.Website,
		"city":           profile.City,
		"company_type":   profile.CompanyType,
		"description":    profile.Description,
		"is_verified":    profile.IsVerified,
		"updated_at":     time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ProfileRepositoryImpl) VerifyEmployerProfile(employerID string) error {
	result := r.db.Model(&models.EmployerProfile{}).Where("id = ?", employerID).Updates(map[string]interface{}{
		"is_verified": true,
		"updated_at":  time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ProfileRepositoryImpl) DeleteEmployerProfile(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.EmployerProfile{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ProfileRepositoryImpl) SearchEmployerProfiles(criteria EmployerSearchCriteria) ([]models.EmployerProfile, int64, error) {
	var profiles []models.EmployerProfile
	query := r.db.Model(&models.EmployerProfile{})

	if criteria.Query != "" {
		search := "%" + criteria.Query + "%"
		query = query.Where("company_name ILIKE ? OR description ILIKE ? OR contact_person ILIKE ?", search, search, search)
	}

	if criteria.City != "" {
		query = query.Where("city = ?", criteria.City)
	}

	if criteria.CompanyType != "" {
		query = query.Where("company_type = ?", criteria.CompanyType)
	}

	if criteria.IsVerified != nil {
		query = query.Where("is_verified = ?", *criteria.IsVerified)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Order("is_verified DESC, company_name ASC").
		Limit(limit).Offset(offset).Find(&profiles).Error

	return profiles, total, err
}

func (r *ProfileRepositoryImpl) FindEmployersWithActiveCastings(limit int) ([]models.EmployerProfile, error) {
	var profiles []models.EmployerProfile

	// Subquery to find employers with active castings
	subquery := r.db.Model(&models.Casting{}).Where("status = ?", models.CastingStatusActive).
		Select("DISTINCT employer_id")

	err := r.db.Where("id IN (?)", subquery).Order("is_verified DESC").
		Limit(limit).Find(&profiles).Error

	return profiles, err
}

// Helper functions

func getModelSortField(sortBy string) string {
	switch sortBy {
	case "rating":
		return "rating"
	case "price":
		return "hourly_rate"
	case "experience":
		return "experience"
	case "views":
		return "profile_views"
	case "created_at":
		return "created_at"
	default:
		return "rating"
	}
}

func getSortOrder(sortOrder string) string {
	if sortOrder == "asc" {
		return "ASC"
	}
	return "DESC"
}
