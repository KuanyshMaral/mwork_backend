package services

import (
	"mwork_backend/internal/dto"
	"mwork_backend/internal/repositories"
)

type AnalyticsService struct {
	analyticsRepo repositories.AnalyticsRepository
}

func NewAnalyticsService(repo repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: repo,
	}
}

func (s *AnalyticsService) GetModelAnalytics(modelID string) (dto.ModelAnalytics, error) {
	views, err := s.analyticsRepo.GetProfileViews(modelID)
	if err != nil {
		return dto.ModelAnalytics{}, err
	}

	rating, err := s.analyticsRepo.GetRating(modelID)
	if err != nil {
		return dto.ModelAnalytics{}, err
	}

	income, err := s.analyticsRepo.GetIncome(modelID)
	if err != nil {
		return dto.ModelAnalytics{}, err
	}

	responses, err := s.analyticsRepo.GetResponseCount(modelID)
	if err != nil {
		return dto.ModelAnalytics{}, err
	}

	return dto.ModelAnalytics{
		Views:     views,
		Rating:    rating,
		Income:    income,
		Responses: responses,
	}, nil
}
