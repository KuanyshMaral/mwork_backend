package models

type MatchResult struct {
	ModelID string   `json:"model_id"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`
}

type MatchRequest struct {
	CastingID string  `json:"casting_id" binding:"required"`
	Limit     int     `json:"limit" binding:"min=1,max=100"`
	MinScore  float64 `json:"min_score" binding:"min=0,max=100"`
}
