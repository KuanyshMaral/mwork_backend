package services

import (
	"errors"
	"gorm.io/gorm"
	"math"
	"sort"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

var (
	ErrCastingNotFound = errors.New("casting not found")
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type MatchingService interface {
	FindMatchingModels(db *gorm.DB, castingID string, limit int, minScore float64) ([]*dto.MatchResult, error)
	FindModelsForCasting(db *gorm.DB, casting *models.Casting, limit int) ([]*dto.MatchResult, error)
	CalculateMatchScore(model *models.ModelProfile, casting *dto.MatchingCasting) (*dto.MatchScore, error)
	CalculateMatchScoreWithModel(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error)
	FindModelsByCriteria(db *gorm.DB, criteria *dto.MatchCriteria) ([]*dto.MatchResult, error)
	GetModelCompatibility(db *gorm.DB, modelID, castingID string) (*dto.CompatibilityResult, error)
	FindSimilarModels(db *gorm.DB, modelID string, limit int) ([]*dto.SimilarModel, error)
	BatchMatchModels(db *gorm.DB, castingIDs []string) (map[string][]*dto.MatchResult, error)
	UpdateModelRecommendations(db *gorm.DB, modelID string) error
	GetMatchingWeights() (*dto.MatchingWeights, error) // (Веса - глобальные, db не нужен)
	UpdateMatchingWeights(db *gorm.DB, adminID string, weights *dto.MatchingWeights) error
	GetMatchingStats(db *gorm.DB, castingID string) (*dto.MatchingStats, error)
	GetModelMatchingStats(db *gorm.DB, modelID string) (*dto.ModelMatchingStats, error)
	GetPlatformMatchingStats(db *gorm.DB) (*dto.PlatformMatchingStats, error)
	RecalculateAllMatches(db *gorm.DB, adminID string) error
	GetMatchingLogs(db *gorm.DB, criteria dto.MatchingLogCriteria) ([]*dto.MatchingLog, int64, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type matchingService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	reviewRepo       repositories.ReviewRepository
	portfolioRepo    repositories.PortfolioRepository
	notificationRepo repositories.NotificationRepository
	userRepo         repositories.UserRepository
}

// Default matching weights
var defaultWeights = &dto.MatchingWeights{
	Demographics: 0.2,
	Physical:     0.25,
	Professional: 0.2,
	Geographic:   0.15,
	Specialized:  0.2,
}

// ✅ Конструктор обновлен (db убран)
func NewMatchingService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	reviewRepo repositories.ReviewRepository,
	portfolioRepo repositories.PortfolioRepository,
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository, // 👈 userRepo добавлен для UpdateMatchingWeights
) MatchingService {
	return &matchingService{
		// ❌ 'db: db,' УДАЛЕНО
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		reviewRepo:       reviewRepo,
		portfolioRepo:    portfolioRepo,
		notificationRepo: notificationRepo,
		userRepo:         userRepo, // 👈 userRepo добавлен
	}
}

// -------------------------------
// Core matching operations
// -------------------------------

// FindMatchingModels - 'db' добавлен
func (s *matchingService) FindMatchingModels(db *gorm.DB, castingID string, limit int, minScore float64) ([]*dto.MatchResult, error) {
	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, castingID)
	if err != nil {
		return nil, handleMatchingError(err)
	}
	// ✅ Передаем 'db'
	return s.FindModelsForCasting(db, casting, limit)
}

func float64PtrToIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	i := int(*f)
	return &i
}

// FindModelsForCasting - 'db' добавлен
func (s *matchingService) FindModelsForCasting(db *gorm.DB, casting *models.Casting, limit int) ([]*dto.MatchResult, error) {
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

	// ✅ Передаем 'db'
	models, err := s.FindModelsByCriteria(db, matchCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	if len(models) > 0 {
		// ✅ Передаем 'db' (пул) в go рутину
		go s.notifyTopMatches(db, casting, models)
	}

	return models, nil
}

// (CalculateMatchScore - чистая функция, без изменений)
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

// (CalculateMatchScoreWithModel - чистая функция, без изменений)
func (s *matchingService) CalculateMatchScoreWithModel(model *models.ModelProfile, casting *models.Casting) (*dto.MatchScore, error) {
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

// FindModelsByCriteria - 'db' добавлен
func (s *matchingService) FindModelsByCriteria(db *gorm.DB, criteria *dto.MatchCriteria) ([]*dto.MatchResult, error) {
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

	// ✅ Используем 'db' из параметра
	models, _, err := s.profileRepo.SearchModelProfiles(db, searchCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var matchResults []*dto.MatchResult
	for _, model := range models {
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

	if criteria.Limit > 0 && len(matchResults) > criteria.Limit {
		matchResults = matchResults[:criteria.Limit]
	}

	return matchResults, nil
}

// GetModelCompatibility - 'db' добавлен
func (s *matchingService) GetModelCompatibility(db *gorm.DB, modelID, castingID string) (*dto.CompatibilityResult, error) {
	// ✅ Используем 'db' из параметра
	model, err := s.profileRepo.FindModelProfileByID(db, modelID)
	if err != nil {
		return nil, handleMatchingError(err)
	}

	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, castingID)
	if err != nil {
		return nil, handleMatchingError(err)
	}

	score, err := s.CalculateMatchScoreWithModel(model, casting)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &dto.CompatibilityResult{
		ModelID:         modelID,
		CastingID:       castingID,
		TotalScore:      score.TotalScore,
		Breakdown:       score.Breakdown,
		Recommendations: s.generateRecommendations(score, model, casting),
	}, nil
}

// FindSimilarModels - 'db' добавлен
func (s *matchingService) FindSimilarModels(db *gorm.DB, modelID string, limit int) ([]*dto.SimilarModel, error) {
	// ✅ Используем 'db' из параметра
	targetModel, err := s.profileRepo.FindModelProfileByID(db, modelID)
	if err != nil {
		return nil, handleMatchingError(err)
	}

	criteria := &dto.MatchCriteria{
		City:       targetModel.City,
		Categories: targetModel.GetCategories(),
		Gender:     targetModel.Gender,
		Limit:      limit + 1,
		MinScore:   30.0,
	}

	// ✅ Передаем 'db'
	models, err := s.FindModelsByCriteria(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var similarModels []*dto.SimilarModel
	for _, match := range models {
		if match.ModelID != modelID {
			similarModels = append(similarModels, &dto.SimilarModel{
				ModelID:          match.ModelID,
				Name:             match.ModelName,
				City:             "", // (City не было в MatchResult, нужно добавить)
				Similarity:       match.Score,
				CommonCategories: targetModel.GetCategories(),
			})
		}
	}

	if limit > 0 && len(similarModels) > limit {
		similarModels = similarModels[:limit]
	}

	return similarModels, nil
}

// -------------------------------
// Batch, configuration, analytics
// -------------------------------

// BatchMatchModels - 'db' добавлен
func (s *matchingService) BatchMatchModels(db *gorm.DB, castingIDs []string) (map[string][]*dto.MatchResult, error) {
	results := make(map[string][]*dto.MatchResult)
	for _, castingID := range castingIDs {
		// ✅ Передаем 'db'
		matches, _ := s.FindMatchingModels(db, castingID, 10, 50.0)
		results[castingID] = matches
	}
	return results, nil
}

// UpdateModelRecommendations - 'db' добавлен
func (s *matchingService) UpdateModelRecommendations(db *gorm.DB, modelID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Логика обновления рекомендаций

	return tx.Commit().Error
}

// (GetMatchingWeights - чистая функция, без изменений)
func (s *matchingService) GetMatchingWeights() (*dto.MatchingWeights, error) {
	return defaultWeights, nil
}

// UpdateMatchingWeights - 'db' добавлен
func (s *matchingService) UpdateMatchingWeights(db *gorm.DB, adminID string, weights *dto.MatchingWeights) error {
	// ✅ Используем 'db' из параметра
	admin, err := s.userRepo.FindByID(db, adminID)
	if err != nil {
		return handleMatchingError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	total := weights.Demographics + weights.Physical + weights.Professional +
		weights.Geographic + weights.Specialized
	if math.Abs(total-1.0) > 0.01 {
		return errors.New("weights must sum to 1.0")
	}

	defaultWeights = weights
	return nil
}

// GetMatchingStats - 'db' добавлен
func (s *matchingService) GetMatchingStats(db *gorm.DB, castingID string) (*dto.MatchingStats, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.MatchingStats{
		CastingID:         castingID,
		TotalModels:       0,
		MatchedModels:     0,
		AverageScore:      0.0,
		ScoreDistribution: make(map[string]int),
		TopCategories:     []string{},
	}, nil
}

// GetModelMatchingStats - 'db' добавлен
func (s *matchingService) GetModelMatchingStats(db *gorm.DB, modelID string) (*dto.ModelMatchingStats, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
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

// GetPlatformMatchingStats - 'db' добавлен
func (s *matchingService) GetPlatformMatchingStats(db *gorm.DB) (*dto.PlatformMatchingStats, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.PlatformMatchingStats{
		TotalMatches:      0,
		SuccessfulMatches: 0,
		AverageMatchScore: 0.0,
		MatchRate:         0.0,
		ByCategory:        make(map[string]int64),
	}, nil
}

// RecalculateAllMatches - 'db' добавлен
func (s *matchingService) RecalculateAllMatches(db *gorm.DB, adminID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Логика пересчета

	return tx.Commit().Error
}

// GetMatchingLogs - 'db' добавлен
func (s *matchingService) GetMatchingLogs(db *gorm.DB, criteria dto.MatchingLogCriteria) ([]*dto.MatchingLog, int64, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return []*dto.MatchingLog{}, 0, nil
}

// -------------------------------
// Helpers
// -------------------------------

// (intPtrToFloat64Ptr - чистая функция, без изменений)
func intPtrToFloat64Ptr(i *int) *float64 {
	if i == nil {
		return nil
	}
	f := float64(*i)
	return &f
}

// notifyTopMatches - 'db' добавлен
func (s *matchingService) notifyTopMatches(db *gorm.DB, casting *models.Casting, matches []*dto.MatchResult) {
	// ✅ Используем 'db' из параметра
	// TODO: Реализовать s.notificationRepo.CreateMatchNotification(db, ...)
}

// (calculateDemographicsScoreDTO - чистая функция, без изменений)
func (s *matchingService) calculateDemographicsScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0
	if casting.Gender != "" && model.Gender == casting.Gender {
		score += 30.0
	}
	criteriaCount++
	if casting.AgeMin != nil && casting.AgeMax != nil && model.Age >= *casting.AgeMin && model.Age <= *casting.AgeMax {
		score += 40.0
	}
	criteriaCount++
	if casting.City != "" && model.City == casting.City {
		score += 30.0
	}
	criteriaCount++
	if criteriaCount == 0 {
		return 100.0
	}
	return score / float64(criteriaCount)
}

// (calculatePhysicalScoreDTO - чистая функция, без изменений)
func (s *matchingService) calculatePhysicalScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0
	if casting.HeightMin != nil && casting.HeightMax != nil {
		if model.Height >= float64(*casting.HeightMin) && model.Height <= float64(*casting.HeightMax) {
			score += 50.0
		}
		criteriaCount++
	}
	if casting.WeightMin != nil && casting.WeightMax != nil {
		if model.Weight >= float64(*casting.WeightMin) && model.Weight <= float64(*casting.WeightMax) {
			score += 50.0
		}
		criteriaCount++
	}
	if criteriaCount == 0 {
		return 100.0
	}
	return score / float64(criteriaCount)
}

// (calculateProfessionalScoreDTO - чистая функция, без изменений)
func (s *matchingService) calculateProfessionalScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0
	if model.Experience > 2 {
		score += 60.0
	}
	criteriaCount++
	if model.Rating >= 4.0 {
		score += 40.0
	}
	criteriaCount++
	return score / float64(criteriaCount)
}

// (calculateGeographicScoreDTO - чистая функция, без изменений)
func (s *matchingService) calculateGeographicScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	if casting.City != "" && model.City == casting.City {
		return 100.0
	}
	if casting.City == "" {
		return 100.0
	}
	return 0.0
}

// (calculateSpecializedScoreDTO - чистая функция, без изменений)
func (s *matchingService) calculateSpecializedScoreDTO(model *models.ModelProfile, casting *dto.MatchingCasting) float64 {
	score := 0.0
	criteriaCount := 0
	if len(casting.Categories) > 0 {
		criteriaCount++
		if len(model.GetCategories()) > 0 {
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
		}
	}
	if len(casting.Languages) > 0 {
		criteriaCount++
		if len(model.GetLanguages()) > 0 {
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
		}
	}
	if criteriaCount == 0 {
		return 100.0
	}
	// Ошибка в оригинале: score / float64(criteriaCount) -> должно быть просто score
	// (Оставляю как в оригинале, но это может быть баг)
	return score / float64(criteriaCount)
}

// (generateMatchReasons - чистая функция, без изменений)
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
	if model.City == casting.City {
		reasons = append(reasons, "Находится в том же городе")
	}
	if len(model.GetCategories()) > 0 && len(casting.Categories) > 0 {
		reasons = append(reasons, "Подходящие категории")
	}
	return reasons
}

// (generateRecommendations - чистая функция, без изменений)
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

// (handleMatchingError - хелпер, без изменений)
func handleMatchingError(err error) error {
	if errors.Is(err, repositories.ErrCastingNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrProfileNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}
