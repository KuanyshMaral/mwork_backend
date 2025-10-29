package dto

// ==============================
// 📊 PLATFORM OVERVIEW DTO
// ==============================

type PlatformOverview struct {
	GrowthMetrics       GrowthMetrics         `json:"growthMetrics"`
	UserAnalytics       UserAnalytics         `json:"userAnalytics"`
	CastingAnalytics    CastingAnalytics      `json:"castingAnalytics"`
	MatchingAnalytics   MatchingAnalytics     `json:"matchingAnalytics"`
	FinancialAnalytics  FinancialAnalytics    `json:"financialAnalytics"`
	GeographicAnalytics GeographicAnalytics   `json:"geographicAnalytics"`
	CategoryAnalytics   CategoryAnalytics     `json:"categoryAnalytics"`
	PerformanceMetrics  PerformanceMetrics    `json:"performanceMetrics"`
	PlatformHealth      PlatformHealthMetrics `json:"platformHealth"`
	Reports             []CustomReport        `json:"reports"`
	RealTimeMetrics     RealTimeMetrics       `json:"realTimeMetrics"`
	AdminDashboard      AdminDashboard        `json:"adminDashboard"`
}

// ==============================
// 📈 GROWTH & USER ANALYTICS
// ==============================

type GrowthMetrics struct {
	TotalUsers        int         `json:"totalUsers"`
	NewUsersThisMonth int         `json:"newUsersThisMonth"`
	MonthlyGrowthRate float64     `json:"monthlyGrowthRate"`
	ActiveUsers       int         `json:"activeUsers"`
	ChurnRate         float64     `json:"churnRate"`
	HistoricalTrends  []DataPoint `json:"historicalTrends"`
}

type UserAnalytics struct {
	Acquisition UserAcquisitionMetrics `json:"acquisition"`
	Retention   UserRetentionMetrics   `json:"retention"`
	Activity    UserActivity           `json:"activity"`
	Churn       ChurnAnalysis          `json:"churn"`
}

type UserAcquisitionMetrics struct {
	NewUsers       int         `json:"newUsers"`
	ReturningUsers int         `json:"returningUsers"`
	ConversionRate float64     `json:"conversionRate"`
	Sources        []DataPoint `json:"sources"`
}

type UserRetentionMetrics struct {
	RetentionRate   float64      `json:"retentionRate"`
	AverageLifespan float64      `json:"averageLifespan"`
	Cohorts         []CohortData `json:"cohorts"`
}

type UserActivity struct {
	DailyActiveUsers   int `json:"dailyActiveUsers"`
	WeeklyActiveUsers  int `json:"weeklyActiveUsers"`
	MonthlyActiveUsers int `json:"monthlyActiveUsers"`
}

// ==============================
// 🎭 CASTING ANALYTICS
// ==============================

type CastingAnalytics struct {
	TotalCastings       int                       `json:"totalCastings"`
	ActiveCastings      int                       `json:"activeCastings"`
	SuccessRate         float64                   `json:"successRate"`
	AverageResponseTime float64                   `json:"averageResponseTime"`
	Performance         CastingPerformanceMetrics `json:"performance"`
	Categories          []*CategoryStats          `json:"categories"` // Измени на указатели
}

type CastingPerformanceMetrics struct {
	CompletionRate float64 `json:"completionRate"`
	EngagementRate float64 `json:"engagementRate"`
}

// ==============================
// ❤️ MATCHING ANALYTICS
// ==============================

type MatchingAnalytics struct {
	TotalMatches int                       `json:"totalMatches"`
	MatchQuality MatchQuality              `json:"matchQuality"`
	Algorithm    AlgorithmPerformance      `json:"algorithm"`
	Efficiency   MatchingEfficiencyMetrics `json:"efficiency"`
}

type MatchQuality struct {
	AverageScore     float64 `json:"averageScore"`
	UserSatisfaction float64 `json:"userSatisfaction"`
}

type AlgorithmPerformance struct {
	Accuracy  float64 `json:"accuracy"`
	LatencyMS int     `json:"latencyMs"`
	Version   string  `json:"version"`
}

type MatchingEfficiencyMetrics struct {
	AverageMatchTime float64 `json:"averageMatchTime"`
	MatchRate        float64 `json:"matchRate"`
	DropOffRate      float64 `json:"dropOffRate"`
}

// ==============================
// 💰 FINANCIAL ANALYTICS
// ==============================

type FinancialAnalytics struct {
	TotalRevenue     float64             `json:"totalRevenue"`
	MonthlyRevenue   []DataPoint         `json:"monthlyRevenue"`
	ARPU             float64             `json:"arpu"`
	SubscriptionData SubscriptionMetrics `json:"subscriptionData"`
}

type SubscriptionMetrics struct {
	ActiveSubscriptions int     `json:"activeSubscriptions"`
	Cancellations       int     `json:"cancellations"`
	RevenueShare        float64 `json:"revenueShare"`
}

// ==============================
// 🌍 GEOGRAPHIC & CATEGORY
// ==============================

type GeographicAnalytics struct {
	Countries []CityStats       `json:"countries"`
	Cities    []CityPerformance `json:"cities"`
}

type CityStats struct {
	Name      string  `json:"name"`
	UserCount int     `json:"userCount"`
	Revenue   float64 `json:"revenue"`
}

type CityPerformance struct {
	Name           string  `json:"name"`
	EngagementRate float64 `json:"engagementRate"`
	ResponseTime   float64 `json:"responseTime"`
}

type CategoryAnalytics struct {
	Categories []*CategoryStats `json:"categories"`
}

type CategoryStats struct {
	Name   string  `json:"name"`
	Count  int     `json:"count"`
	Rating float64 `json:"rating"`
}

// ==============================
// ⚙️ PERFORMANCE & SYSTEM HEALTH
// ==============================

type PerformanceMetrics struct {
	ResponseTimes ResponseTimes `json:"responseTimes"`
	Throughput    Throughput    `json:"throughput"`
}

type ResponseTimes struct {
	AverageMS float64 `json:"averageMs"`
	MaxMS     float64 `json:"maxMs"`
}

type Throughput struct {
	RequestsPerSecond float64 `json:"requestsPerSecond"`
}

type PlatformHealthMetrics struct {
	SystemHealth   SystemHealth   `json:"systemHealth"`
	ResourceUsage  ResourceUsage  `json:"resourceUsage"`
	DatabaseHealth DatabaseHealth `json:"databaseHealth"`
	APIPerformance APIPerformance `json:"apiPerformance"`
	StorageUsage   StorageUsage   `json:"storageUsage"`
}

type ResourceUsage struct {
	CPUUsage float64 `json:"cpuUsage"`
	RAMUsage float64 `json:"ramUsage"`
}

type DatabaseHealth struct {
	Status       string  `json:"status"`
	QueryLatency float64 `json:"queryLatency"`
}

type APIPerformance struct {
	AverageLatency float64 `json:"averageLatency"`
	ErrorRate      float64 `json:"errorRate"`
}

type StorageUsage struct {
	UsedGB  float64 `json:"usedGB"`
	TotalGB float64 `json:"totalGB"`
}

// ==============================
// 🧠 REPORTS & ADMIN DASHBOARD
// ==============================

type CustomReport struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Data        any    `json:"data"`
}

type CustomReportRequest struct {
	Type   string   `json:"type"`
	Period string   `json:"period"`
	Fields []string `json:"fields"`
}

type RealTimeMetrics struct {
	KeyMetrics      KeyMetrics      `json:"keyMetrics"`
	UserActivity    UserActivity    `json:"userActivity"`
	CastingActivity CastingActivity `json:"castingActivity"`
	MatchActivity   MatchActivity   `json:"matchActivity"`
	SystemEvents    []SystemEvent   `json:"systemEvents"`
}

type KeyMetrics struct {
	ActiveUsers int     `json:"activeUsers"`
	NewUsers    int     `json:"newUsers"`
	Revenue     float64 `json:"revenue"`
}

type CastingActivity struct {
	OpenCastings int `json:"openCastings"`
	NewCastings  int `json:"newCastings"`
}

type MatchActivity struct {
	ActiveMatches int `json:"activeMatches"`
	NewMatches    int `json:"newMatches"`
}

type SystemEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"eventType"`
	Message   string `json:"message"`
}

type AdminDashboard struct {
	RecentActivity []RecentActivity `json:"recentActivity"`
	SystemHealth   SystemHealth     `json:"systemHealth"`
	Alerts         []Alert          `json:"alerts"`
}

type RecentActivity struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"userId"`
	Action    string `json:"action"`
}

type SystemHealth struct {
	CPUUsage float64 `json:"cpuUsage"`
	RAMUsage float64 `json:"ramUsage"`
}

type Alert struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ==============================
// 🧮 SUPPORT STRUCTURES
// ==============================

type DataPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type CohortData struct {
	CohortName string  `json:"cohortName"`
	Retention  float64 `json:"retention"`
}

type ChurnAnalysis struct {
	Reasons []DataPoint `json:"reasons"`
	Rate    float64     `json:"rate"`
}

type PredefinedReport struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
