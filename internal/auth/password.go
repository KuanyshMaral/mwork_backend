package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword создает bcrypt хеш пароля
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash проверяет пароль против хеша
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword проверяет сложность пароля
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	// Дополнительные проверки сложности пароля можно добавить здесь
	return nil
}
