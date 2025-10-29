package services

import (
	"errors"
	"math"
	"sort"

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
	CalculateMatchScore(model *models.ModelProfile, casting *dto.MatchingCasting) (*dto.MatchScore, error)
	CalculateMatchScoreWithModel(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error)

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
	// Конвертируем модель Casting в DTO MatchingCasting
	criteria := &dto.MatchingCasting{
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

	// Создаем MatchCriteria из MatchingCasting
	matchCriteria := &dto.MatchCriteria{
		City:       criteria.City,
		Categories: criteria.Categories,
		Gender:     criteria.Gender,
		MinAge:     criteria.AgeMin,
		MaxAge:     criteria.AgeMax,
		MinHeight:  float64PtrToIntPtr(criteria.HeightMin),
		MaxHeight:  float64PtrToIntPtr(criteria.HeightMax),
		MinWeight:  float64PtrToIntPtr(criteria.WeightMin),
		MaxWeight:  float64PtrToIntPtr(criteria.WeightMax),
		Languages:  criteria.Languages,
		Limit:      limit,
		MinScore:   50.0,
	}

	models, err := s.FindModelsByCriteria(matchCriteria)
	if err != nil {
		return nil, err
	}

	if len(models) > 0 {
		go s.notifyTopMatches(casting, models)
	}

	return models, nil
}

func (s *matchingService) CalculateMatchScore(model *models.ModelProfile, casting *dto.MatchingCasting) (*dto.MatchScore, error) {
	breakdown := &dto.CompatibilityBreakdown{}
	categoryScores := make(map[string]float64)

	demographicsScore := s.calculateDemographicsScoreDTO(model, casting)
	breakdown.Demographics = demographicsScore
	categoryScores["demographics"] = demographicsScore

	physicalScore := s.calculatePhysicalScoreDTO(model, casting)
	breakdown.Physical = physicalScore
	categoryScores["physical"] = physicalScore

	professionalScore := s.calculateProfessionalScoreDTO(model, casting)
	breakdown.Professional = professionalScore
	categoryScores["professional"] = professionalScore

	geographicScore := s.calculateGeographicScoreDTO(model, casting)
	breakdown.Geographic = geographicScore
	categoryScores["geographic"] = geographicScore

	specializedScore := s.calculateSpecializedScoreDTO(model, casting)
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

func (s *matchingService) CalculateMatchScoreWithModel(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error) {
	// Конвертируем модель в DTO
	castingDTO := &dto.MatchingCasting{
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

	return s.CalculateMatchScore(model, castingDTO)
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
		// Используем DTO вместо модели
		mockCasting := &dto.MatchingCasting{
			City:       criteria.City,
			Categories: criteria.Categories,
			Gender:     criteria.Gender,
			AgeMin:     criteria.MinAge,
			AgeMax:     criteria.MaxAge,
			HeightMin:  intPtrToFloat64Ptr(criteria.MinHeight),
			HeightMax:  intPtrToFloat64Ptr(criteria.MaxHeight),
			WeightMin:  intPtrToFloat64Ptr(criteria.MinWeight),
			WeightMax:  intPtrToFloat64Ptr(criteria.MaxWeight),
			JobType:    criteria.JobType,
			Languages:  criteria.Languages,
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

	// Используем метод для работы с моделями
	score, err := s.CalculateMatchScoreWithModel(model, casting)
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
		Categories: targetModel.GetCategories(),
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
				CommonCategories: targetModel.GetCategories(),
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

// Вспомогательная функция для конвертации *int в *float64
func intPtrToFloat64Ptr(i *int) *float64 {
	if i == nil {
		return nil
	}
	f := float64(*i)
	return &f
}

func (s *matchingService) notifyTopMatches(casting *models.Casting, matches []*dto.MatchResult) {
	// Placeholder: notification logic here
}

// DTO-based scoring methods
func (s *matchingService) calculateDemographicsScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0

	// Gender match
	if casting.Gender != "" && model.Gender == casting.Gender {
		score += 30.0
	}
	criteriaCount++

	// Age match
	if casting.AgeMin != nil && casting.AgeMax != nil && model.Age >= *casting.AgeMin && model.Age <= *casting.AgeMax {
		score += 40.0
	}
	criteriaCount++

	// City match
	if casting.City != "" && model.City == casting.City {
		score += 30.0
	}
	criteriaCount++

	return score / float64(criteriaCount)
}

func (s *matchingService) calculatePhysicalScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0

	// Height match
	if casting.HeightMin != nil && casting.HeightMax != nil {
		if model.Height >= int(*casting.HeightMin) && model.Height <= int(*casting.HeightMax) {
			score += 50.0
		}
		criteriaCount++
	}

	// Weight match
	if casting.WeightMin != nil && casting.WeightMax != nil {
		if model.Weight >= int(*casting.WeightMin) && model.Weight <= int(*casting.WeightMax) {
			score += 50.0
		}
		criteriaCount++
	}

	if criteriaCount == 0 {
		return 100.0 // No physical criteria specified
	}

	return score / float64(criteriaCount)
}

func (s *matchingService) calculateProfessionalScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0

	// Experience level match (simplified)
	if model.Experience > 2 {
		score += 60.0
	}
	criteriaCount++

	// Rating match
	if model.Rating >= 4.0 {
		score += 40.0
	}
	criteriaCount++

	return score / float64(criteriaCount)
}

func (s *matchingService) calculateGeographicScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	// Simple city-based scoring
	if casting.City != "" && model.City == casting.City {
		return 100.0
	}
	return 0.0
}

func (s *matchingService) calculateSpecializedScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0

	// Category match
	if len(casting.Categories) > 0 && len(model.GetCategories()) > 0 {
		commonCategories := 0
		for _, cat := range casting.Categories {
			for _, modelCat := range model.GetCategories() {
				if cat == modelCat {
					commonCategories++
					break
				}
			}
		}
		if commonCategories > 0 {
			score += float64(commonCategories) / float64(len(casting.Categories)) * 60.0
		}
		criteriaCount++
	}

	// Language match
	if len(casting.Languages) > 0 && len(model.GetLanguages()) > 0 {
		commonLanguages := 0
		for _, lang := range casting.Languages {
			for _, modelLang := range model.GetLanguages() {
				if lang == modelLang {
					commonLanguages++
					break
				}
			}
		}
		if commonLanguages > 0 {
			score += float64(commonLanguages) / float64(len(casting.Languages)) * 40.0
		}
		criteriaCount++
	}

	if criteriaCount == 0 {
		return 100.0 // No specialized criteria specified
	}

	return score / float64(criteriaCount)
}

func (s *matchingService) generateMatchReasons(score *dto.MatchScore, model *models.ModelProfile, casting *dto.MatchingCasting) []string {
	var reasons []string

	if score.Breakdown != nil {
		if score.Breakdown.Geographic > 80.0 {
			reasons = append(reasons, "Идеальное географическое соответствие")
		}
		if score.Breakdown.Demographics > 70.0 {
			reasons = append(reasons, "Соответствие демографическим требованиям")
		}
		if score.Breakdown.Physical > 60.0 {
			reasons = append(reasons, "Подходящие физические параметры")
		}
		if score.Breakdown.Professional > 50.0 {
			reasons = append(reasons, "Профессиональное соответствие")
		}
		if score.Breakdown.Specialized > 40.0 {
			reasons = append(reasons, "Специализированные навыки")
		}
	}

	// Дополнительные проверки на основе конкретных параметров
	if model.City == casting.City {
		reasons = append(reasons, "Находится в том же городе")
	}

	if len(model.GetCategories()) > 0 && len(casting.Categories) > 0 {
		reasons = append(reasons, "Подходящие категории")
	}

	return reasons
}

func (s *matchingService) generateRecommendations(score *dto.MatchScore, model *models.ModelProfile, casting *models.Casting) []string {
	var recommendations []string

	if score.Breakdown != nil {
		if score.Breakdown.Geographic < 50.0 {
			recommendations = append(recommendations, "Рассмотрите модели из других городов")
		}
		if score.Breakdown.Physical < 60.0 {
			recommendations = append(recommendations, "Расширьте физические критерии")
		}
		if score.Breakdown.Specialized < 40.0 {
			recommendations = append(recommendations, "Ищите модели с более специализированными навыками")
		}
	}

	if score.TotalScore > 80.0 {
		recommendations = append(recommendations, "Высокий потенциал для сотрудничества")
	}

	return recommendations
}
