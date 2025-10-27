package services

import (
	"errors"
	"strings"

	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

type SearchService interface {
	// Casting search operations
	SearchCastings(req *dto.SearchCastingsRequest) (*dto.PaginatedResponse, error)
	SearchCastingsAdvanced(req *dto.AdvancedCastingSearchRequest) (*dto.PaginatedResponse, error)
	GetCastingSearchSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error)

	// Model search operations
	SearchModels(req *dto.SearchModelsRequest) (*dto.PaginatedResponse, error)
	SearchModelsAdvanced(req *dto.AdvancedModelSearchRequest) (*dto.PaginatedResponse, error)
	GetModelSearchSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error)

	// Employer search operations
	SearchEmployers(req *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error)

	// Unified search
	UnifiedSearch(req *dto.UnifiedSearchRequest) (*dto.UnifiedSearchResponse, error)
	GetSearchAutoComplete(query string) (*dto.AutoCompleteResponse, error)

	// Search analytics and features
	GetPopularSearches(limit int) ([]*dto.PopularSearch, error)
	GetSearchTrends(days int) (*dto.SearchTrends, error)
	SaveSearchHistory(userID, query, searchType string) error
	GetSearchHistory(userID string, limit int) ([]*dto.SearchHistoryItem, error)
	ClearSearchHistory(userID string) error

	// Admin search operations
	GetSearchAnalytics(days int) (*dto.SearchAnalytics, error)
	ReindexSearchData(adminID string) error
}

type searchService struct {
	castingRepo   repositories.CastingRepository
	profileRepo   repositories.ProfileRepository
	portfolioRepo repositories.PortfolioRepository
	reviewRepo    repositories.ReviewRepository
}

func NewSearchService(
	castingRepo repositories.CastingRepository,
	profileRepo repositories.ProfileRepository,
	portfolioRepo repositories.PortfolioRepository,
	reviewRepo repositories.ReviewRepository,
) SearchService {
	return &searchService{
		castingRepo:   castingRepo,
		profileRepo:   profileRepo,
		portfolioRepo: portfolioRepo,
		reviewRepo:    reviewRepo,
	}
}

// ================================
// Implementation methods
// ================================

func (s *searchService) SearchCastings(req *dto.SearchCastingsRequest) (*dto.PaginatedResponse, error) {
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

	castings, total, err := s.castingRepo.SearchCastings(criteria)
	if err != nil {
		return nil, err
	}

	return s.buildPaginatedResponse(castings, total, req.Page, req.PageSize), nil
}

func (s *searchService) SearchCastingsAdvanced(req *dto.AdvancedCastingSearchRequest) (*dto.PaginatedResponse, error) {
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

	response, err := s.SearchCastings(basicReq)
	if err != nil {
		return nil, err
	}

	// Additional filters (urgent, with photos) can be applied here

	return response, nil
}

func (s *searchService) GetCastingSearchSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error) {
	var suggestions []*dto.SearchSuggestion

	if query == "" {
		return suggestions, nil
	}

	citySuggestions, err := s.getCitySuggestions(query, "casting", limit/3)
	if err == nil {
		suggestions = append(suggestions, citySuggestions...)
	}

	categorySuggestions, err := s.getCategorySuggestions(query, "casting", limit/3)
	if err == nil {
		suggestions = append(suggestions, categorySuggestions...)
	}

	titleSuggestions, err := s.getCastingTitleSuggestions(query, limit/3)
	if err == nil {
		suggestions = append(suggestions, titleSuggestions...)
	}

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

func (s *searchService) SearchModels(req *dto.SearchModelsRequest) (*dto.PaginatedResponse, error) {
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

	models, total, err := s.profileRepo.SearchModelProfiles(criteria)
	if err != nil {
		return nil, err
	}

	return s.buildPaginatedResponse(models, total, req.Page, req.PageSize), nil
}

func (s *searchService) SearchModelsAdvanced(req *dto.AdvancedModelSearchRequest) (*dto.PaginatedResponse, error) {
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

	response, err := s.SearchModels(basicReq)
	if err != nil {
		return nil, err
	}

	// Additional filters (portfolio, clothing size, availability) can be applied here

	return response, nil
}

func (s *searchService) GetModelSearchSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error) {
	var suggestions []*dto.SearchSuggestion

	if query == "" {
		return suggestions, nil
	}

	citySuggestions, err := s.getCitySuggestions(query, "model", limit/4)
	if err == nil {
		suggestions = append(suggestions, citySuggestions...)
	}

	categorySuggestions, err := s.getCategorySuggestions(query, "model", limit/4)
	if err == nil {
		suggestions = append(suggestions, categorySuggestions...)
	}

	nameSuggestions, err := s.getModelNameSuggestions(query, limit/4)
	if err == nil {
		suggestions = append(suggestions, nameSuggestions...)
	}

	languageSuggestions := s.getLanguageSuggestions(query, limit/4)
	suggestions = append(suggestions, languageSuggestions...)

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

func (s *searchService) SearchEmployers(req *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error) {
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

	employers, total, err := s.profileRepo.SearchEmployerProfiles(criteria)
	if err != nil {
		return nil, err
	}

	return s.buildPaginatedResponse(employers, total, req.Page, req.PageSize), nil
}

func (s *searchService) UnifiedSearch(req *dto.UnifiedSearchRequest) (*dto.UnifiedSearchResponse, error) {
	response := &dto.UnifiedSearchResponse{}
	totalResults := 0

	switch req.Type {
	case "all", "":
		castingReq := &dto.SearchCastingsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		if castings, err := s.SearchCastings(castingReq); err == nil {
			response.Castings = castings
			totalResults += int(castings.Total)
		}

		modelReq := &dto.SearchModelsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		if models, err := s.SearchModels(modelReq); err == nil {
			response.Models = models
			totalResults += int(models.Total)
		}

		employerReq := &dto.SearchEmployersRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize / 3}
		if employers, err := s.SearchEmployers(employerReq); err == nil {
			response.Employers = employers
			totalResults += int(employers.Total)
		}
	case "castings":
		castingReq := &dto.SearchCastingsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		if castings, err := s.SearchCastings(castingReq); err == nil {
			response.Castings = castings
			totalResults = int(castings.Total)
		}
	case "models":
		modelReq := &dto.SearchModelsRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		if models, err := s.SearchModels(modelReq); err == nil {
			response.Models = models
			totalResults = int(models.Total)
		}
	case "employers":
		employerReq := &dto.SearchEmployersRequest{Query: req.Query, City: req.City, Page: req.Page, PageSize: req.PageSize}
		if employers, err := s.SearchEmployers(employerReq); err == nil {
			response.Employers = employers
			totalResults = int(employers.Total)
		}
	}

	response.TotalResults = totalResults
	return response, nil
}

func (s *searchService) GetSearchAutoComplete(query string) (*dto.AutoCompleteResponse, error) {
	response := &dto.AutoCompleteResponse{}

	if len(query) < 2 {
		return response, nil
	}

	var allSuggestions []*dto.SearchSuggestion
	castingSuggestions, _ := s.GetCastingSearchSuggestions(query, 5)
	allSuggestions = append(allSuggestions, castingSuggestions...)

	modelSuggestions, _ := s.GetModelSearchSuggestions(query, 5)
	allSuggestions = append(allSuggestions, modelSuggestions...)

	uniqueSuggestions := s.removeDuplicateSuggestions(allSuggestions)
	if len(uniqueSuggestions) > 10 {
		uniqueSuggestions = uniqueSuggestions[:10]
	}

	response.Suggestions = uniqueSuggestions
	response.Categories = s.getPopularCategoriesForQuery(query)
	response.Cities = s.getPopularCitiesForQuery(query)

	return response, nil
}

func (s *searchService) GetPopularSearches(limit int) ([]*dto.PopularSearch, error) {
	return []*dto.PopularSearch{
		{Query: "фотомодель", Type: "model", Count: 150, Trend: "up"},
		{Query: "реклама", Type: "casting", Count: 120, Trend: "stable"},
		{Query: "Москва", Type: "city", Count: 100, Trend: "up"},
		{Query: "fashion", Type: "category", Count: 80, Trend: "down"},
		{Query: "видеосъемка", Type: "casting", Count: 75, Trend: "up"},
	}, nil
}

func (s *searchService) GetSearchTrends(days int) (*dto.SearchTrends, error) {
	return &dto.SearchTrends{
		TotalSearches: 1000,
		PopularQueries: []*dto.PopularSearch{
			{Query: "фотомодель", Type: "model", Count: 150, Trend: "up"},
			{Query: "реклама", Type: "casting", Count: 120, Trend: "stable"},
		},
		TopCategories: map[string]int64{"fashion": 200, "advertising": 150, "video": 100},
		TopCities:     map[string]int64{"Москва": 300, "Санкт-Петербург": 200, "Новосибирск": 50},
		SearchByType:  map[string]int64{"castings": 400, "models": 350, "employers": 250},
	}, nil
}

func (s *searchService) SaveSearchHistory(userID, query, searchType string) error {
	return nil
}

func (s *searchService) GetSearchHistory(userID string, limit int) ([]*dto.SearchHistoryItem, error) {
	return []*dto.SearchHistoryItem{
		{ID: "1", Query: "фотомодель Москва", Type: "model", Results: 25, CreatedAt: "2024-01-15T10:30:00Z"},
		{ID: "2", Query: "реклама видео", Type: "casting", Results: 18, CreatedAt: "2024-01-14T15:45:00Z"},
	}, nil
}

func (s *searchService) ClearSearchHistory(userID string) error {
	return nil
}

func (s *searchService) GetSearchAnalytics(days int) (*dto.SearchAnalytics, error) {
	return &dto.SearchAnalytics{
		TotalSearches:      5000,
		SuccessfulSearches: 4500,
		AverageResults:     23.5,
		NoResultRate:       0.1,
		PopularFilters:     map[string]int64{"city": 1200, "category": 900, "salary": 600, "experience": 400},
		SearchPerformance:  &dto.SearchPerformance{AverageResponseTime: 0.15, CacheHitRate: 0.65, ErrorRate: 0.02},
	}, nil
}

func (s *searchService) ReindexSearchData(adminID string) error {
	return nil
}

// ================================
// Helper methods
// ================================

func (s *searchService) validateSearchRequest(page, pageSize int) error {
	if page < 1 {
		return errors.New("page must be at least 1")
	}
	if pageSize < 1 || pageSize > 100 {
		return errors.New("page size must be between 1 and 100")
	}
	return nil
}

func (s *searchService) buildPaginatedResponse(data interface{}, total int64, page, pageSize int) *dto.PaginatedResponse {
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

func (s *searchService) getCitySuggestions(query, searchType string, limit int) ([]*dto.SearchSuggestion, error) {
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

func (s *searchService) getCategorySuggestions(query, searchType string, limit int) ([]*dto.SearchSuggestion, error) {
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

func (s *searchService) getCastingTitleSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error) {
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

func (s *searchService) getModelNameSuggestions(query string, limit int) ([]*dto.SearchSuggestion, error) {
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

func (s *searchService) getLanguageSuggestions(query string, limit int) []*dto.SearchSuggestion {
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

func (s *searchService) getPopularCategoriesForQuery(query string) []string {
	return []string{"fashion", "advertising"}
}

func (s *searchService) getPopularCitiesForQuery(query string) []string {
	return []string{"Москва", "Санкт-Петербург"}
}
