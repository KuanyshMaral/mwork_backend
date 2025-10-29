package algorithms

import (
	"encoding/json"
	"mwork_backend/internal/models"
)

// CalculateMatchScore calculates how well a model matches a casting (0-100)
func CalculateMatchScore(casting *models.Casting, model *models.ModelProfile) (float64, []string) {
	score := 0.0
	reasons := []string{}

	// City match (30 points)
	if casting.City == model.City {
		score += 30
		reasons = append(reasons, "Same city")
	}

	// Age match (20 points)
	if casting.AgeMin != nil && casting.AgeMax != nil {
		if model.Age >= *casting.AgeMin && model.Age <= *casting.AgeMax {
			score += 20
			reasons = append(reasons, "Age matches requirements")
		}
	} else {
		// No age requirement, give partial points
		score += 10
	}

	// Height match (15 points)
	if casting.HeightMin != nil && casting.HeightMax != nil {
		// FIX: Convert model.Height (int) to float64 for comparison
		if float64(model.Height) >= *casting.HeightMin && float64(model.Height) <= *casting.HeightMax {
			score += 15
			reasons = append(reasons, "Height matches requirements")
		}
	} else {
		score += 7
	}

	// Weight match (10 points)
	if casting.WeightMin != nil && casting.WeightMax != nil {
		// FIX: Convert model.Weight (int) to float64 for comparison
		if float64(model.Weight) >= *casting.WeightMin && float64(model.Weight) <= *casting.WeightMax {
			score += 10
			reasons = append(reasons, "Weight matches requirements")
		}
	} else {
		score += 5
	}

	// Gender match (10 points)
	if casting.Gender != "" {
		if casting.Gender == model.Gender {
			score += 10
			reasons = append(reasons, "Gender matches")
		}
	} else {
		score += 5
	}

	// Categories overlap (25 points)
	var castingCategories []string
	if len(casting.Categories) > 0 {
		json.Unmarshal(casting.Categories, &castingCategories)
	}
	// FIX: Use the model's getter method which returns []string
	categoryScore := calculateCategoryOverlap(castingCategories, model.GetCategories())
	score += categoryScore
	if categoryScore > 0 {
		reasons = append(reasons, "Matching categories")
	}

	// Clothing size match (5 points)
	if casting.ClothingSize != nil && *casting.ClothingSize == model.ClothingSize {
		score += 5
		reasons = append(reasons, "Clothing size matches")
	}

	// Shoe size match (5 points)
	if casting.ShoeSize != nil && *casting.ShoeSize == model.ShoeSize {
		score += 5
		reasons = append(reasons, "Shoe size matches")
	}

	// Experience match (10 points)
	if casting.ExperienceLevel != nil {
		// Simple heuristic: "junior" = 0-2 years, "middle" = 2-5 years, "senior" = 5+ years
		requiredExp := 0
		switch *casting.ExperienceLevel {
		case "junior":
			requiredExp = 0
		case "middle":
			requiredExp = 2
		case "senior":
			requiredExp = 5
		}
		if model.Experience >= requiredExp {
			score += 10
			reasons = append(reasons, "Sufficient experience")
		}
	} else {
		score += 5
	}

	// Language match (10 points)
	var castingLanguages []string
	if len(casting.Languages) > 0 {
		json.Unmarshal(casting.Languages, &castingLanguages)
	}
	// FIX: Use the model's getter method which returns []string
	languageScore := calculateLanguageOverlap(castingLanguages, model.GetLanguages())
	score += languageScore
	if languageScore > 0 {
		reasons = append(reasons, "Speaks required languages")
	}

	// Price compatibility (10 points)
	if casting.PaymentMin > 0 && casting.PaymentMax > 0 {
		// Check if model's hourly rate is within budget
		if model.HourlyRate >= casting.PaymentMin && model.HourlyRate <= casting.PaymentMax {
			score += 10
			reasons = append(reasons, "Price within budget")
		}
	} else {
		score += 5
	}

	// Bonus: High rating (up to 10 points)
	if model.Rating >= 4.5 {
		score += 10
		reasons = append(reasons, "High rating")
	} else if model.Rating >= 4.0 {
		score += 5
	}

	// Normalize to 0-100 scale (max possible is ~145, normalize to 100)
	normalizedScore := (score / 145.0) * 100.0
	if normalizedScore > 100 {
		normalizedScore = 100
	}

	return normalizedScore, reasons
}

// calculateCategoryOverlap calculates overlap between two category arrays (0-25 points)
func calculateCategoryOverlap(castingCategories, modelCategories []string) float64 {
	if len(castingCategories) == 0 {
		return 12.5 // No specific requirement, give half points
	}

	matches := 0
	for _, cc := range castingCategories {
		for _, mc := range modelCategories {
			if cc == mc {
				matches++
				break
			}
		}
	}

	// Calculate percentage of required categories that match
	overlapPercent := float64(matches) / float64(len(castingCategories))
	return overlapPercent * 25.0
}

// calculateLanguageOverlap calculates language overlap (0-10 points)
func calculateLanguageOverlap(castingLanguages, modelLanguages []string) float64 {
	if len(castingLanguages) == 0 {
		return 5 // No specific requirement, give half points
	}

	matches := 0
	for _, cl := range castingLanguages {
		for _, ml := range modelLanguages {
			if cl == ml {
				matches++
				break
			}
		}
	}

	// Calculate percentage of required languages that match
	overlapPercent := float64(matches) / float64(len(castingLanguages))
	return overlapPercent * 10.0
}
