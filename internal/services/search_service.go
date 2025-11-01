package services

import (
	"errors"
	"gorm.io/gorm"
	"strings"

	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type SearchService interface {
	SearchCastings(db *gorm.DB, req *dto.SearchCastingsRequest) (*dto.PaginatedResponse, error)
	SearchCastingsAdvanced(db *gorm.DB, req *dto.AdvancedCastingSearchRequest) (*dto.PaginatedResponse, error)
	GetCastingSearchSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error)
	SearchModels(db *gorm.DB, req *dto.SearchModelsRequest) (*dto.PaginatedResponse, error)
	SearchModelsAdvanced(db *gorm.DB, req *dto.AdvancedModelSearchRequest) (*dto.PaginatedResponse, error)
	GetModelSearchSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error)
	SearchEmployers(db *gorm.DB, req *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error)
	UnifiedSearch(db *gorm.DB, req *dto.UnifiedSearchRequest) (*dto.UnifiedSearchResponse, error)
	GetSearchAutoComplete(db *gorm.DB, query string) (*dto.AutoCompleteResponse, error)
	GetPopularSearches(db *gorm.DB, limit int) ([]*dto.PopularSearch, error)
	GetSearchTrends(db *gorm.DB, days int) (*dto.SearchTrends, error)
	SaveSearchHistory(db *gorm.DB, userID, query, searchType string) error
	GetSearchHistory(db *gorm.DB, userID string, limit int) ([]*dto.SearchHistoryItem, error)
	ClearSearchHistory(db *gorm.DB, userID string) error
	GetSearchAnalytics(db *gorm.DB, days int) (*dto.SearchAnalytics, error)
	ReindexSearchData(db *gorm.DB, adminID string) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type searchService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	castingRepo   repositories.CastingRepository
	profileRepo   repositories.ProfileRepository
	portfolioRepo repositories.PortfolioRepository
	reviewRepo    repositories.ReviewRepository
	// TODO: Тебе понадобятся репозитории для истории поиска и аналитики
	// searchHistoryRepo repositories.SearchHistoryRepository
	// analyticsRepo     repositories.AnalyticsRepository
}

// ✅ Конструктор обновлен (db убран)
func NewSearchService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	castingRepo repositories.CastingRepository,
	profileRepo repositories.ProfileRepository,
	portfolioRepo repositories.PortfolioRepository,
	reviewRepo repositories.ReviewRepository,
) SearchService {
	return &searchService{
		// ❌ 'db: db,' УДАЛЕНО
		castingRepo:   castingRepo,
		profileRepo:   profileRepo,
		portfolioRepo: portfolioRepo,
		reviewRepo:    reviewRepo,
	}
}

// ================================
// Implementation methods
// ================================

// SearchCastings - 'db' добавлен
func (s *searchService) SearchCastings(db *gorm.DB, req *dto.SearchCastingsRequest) (*dto.PaginatedResponse, error) {
	if err := s.validateSearchRequest(req.Page, req.PageSize); err != nil {
		return nil, err
	}
	criteria := repositories.CastingSearchCriteria{
		Query:      req.Query,
		City:       req.City,
		Categories: req.Categories,
		Gender:     req.Gender,
		MinAge:     req.MinAge,
		MaxAge:     req.MaxAge,
		MinSalary:  req.MinSalary,
		MaxSalary:  req.MaxSalary,
		JobType:    req.JobType,
		Status:     req.Status,
		EmployerID: req.EmployerID,
		Page:       req.Page,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// ✅ Используем 'db' из параметра
	castings, total, err := s.castingRepo.SearchCastings(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildPaginatedResponse(castings, total, req.Page, req.PageSize), nil
}

// SearchCastingsAdvanced - 'db' добавлен
func (s *searchService) SearchCastingsAdvanced(db *gorm.DB, req *dto.AdvancedCastingSearchRequest) (*dto.PaginatedResponse, error) {
	basicReq := &dto.SearchCastingsRequest{
		Query:      req.Query,
		City:       req.City,
		Categories: req.Categories,
		MinSalary:  req.MinSalary,
		MaxSalary:  req.MaxSalary,
		Gender:     req.Gender,
		MinAge:     req.MinAge,
		MaxAge:     req.MaxAge,
		JobType:    req.JobType,
		Status:     req.Status,
		Page:       req.Page,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// ✅ Передаем 'db'
	response, err := s.SearchCastings(db, basicReq)
	if err != nil {
		return nil, err
	}
	// ... (доп. фильтры, если они есть, должны использовать 'db')
	return response, nil
}

// GetCastingSearchSuggestions - 'db' добавлен
func (s *searchService) GetCastingSearchSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error) {
	var suggestions []*dto.SearchSuggestion
	if query == "" {
		return suggestions, nil
	}

	// ✅ Передаем 'db'
	citySuggestions, err := s.getCitySuggestions(db, query, "casting", limit/3)
	if err == nil {
		suggestions = append(suggestions, citySuggestions...)
	}
	// ✅ Передаем 'db'
	categorySuggestions, err := s.getCategorySuggestions(db, query, "casting", limit/3)
	if err == nil {
		suggestions = append(suggestions, categorySuggestions...)
	}
	// ✅ Передаем 'db'
	titleSuggestions, err := s.getCastingTitleSuggestions(db, query, limit/3)
	if err == nil {
		suggestions = append(suggestions, titleSuggestions...)
	}

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions, nil
}

// SearchModels - 'db' добавлен
func (s *searchService) SearchModels(db *gorm.DB, req *dto.SearchModelsRequest) (*dto.PaginatedResponse, error) {
	if err := s.validateSearchRequest(req.Page, req.PageSize); err != nil {
		return nil, err
	}
	criteria := repositories.ModelSearchCriteria{
		Query:         req.Query,
		City:          req.City,
		Categories:    req.Categories,
		Gender:        req.Gender,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		MinHeight:     req.MinHeight,
		MaxHeight:     req.MaxHeight,
		MinWeight:     req.MinWeight,
		MaxWeight:     req.MaxWeight,
		MinPrice:      req.MinPrice,
		MaxPrice:      req.MaxPrice,
		MinExperience: req.MinExperience,
		Languages:     req.Languages,
		AcceptsBarter: req.AcceptsBarter,
		MinRating:     req.MinRating,
		IsPublic:      req.IsPublic,
		Page:          req.Page,
		PageSize:      req.PageSize,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
	}

	// ✅ Используем 'db' из параметра
	models, total, err := s.profileRepo.SearchModelProfiles(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildPaginatedResponse(models, total, req.Page, req.PageSize), nil
}

// SearchModelsAdvanced - 'db' добавлен
func (s *searchService) SearchModelsAdvanced(db *gorm.DB, req *dto.AdvancedModelSearchRequest) (*dto.PaginatedResponse, error) {
	basicReq := &dto.SearchModelsRequest{
		Query:         req.Query,
		City:          req.City,
		Categories:    req.Categories,
		Gender:        req.Gender,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		MinHeight:     req.MinHeight,
		MaxHeight:     req.MaxHeight,
		MinWeight:     req.MinWeight,
		MaxWeight:     req.MaxWeight,
		MinPrice:      req.MinPrice,
		MaxPrice:      req.MaxPrice,
		MinExperience: req.MinExperience,
		Languages:     req.Languages,
		AcceptsBarter: req.AcceptsBarter,
		MinRating:     req.MinRating,
		Page:          req.Page,
		PageSize:      req.PageSize,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
	}

	// ✅ Передаем 'db'
	response, err := s.SearchModels(db, basicReq)
	if err != nil {
		return nil, err
	}
	// ... (доп. фильтры, если они есть, должны использовать 'db')
	return response, nil
}

// GetModelSearchSuggestions - 'db' добавлен
func (s *searchService) GetModelSearchSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error) {
	var suggestions []*dto.SearchSuggestion
	if query == "" {
		return suggestions, nil
	}

	// ✅ Передаем 'db'
	citySuggestions, err := s.getCitySuggestions(db, query, "model", limit/4)
	if err == nil {
		suggestions = append(suggestions, citySuggestions...)
	}
	// ✅ Передаем 'db'
	categorySuggestions, err := s.getCategorySuggestions(db, query, "model", limit/4)
	if err == nil {
		suggestions = append(suggestions, categorySuggestions...)
	}
	// ✅ Передаем 'db'
	nameSuggestions, err := s.getModelNameSuggestions(db, query, limit/4)
	if err == nil {
		suggestions = append(suggestions, nameSuggestions...)
	}
	// ✅ Передаем 'db' (хотя заглушка и не использует)
	languageSuggestions := s.getLanguageSuggestions(db, query, limit/4)
	suggestions = append(suggestions, languageSuggestions...)

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions, nil
}

// SearchEmployers - 'db' добавлен
func (s *searchService) SearchEmployers(db *gorm.DB, req *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error) {
	if err := s.validateSearchRequest(req.Page, req.PageSize); err != nil {
		return nil, err
	}
	criteria := repositories.EmployerSearchCriteria{
		Query:       req.Query,
		City:        req.City,
		CompanyType: req.CompanyType,
		IsVerified:  req.IsVerified,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	// ✅ Используем 'db' из параметра
	employers, total, err := s.profileRepo.SearchEmployerProfiles(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildPaginatedResponse(employers, total, req.Page, req.PageSize), nil
}

// UnifiedSearch - 'db' добавлен
func (s *searchService) UnifiedSearch(db *gorm.DB, req *dto.UnifiedSearchRequest) (*dto.UnifiedSearchResponse, error) {
	response := &dto.UnifiedSearchResponse{}
	totalResults := 0

	switch req.Type {
	case "all", "":
		castingReq := &dto.SearchCastingsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		// ✅ Передаем 'db'
		if castings, err := s.SearchCastings(db, castingReq); err == nil {
			response.Castings = castings
			totalResults += int(castings.Total)
		}
		modelReq := &dto.SearchModelsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		// ✅ Передаем 'db'
		if models, err := s.SearchModels(db, modelReq); err == nil {
			response.Models = models
			totalResults += int(models.Total)
		}
		employerReq := &dto.SearchEmployersRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		// ✅ Передаем 'db'
		if employers, err := s.SearchEmployers(db, employerReq); err == nil {
			response.Employers = employers
			totalResults += int(employers.Total)
		}
	case "castings":
		castingReq := &dto.SearchCastingsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		// ✅ Передаем 'db'
		if castings, err := s.SearchCastings(db, castingReq); err == nil {
			response.Castings = castings
			totalResults = int(castings.Total)
		}
	case "models":
		modelReq := &dto.SearchModelsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		// ✅ Передаем 'db'
		if models, err := s.SearchModels(db, modelReq); err == nil {
			response.Models = models
			totalResults = int(models.Total)
		}
	case "employers":
		employerReq := &dto.SearchEmployersRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		// ✅ Передаем 'db'
		if employers, err := s.SearchEmployers(db, employerReq); err == nil {
			response.Employers = employers
			totalResults = int(employers.Total)
		}
	}

	response.TotalResults = totalResults
	return response, nil
}

// GetSearchAutoComplete - 'db' добавлен
func (s *searchService) GetSearchAutoComplete(db *gorm.DB, query string) (*dto.AutoCompleteResponse, error) {
	response := &dto.AutoCompleteResponse{}
	if len(query) < 2 {
		return response, nil
	}

	var allSuggestions []*dto.SearchSuggestion
	// ✅ Передаем 'db'
	castingSuggestions, _ := s.GetCastingSearchSuggestions(db, query, 5)
	allSuggestions = append(allSuggestions, castingSuggestions...)
	// ✅ Передаем 'db'
	modelSuggestions, _ := s.GetModelSearchSuggestions(db, query, 5)
	allSuggestions = append(allSuggestions, modelSuggestions...)

	uniqueSuggestions := s.removeDuplicateSuggestions(allSuggestions)
	if len(uniqueSuggestions) > 10 {
		uniqueSuggestions = uniqueSuggestions[:10]
	}
	response.Suggestions = uniqueSuggestions
	// ✅ Передаем 'db'
	response.Categories = s.getPopularCategoriesForQuery(db, query)
	// ✅ Передаем 'db'
	response.Cities = s.getPopularCitiesForQuery(db, query)

	return response, nil
}

// GetPopularSearches - 'db' добавлен
func (s *searchService) GetPopularSearches(db *gorm.DB, limit int) ([]*dto.PopularSearch, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Заменить на s.analyticsRepo.GetPopularSearches(db, limit)
	return []*dto.PopularSearch{
		{Query: "фотомодель", Type: "model", Count: 150, Trend: "up"},
		{Query: "реклама", Type: "casting", Count: 120, Trend: "stable"},
	}, nil
}

// GetSearchTrends - 'db' добавлен
func (s *searchService) GetSearchTrends(db *gorm.DB, days int) (*dto.SearchTrends, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Заменить на s.analyticsRepo.GetSearchTrends(db, days)
	return &dto.SearchTrends{
		TotalSearches:  1000,
		PopularQueries: []*dto.PopularSearch{{Query: "фотомодель", Type: "model", Count: 150, Trend: "up"}},
		TopCategories:  map[string]int64{"fashion": 200, "advertising": 150},
		TopCities:      map[string]int64{"Москва": 300, "Санкт-Петербург": 200},
		SearchByType:   map[string]int64{"castings": 400, "models": 350},
	}, nil
}

// SaveSearchHistory - 'db' добавлен
func (s *searchService) SaveSearchHistory(db *gorm.DB, userID, query, searchType string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Реализовать s.searchHistoryRepo.Create(tx, userID, query, searchType)

	return tx.Commit().Error
}

// GetSearchHistory - 'db' добавлен
func (s *searchService) GetSearchHistory(db *gorm.DB, userID string, limit int) ([]*dto.SearchHistoryItem, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Заменить на s.searchHistoryRepo.GetHistory(db, userID, limit)
	return []*dto.SearchHistoryItem{
		{ID: "1", Query: "фотомодель Москва", Type: "model", Results: 25, CreatedAt: "2024-01-15T10:30:00Z"},
	}, nil
}

// ClearSearchHistory - 'db' добавлен
func (s *searchService) ClearSearchHistory(db *gorm.DB, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Реализовать s.searchHistoryRepo.DeleteForUser(tx, userID)

	return tx.Commit().Error
}

// GetSearchAnalytics - 'db' добавлен
func (s *searchService) GetSearchAnalytics(db *gorm.DB, days int) (*dto.SearchAnalytics, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Заменить на s.analyticsRepo.GetSearchAnalytics(db, days)
	return &dto.SearchAnalytics{
		TotalSearches:      5000,
		SuccessfulSearches: 4500,
		AverageResults:     23.5,
		NoResultRate:       0.1,
		PopularFilters:     map[string]int64{"city": 1200, "category": 900},
		SearchPerformance:  &dto.SearchPerformance{AverageResponseTime: 0.15, CacheHitRate: 0.65, ErrorRate: 0.02},
	}, nil
}

// ReindexSearchData - 'db' добавлен
func (s *searchService) ReindexSearchData(db *gorm.DB, adminID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Проверить права adminID (s.userRepo.FindByID(tx, adminID))
	// TODO: Реализовать логику реиндексации (s.searchRepo.ReindexAll(tx))

	return tx.Commit().Error
}

// ================================
// Helper methods
// ================================

// (Чистая функция - без изменений)
func (s *searchService) validateSearchRequest(page, pageSize int) error {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return nil
}

// (Чистая функция - без изменений)
func (s *searchService) buildPaginatedResponse(data interface{}, total int64, page, pageSize int) *dto.PaginatedResponse {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	hasMore := page < totalPages

	return &dto.PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    hasMore,
	}
}

// ✅ Хелперы-заглушки обновлены, чтобы принимать 'db'
func (s *searchService) getCitySuggestions(db *gorm.DB, query, searchType string, limit int) ([]*dto.SearchSuggestion, error) {
	// TODO: Заменить на реальный вызов (e.g. s.analyticsRepo.GetCitySuggestions(db, ...))
	cities := []string{"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань"}
	var suggestions []*dto.SearchSuggestion
	for _, city := range cities {
		if strings.Contains(strings.ToLower(city), strings.ToLower(query)) {
			suggestions = append(suggestions, &dto.SearchSuggestion{Type: "city", Value: city, Count: 100})
		}
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (s *searchService) getCategorySuggestions(db *gorm.DB, query, searchType string, limit int) ([]*dto.SearchSuggestion, error) {
	// TODO: Заменить на реальный вызов
	categories := []string{"fashion", "advertising", "video", "photography", "modeling"}
	var suggestions []*dto.SearchSuggestion
	for _, cat := range categories {
		if strings.Contains(strings.ToLower(cat), strings.ToLower(query)) {
			suggestions = append(suggestions, &dto.SearchSuggestion{Type: "category", Value: cat, Count: 50})
		}
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (s *searchService) getCastingTitleSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error) {
	// TODO: Заменить на s.castingRepo.GetTitleSuggestions(db, query, limit)
	titles := []string{"Реклама одежды", "Фотомодель для рекламы", "Съемка видео", "Кастинг актеров"}
	var suggestions []*dto.SearchSuggestion
	for _, t := range titles {
		if strings.Contains(strings.ToLower(t), strings.ToLower(query)) {
			suggestions = append(suggestions, &dto.SearchSuggestion{Type: "casting", Value: t, Count: 30})
		}
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (s *searchService) getModelNameSuggestions(db *gorm.DB, query string, limit int) ([]*dto.SearchSuggestion, error) {
	// TODO: Заменить на s.profileRepo.GetNameSuggestions(db, query, limit)
	names := []string{"Анна Иванова", "Екатерина Петрова", "Мария Смирнова"}
	var suggestions []*dto.SearchSuggestion
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			suggestions = append(suggestions, &dto.SearchSuggestion{Type: "model", Value: name, Count: 20})
		}
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (s *searchService) getLanguageSuggestions(db *gorm.DB, query string, limit int) []*dto.SearchSuggestion {
	// (Чистая функция, т.к. список языков статичен)
	languages := []string{"English", "Русский", "Kazakh"}
	var suggestions []*dto.SearchSuggestion
	for _, lang := range languages {
		if strings.Contains(strings.ToLower(lang), strings.ToLower(query)) {
			suggestions = append(suggestions, &dto.SearchSuggestion{Type: "language", Value: lang, Count: 10})
		}
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions
}

// (Чистая функция - без изменений)
func (s *searchService) removeDuplicateSuggestions(suggestions []*dto.SearchSuggestion) []*dto.SearchSuggestion {
	seen := make(map[string]bool)
	var result []*dto.SearchSuggestion
	for _, s := range suggestions {
		key := s.Type + ":" + s.Value
		if !seen[key] {
			result = append(result, s)
			seen[key] = true
		}
	}
	return result
}

func (s *searchService) getPopularCategoriesForQuery(db *gorm.DB, query string) []string {
	// TODO: Заменить на реальный вызов (s.analyticsRepo.GetPopularCategories(db, query))
	return []string{"fashion", "advertising"}
}

func (s *searchService) getPopularCitiesForQuery(db *gorm.DB, query string) []string {
	// TODO: Заменить на реальный вызов (s.analyticsRepo.GetPopularCities(db, query))
	return []string{"Москва", "Санкт-Петербург"}
}

// (Вспомогательный хелпер для ошибок - без изменений)
func handleSearchError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrCastingNotFound) ||
		errors.Is(err, repositories.ErrProfileNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}
