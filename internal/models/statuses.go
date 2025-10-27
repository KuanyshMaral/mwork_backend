package models

type UserStatus string
type UserRole string
type CastingStatus string
type ResponseStatus string
type SubscriptionStatus string
type PaymentStatus string

const (
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"

	UserRoleModel    UserRole = "model"
	UserRoleEmployer UserRole = "employer"
	UserRoleAdmin    UserRole = "admin"

	CastingStatusDraft     CastingStatus = "draft"
	CastingStatusActive    CastingStatus = "active"
	CastingStatusClosed    CastingStatus = "closed"
	CastingStatusCancelled CastingStatus = "cancelled"

	ResponseStatusPending   ResponseStatus = "pending"
	ResponseStatusAccepted  ResponseStatus = "accepted"
	ResponseStatusRejected  ResponseStatus = "rejected"
	ResponseStatusWithdrawn ResponseStatus = "withdrawn"

	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"

	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)
