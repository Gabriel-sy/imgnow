package util

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateHash() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func HashPassword(password *string) (*string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(*password), 14)
	if err != nil {
		return nil, err
	}
	hashedPassword := string(bytes)
	return &hashedPassword, nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
