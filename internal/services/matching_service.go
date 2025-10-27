package services

import (
	"errors"
	"math"
	"sort"

	"gorm.io/datatypes"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

var (
	ErrCastingNotFound = errors.New("casting not found")
)

type MatchingService interface {
	// Core matching operations
	FindMatchingModels(castingID string, limit int, minScore float64) ([]*dto.MatchResult, error)
	FindModelsForCasting(casting *models.Casting, limit int) ([]*dto.MatchResult, error)
	CalculateMatchScore(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error)

	// Advanced matching
	FindModelsByCriteria(criteria *dto.MatchCriteria) ([]*dto.MatchResult, error)
	GetModelCompatibility(modelID, castingID string) (*dto.CompatibilityResult, error)
	FindSimilarModels(modelID string, limit int) ([]*dto.SimilarModel, error)

	// Batch operations
	BatchMatchModels(castingIDs []string) (map[string][]*dto.MatchResult, error)
	UpdateModelRecommendations(modelID string) error

	// Matching configuration
	GetMatchingWeights() (*dto.MatchingWeights, error)
	UpdateMatchingWeights(adminID string, weights *dto.MatchingWeights) error

	// Analytics and insights
	GetMatchingStats(castingID string) (*dto.MatchingStats, error)
	GetModelMatchingStats(modelID string) (*dto.ModelMatchingStats, error)
	GetPlatformMatchingStats() (*dto.PlatformMatchingStats, error)

	// Admin operations
	RecalculateAllMatches(adminID string) error
	GetMatchingLogs(criteria dto.MatchingLogCriteria) ([]*dto.MatchingLog, int64, error)
}

type matchingService struct {
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	reviewRepo       repositories.ReviewRepository
	portfolioRepo    repositories.PortfolioRepository
	notificationRepo repositories.NotificationRepository
}

// Default matching weights
var defaultWeights = &dto.MatchingWeights{
	Demographics: 0.2,
	Physical:     0.25,
	Professional: 0.2,
	Geographic:   0.15,
	Specialized:  0.2,
}

func NewMatchingService(
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	reviewRepo repositories.ReviewRepository,
	portfolioRepo repositories.PortfolioRepository,
	notificationRepo repositories.NotificationRepository,
) MatchingService {
	return &matchingService{
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		reviewRepo:       reviewRepo,
		portfolioRepo:    portfolioRepo,
		notificationRepo: notificationRepo,
	}
}

// -------------------------------
// Core matching operations
// -------------------------------

func (s *matchingService) FindMatchingModels(castingID string, limit int, minScore float64) ([]*dto.MatchResult, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, ErrCastingNotFound
	}
	return s.FindModelsForCasting(casting, limit)
}

func float64PtrToIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	i := int(*f)
	return &i
}

func (s *matchingService) FindModelsForCasting(casting *models.Casting, limit int) ([]*dto.MatchResult, error) {
	criteria := &dto.MatchCriteria{
		City:       casting.City,
		Categories: s.parseCategories(casting.Categories),
		Gender:     casting.Gender,
		MinAge:     casting.AgeMin,
		MaxAge:     casting.AgeMax,
		MinHeight:  float64PtrToIntPtr(casting.HeightMin),
		MaxHeight:  float64PtrToIntPtr(casting.HeightMax),
		JobType:    casting.JobType,
		Limit:      limit,
		MinScore:   50.0,
	}

	models, err := s.FindModelsByCriteria(criteria)
	if err != nil {
		return nil, err
	}

	if len(models) > 0 {
		go s.notifyTopMatches(casting, models)
	}

	return models, nil
}

func (s *matchingService) CalculateMatchScore(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error) {
	breakdown := &dto.CompatibilityBreakdown{}
	categoryScores := make(map[string]float64)

	demographicsScore := s.calculateDemographicsScore(model, casting)
	breakdown.Demographics = demographicsScore
	categoryScores["demographics"] = demographicsScore

	physicalScore := s.calculatePhysicalScore(model, casting)
	breakdown.Physical = physicalScore
	categoryScores["physical"] = physicalScore

	professionalScore := s.calculateProfessionalScore(model, casting)
	breakdown.Professional = professionalScore
	categoryScores["professional"] = professionalScore

	geographicScore := s.calculateGeographicScore(model, casting)
	breakdown.Geographic = geographicScore
	categoryScores["geographic"] = geographicScore

	specializedScore := s.calculateSpecializedScore(model, casting)
	breakdown.Specialized = specializedScore
	categoryScores["specialized"] = specializedScore

	totalScore := (demographicsScore * defaultWeights.Demographics) +
		(physicalScore * defaultWeights.Physical) +
		(professionalScore * defaultWeights.Professional) +
		(geographicScore * defaultWeights.Geographic) +
		(specializedScore * defaultWeights.Specialized)

	return &dto.MatchScore{
		TotalScore:     math.Round(totalScore*100) / 100,
		CategoryScores: categoryScores,
		Breakdown:      breakdown,
	}, nil
}

// -------------------------------
// Advanced matching
// -------------------------------

func (s *matchingService) FindModelsByCriteria(criteria *dto.MatchCriteria) ([]*dto.MatchResult, error) {
	searchCriteria := repositories.ModelSearchCriteria{
		City:       criteria.City,
		Categories: criteria.Categories,
		Gender:     criteria.Gender,
		MinAge:     criteria.MinAge,
		MaxAge:     criteria.MaxAge,
		MinHeight:  criteria.MinHeight,
		MaxHeight:  criteria.MaxHeight,
		MinWeight:  criteria.MinWeight,
		MaxWeight:  criteria.MaxWeight,
		MinRating:  criteria.MinRating,
		Languages:  criteria.Languages,
		Page:       1,
		PageSize:   criteria.Limit,
		IsPublic:   &[]bool{true}[0],
	}

	models, _, err := s.profileRepo.SearchModelProfiles(searchCriteria)
	if err != nil {
		return nil, err
	}

	var matchResults []*dto.MatchResult
	for _, model := range models {
		mockCasting := &models.Casting{
			City:       criteria.City,
			Categories: s.formatCategories(criteria.Categories),
			Gender:     criteria.Gender,
			AgeMin:     criteria.MinAge,
			AgeMax:     criteria.MaxAge,
			HeightMin:  intPtrToFloat64Ptr(criteria.MinHeight),
			HeightMax:  intPtrToFloat64Ptr(criteria.MaxHeight),
			JobType:    criteria.JobType,
		}

		score, err := s.CalculateMatchScore(&model, mockCasting)
		if err != nil {
			continue
		}

		if score.TotalScore >= criteria.MinScore {
			matchResults = append(matchResults, &dto.MatchResult{
				ModelID:       model.ID,
				ModelName:     model.Name,
				Score:         score.TotalScore,
				Reasons:       s.generateMatchReasons(score, &model, mockCasting),
				Compatibility: score.Breakdown,
			})
		}
	}

	sort.Slice(matchResults, func(i, j int) bool {
		return matchResults[i].Score > matchResults[j].Score
	})

	if len(matchResults) > criteria.Limit {
		matchResults = matchResults[:criteria.Limit]
	}

	return matchResults, nil
}

func (s *matchingService) GetModelCompatibility(modelID, castingID string) (*dto.CompatibilityResult, error) {
	model, err := s.profileRepo.FindModelProfileByID(modelID)
	if err != nil {
		return nil, errors.New("model not found")
	}

	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, ErrCastingNotFound
	}

	score, err := s.CalculateMatchScore(model, casting)
	if err != nil {
		return nil, err
	}

	return &dto.CompatibilityResult{
		ModelID:         modelID,
		CastingID:       castingID,
		TotalScore:      score.TotalScore,
		Breakdown:       score.Breakdown,
		Recommendations: s.generateRecommendations(score, model, casting),
	}, nil
}

func (s *matchingService) FindSimilarModels(modelID string, limit int) ([]*dto.SimilarModel, error) {
	targetModel, err := s.profileRepo.FindModelProfileByID(modelID)
	if err != nil {
		return nil, errors.New("model not found")
	}

	criteria := &dto.MatchCriteria{
		City:       targetModel.City,
		Categories: s.parseCategories(targetModel.Categories),
		Gender:     targetModel.Gender,
		Limit:      limit + 1,
		MinScore:   30.0,
	}

	models, err := s.FindModelsByCriteria(criteria)
	if err != nil {
		return nil, err
	}

	var similarModels []*dto.SimilarModel
	for _, match := range models {
		if match.ModelID != modelID {
			similarModels = append(similarModels, &dto.SimilarModel{
				ModelID:          match.ModelID,
				Name:             match.ModelName,
				City:             targetModel.City,
				Similarity:       match.Score,
				CommonCategories: s.parseCategories(targetModel.Categories),
			})
		}
	}

	if len(similarModels) > limit {
		similarModels = similarModels[:limit]
	}

	return similarModels, nil
}

// -------------------------------
// Batch, configuration, analytics
// -------------------------------

func (s *matchingService) BatchMatchModels(castingIDs []string) (map[string][]*dto.MatchResult, error) {
	results := make(map[string][]*dto.MatchResult)
	for _, castingID := range castingIDs {
		matches, _ := s.FindMatchingModels(castingID, 10, 50.0)
		results[castingID] = matches
	}
	return results, nil
}

func (s *matchingService) UpdateModelRecommendations(modelID string) error {
	return nil
}

func (s *matchingService) GetMatchingWeights() (*dto.MatchingWeights, error) {
	return defaultWeights, nil
}

func (s *matchingService) UpdateMatchingWeights(adminID string, weights *dto.MatchingWeights) error {
	admin, err := s.profileRepo.FindEmployerProfileByUserID(adminID)
	if err != nil || !admin.IsVerified {
		return errors.New("insufficient permissions")
	}

	total := weights.Demographics + weights.Physical + weights.Professional +
		weights.Geographic + weights.Specialized
	if math.Abs(total-1.0) > 0.01 {
		return errors.New("weights must sum to 1.0")
	}

	defaultWeights = weights
	return nil
}

func (s *matchingService) GetMatchingStats(castingID string) (*dto.MatchingStats, error) {
	return &dto.MatchingStats{
		CastingID:         castingID,
		TotalModels:       0,
		MatchedModels:     0,
		AverageScore:      0.0,
		ScoreDistribution: make(map[string]int),
		TopCategories:     []string{},
	}, nil
}

func (s *matchingService) GetModelMatchingStats(modelID string) (*dto.ModelMatchingStats, error) {
	return &dto.ModelMatchingStats{
		ModelID:         modelID,
		TotalCastings:   0,
		MatchedCastings: 0,
		MatchRate:       0.0,
		AverageScore:    0.0,
		ResponseRate:    0.0,
		TopMatchReasons: []string{},
	}, nil
}

func (s *matchingService) GetPlatformMatchingStats() (*dto.PlatformMatchingStats, error) {
	return &dto.PlatformMatchingStats{
		TotalMatches:      0,
		SuccessfulMatches: 0,
		AverageMatchScore: 0.0,
		MatchRate:         0.0,
		ByCategory:        make(map[string]int64),
	}, nil
}

func (s *matchingService) RecalculateAllMatches(adminID string) error {
	return nil
}

func (s *matchingService) GetMatchingLogs(criteria dto.MatchingLogCriteria) ([]*dto.MatchingLog, int64, error) {
	return []*dto.MatchingLog{}, 0, nil
}

// -------------------------------
// Helpers
// -------------------------------

func (s *matchingService) parseCategories(categoriesData datatypes.JSON) []string {
	return dto.ParseCategories(categoriesData)
}

func (s *matchingService) formatCategories(categories []string) datatypes.JSON {
	return dto.FormatCategories(categories)
}

func getIntValue(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func (s *matchingService) notifyTopMatches(casting *models.Casting, matches []*dto.MatchResult) {
	// Placeholder: notification logic here
}

func (s *matchingService) calculateDemographicsScore(model *models.ModelProfile, casting *models.Casting) float64 {
	return 80.0
}

func (s *matchingService) calculatePhysicalScore(model *models.ModelProfile, casting *models.Casting) float64 {
	return 75.0
}

func (s *matchingService) calculateProfessionalScore(model *models.ModelProfile, casting *models.Casting) float64 {
	return 70.0
}

func (s *matchingService) calculateGeographicScore(model *models.ModelProfile, casting *models.Casting) float64 {
	return 85.0
}

func (s *matchingService) calculateSpecializedScore(model *models.ModelProfile, casting *models.Casting) float64 {
	return 90.0
}

func (s *matchingService) generateMatchReasons(score *dto.MatchScore, model *models.ModelProfile, casting *models.Casting) []string {
	return []string{"Good demographic fit", "Matches physical criteria"}
}

func (s *matchingService) generateRecommendations(score *dto.MatchScore, model *models.ModelProfile, casting *models.Casting) []string {
	return []string{"Recommended for this casting"}
}
