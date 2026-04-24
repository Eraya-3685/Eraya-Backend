package util

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	// Cost 12 = ~250ms — industry standard (cost 14 = ~1-2s, unnecessarily slow for an API)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
