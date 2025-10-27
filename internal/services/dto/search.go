package dto

// ====================
//  Request DTOs
// ====================

// --- Castings ---
type SearchCastingsRequest struct {
	Query      string   `form:"query"`
	City       string   `form:"city"`
	Categories []string `form:"categories"`
	MinSalary  *int     `form:"min_salary"`
	MaxSalary  *int     `form:"max_salary"`
	Gender     string   `form:"gender"`
	MinAge     *int     `form:"min_age"`
	MaxAge     *int     `form:"max_age"`
	JobType    string   `form:"job_type"`
	Status     string   `form:"status"`
	EmployerID string   `form:"employer_id"`
	Page       int      `form:"page" binding:"min=1"`
	PageSize   int      `form:"page_size" binding:"min=1,max=100"`
	SortBy     string   `form:"sort_by"`
	SortOrder  string   `form:"sort_order"`
}

type AdvancedCastingSearchRequest struct {
	SearchCastingsRequest
	DateFrom   *string `form:"date_from"`
	DateTo     *string `form:"date_to"`
	Experience string  `form:"experience"`
	WithPhotos *bool   `form:"with_photos"`
	UrgentOnly *bool   `form:"urgent_only"`
}

// --- Models ---
type SearchModelsRequest struct {
	Query         string   `form:"query"`
	City          string   `form:"city"`
	Categories    []string `form:"categories"`
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
	Languages     []string `form:"languages"`
	AcceptsBarter *bool    `form:"accepts_barter"`
	MinRating     *float64 `form:"min_rating"`
	IsPublic      *bool    `form:"is_public"`
	Page          int      `form:"page" binding:"min=1"`
	PageSize      int      `form:"page_size" binding:"min=1,max=100"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order"`
}

type AdvancedModelSearchRequest struct {
	SearchModelsRequest
	ClothingSize string `form:"clothing_size"`
	ShoeSize     string `form:"shoe_size"`
	HasPortfolio *bool  `form:"has_portfolio"`
	HasReviews   *bool  `form:"has_reviews"`
	Availability string `form:"availability"`
}

// --- Employers ---
type SearchEmployersRequest struct {
	Query       string `form:"query"`
	City        string `form:"city"`
	CompanyType string `form:"company_type"`
	IsVerified  *bool  `form:"is_verified"`
	Page        int    `form:"page" binding:"min=1"`
	PageSize    int    `form:"page_size" binding:"min=1,max=100"`
}

// --- Unified search ---
type UnifiedSearchRequest struct {
	Query    string `form:"query" binding:"required"`
	Type     string `form:"type"`
	City     string `form:"city"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=50"`
}

// ====================
//  Response DTOs
// ====================

// --- Pagination ---
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

// --- Unified search ---
type UnifiedSearchResponse struct {
	Castings     *PaginatedResponse `json:"castings,omitempty"`
	Models       *PaginatedResponse `json:"models,omitempty"`
	Employers    *PaginatedResponse `json:"employers,omitempty"`
	TotalResults int                `json:"total_results"`
}

// --- Suggestions / Autocomplete ---
type SearchSuggestion struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Count int64  `json:"count,omitempty"`
}

type AutoCompleteResponse struct {
	Suggestions []*SearchSuggestion `json:"suggestions"`
	Categories  []string            `json:"categories,omitempty"`
	Cities      []string            `json:"cities,omitempty"`
}

// --- Popular searches and trends ---
type PopularSearch struct {
	Query string `json:"query"`
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Trend string `json:"trend"`
}

type SearchTrends struct {
	TotalSearches  int64            `json:"total_searches"`
	PopularQueries []*PopularSearch `json:"popular_queries"`
	TopCategories  map[string]int64 `json:"top_categories"`
	TopCities      map[string]int64 `json:"top_cities"`
	SearchByType   map[string]int64 `json:"search_by_type"`
}

// --- Search history ---
type SearchHistoryItem struct {
	ID        string `json:"id"`
	Query     string `json:"query"`
	Type      string `json:"type"`
	Results   int64  `json:"results"`
	CreatedAt string `json:"created_at"`
}

// --- Search analytics ---
type SearchAnalytics struct {
	TotalSearches      int64              `json:"total_searches"`
	SuccessfulSearches int64              `json:"successful_searches"`
	AverageResults     float64            `json:"average_results"`
	NoResultRate       float64            `json:"no_result_rate"`
	PopularFilters     map[string]int64   `json:"popular_filters"`
	SearchPerformance  *SearchPerformance `json:"performance"`
}

type SearchPerformance struct {
	AverageResponseTime float64 `json:"average_response_time"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	ErrorRate           float64 `json:"error_rate"`
}
