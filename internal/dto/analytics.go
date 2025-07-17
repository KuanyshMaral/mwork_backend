package dto

type ModelAnalytics struct {
	Views     int     `json:"views"`
	Rating    float64 `json:"rating"`
	Income    float64 `json:"income"`
	Responses int     `json:"responses"`
}
