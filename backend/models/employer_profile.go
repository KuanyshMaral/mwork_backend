package models

import "time"

type EmployerProfile struct {
	ID            string
	UserID        string
	CompanyName   string
	ContactPerson string
	Phone         string
	Website       string
	City          string
	CompanyType   string
	Description   string
	IsVerified    bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
