package models

type SearchCastingsRequest struct {
	Query      string   `form:"query"`
	City       string   `form:"city"`
	Categories []string `form:"categories"`
	MinSalary  *int     `form:"min_salary"`
	MaxSalary  *int     `form:"max_salary"`
	Gender     string   `form:"gender"`
	MinAge     *int     `form:"min_age"`
	MaxAge     *int     `form:"max_age"`
	Page       int      `form:"page" binding:"min=1"`
	PageSize   int      `form:"page_size" binding:"min=1,max=100"`
}

type SearchModelsRequest struct {
	Query         string   `form:"query"`
	City          string   `form:"city"`
	Categories    []string `form:"categories"`
	Gender        string   `form:"gender"`
	MinAge        *int     `form:"min_age"`
	MaxAge        *int     `form:"max_age"`
	MinPrice      *int     `form:"min_price"`
	MaxPrice      *int     `form:"max_price"`
	MinExperience *int     `form:"min_experience"`
	Languages     []string `form:"languages"`
	AcceptsBarter *bool    `form:"accepts_barter"`
	MinRating     *float64 `form:"min_rating"`
	Page          int      `form:"page" binding:"min=1"`
	PageSize      int      `form:"page_size" binding:"min=1,max=100"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}
