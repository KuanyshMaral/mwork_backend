package dto

type PlanWithStats struct {
	ID           string
	Name         string
	Price        float64
	Currency     string
	DurationDays int
	UserCount    int
}

type PlanRevenue struct {
	PlanID        string
	PlanName      string
	TotalRevenue  float64
	PurchaseCount int
}

type PlanStats struct {
	PlanID     string
	TotalUsers int
}

type PlanBase struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	Currency     string  `json:"currency"`
	DurationDays int     `json:"duration_days"`
}
