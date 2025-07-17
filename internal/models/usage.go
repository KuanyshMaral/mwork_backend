package models

type Usage struct {
	Publications int `json:"publications"`
	Responses    int `json:"responses"`
	Messages     int `json:"messages"`
	Promotions   int `json:"promotions"`
}
