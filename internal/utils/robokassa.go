package utils

import (
	"github.com/google/uuid"
)

func GenerateOrderID() string {
	return uuid.NewString()
}
