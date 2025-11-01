package dto

import (
	"encoding/json"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/datatypes"
)

// ========================
// Matching DTOs
// ========================

// MatchResult
type MatchResult struct {
	ModelID       string                  `json:"model_id"`
	ModelName     string                  `json:"model_name"`
	City          string                  `json:"city,omitempty"`
	Score         float64                 `json:"score"`
	Reasons       []string                `json:"reasons"`
	Compatibility *CompatibilityBreakdown `json:"compatibility,omitempty"`
}

// MatchScore
type MatchScore struct {
	TotalScore     float64                 `json:"total_score"`
	CategoryScores map[string]float64      `json:"category_scores"`
	Breakdown      *CompatibilityBreakdown `json:"breakdown"`
}

// CompatibilityResult
type CompatibilityResult struct {
	ModelID         string                  `json:"model_id"`
	CastingID       string                  `json:"casting_id"`
	TotalScore      float64                 `json:"total_score"`
	Breakdown       *CompatibilityBreakdown `json:"breakdown"`
	Recommendations []string                `json:"recommendations"`
}

// CompatibilityBreakdown
type CompatibilityBreakdown struct {
	Demographics float64 `json:"demographics"`
	Physical     float64 `json:"physical"`
	Professional float64 `json:"professional"`
	Geographic   float64 `json:"geographic"`
	Specialized  float64 `json:"specialized"`
}

// MatchCriteria
type MatchCriteria struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
	Gender     string   `json:"gender" validate:"omitempty,is-gender"` // Custom rule
	MinAge     *int     `json:"min_age" validate:"omitempty,min=0"`
	MaxAge     *int     `json:"max_age" validate:"omitempty,min=0,gtefield=MinAge"`
	MinHeight  *int     `json:"min_height" validate:"omitempty,min=0"`
	MaxHeight  *int     `json:"max_height" validate:"omitempty,min=0,gtefield=MinHeight"`
	MinWeight  *int     `json:"min_weight" validate:"omitempty,min=0"`
	MaxWeight  *int     `json:"max_weight" validate:"omitempty,min=0,gtefield=MinWeight"`
	MinRating  *float64 `json:"min_rating" validate:"omitempty,min=0,max=5"`
	JobType    string   `json:"job_type" validate:"omitempty,is-job-type"` // Custom rule
	Languages  []string `json:"languages"`
	Limit      int      `json:"limit" validate:"omitempty,min=0,max=100"`     // Allow 0 for default
	MinScore   float64  `json:"min_score" validate:"omitempty,min=0,max=100"` // Allow 0 for default
}

// SimilarModel
type SimilarModel struct {
	ModelID          string   `json:"model_id"`
	Name             string   `json:"name"`
	City             string   `json:"city"`
	Similarity       float64  `json:"similarity"`
	CommonCategories []string `json:"common_categories"`
}

// MatchingWeights
type MatchingWeights struct {
	// Added 'required' as these must be set
	Demographics float64 `json:"demographics" validate:"required,min=0,max=1"`
	Physical     float64 `json:"physical" validate:"required,min=0,max=1"`
	Professional float64 `json:"professional" validate:"required,min=0,max=1"`
	Geographic   float64 `json:"geographic" validate:"required,min=0,max=1"`
	Specialized  float64 `json:"specialized" validate:"required,min=0,max=1"`
}

// MatchingStats
type MatchingStats struct {
	CastingID         string         `json:"casting_id"`
	TotalModels       int64          `json:"total_models"`
	MatchedModels     int64          `json:"matched_models"`
	AverageScore      float64        `json:"average_score"`
	ScoreDistribution map[string]int `json:"score_distribution"`
	TopCategories     []string       `json:"top_categories"`
}

// ModelMatchingStats
type ModelMatchingStats struct {
	ModelID         string   `json:"model_id"`
	TotalCastings   int64    `json:"total_castings"`
	MatchedCastings int64    `json:"matched_castings"`
	MatchRate       float64  `json:"match_rate"`
	AverageScore    float64  `json:"average_score"`
	ResponseRate    float64  `json:"response_rate"`
	TopMatchReasons []string `json:"top_match_reasons"`
}

// PlatformMatchingStats
type PlatformMatchingStats struct {
	TotalMatches      int64            `json:"total_matches"`
	SuccessfulMatches int64            `json:"successful_matches"`
	AverageMatchScore float64          `json:"average_match_score"`
	MatchRate         float64          `json:"match_rate"`
	ByCategory        map[string]int64 `json:"by_category"`
	ByCity            map[string]int64 `json:"by_city"`
}

// MatchingLog
type MatchingLog struct {
	ID        string    `json:"id"`
	CastingID string    `json:"casting_id"`
	ModelID   string    `json:"model_id"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

// MatchingLogCriteria
type MatchingLogCriteria struct {
	CastingID string    `form:"casting_id"`
	ModelID   string    `form:"model_id"`
	MinScore  float64   `form:"min_score" validate:"omitempty,min=0,max=100"`
	DateFrom  time.Time `form:"date_from" validate:"omitempty,ltfield=DateTo"`
	DateTo    time.Time `form:"date_to" validate:"omitempty,gtfield=DateFrom"`
	Page      int       `form:"page" validate:"omitempty,min=1"`
	PageSize  int       `form:"page_size" validate:"omitempty,min=1,max=100"`
}

// ========================
// Helper functions
// ========================

func ParseCategories(categoriesData datatypes.JSON) []string {
	var categories []string
	if len(categoriesData) > 0 {
		json.Unmarshal(categoriesData, &categories)
	}
	return categories
}

func FormatCategories(categories []string) datatypes.JSON {
	if len(categories) == 0 {
		return datatypes.JSON("[]")
	}
	jsonData, _ := json.Marshal(categories)
	return datatypes.JSON(jsonData)
}

type MatchingCasting struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
	Gender     string   `json:"gender"`
	AgeMin     *int     `json:"age_min,omitempty"`
	AgeMax     *int     `json:"age_max,omitempty"`
	HeightMin  *float64 `json:"height_min,omitempty"`
	HeightMax  *float64 `json:"height_max,omitempty"`
	WeightMin  *float64 `json:"weight_min,omitempty"`
	WeightMax  *float64 `json:"weight_max,omitempty"`
	JobType    string   `json:"job_type"`
	Languages  []string `json:"languages,omitempty"`
}

func CastingToMatchingDTO(casting *models.Casting) *MatchingCasting {
	return &MatchingCasting{
		City:       casting.City,
		Categories: casting.GetCategories(),
		Gender:     casting.Gender,
		AgeMin:     casting.AgeMin,
		AgeMax:     casting.AgeMax,
		HeightMin:  casting.HeightMin,
		HeightMax:  casting.HeightMax,
		WeightMin:  casting.WeightMin,
		WeightMax:  casting.WeightMax,
		JobType:    casting.JobType,
		Languages:  casting.GetLanguages(),
	}
}
