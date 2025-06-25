package utils

import (
	"math/rand"
	"regexp"
)

// check if the username is valid
// 3-20 characters, only letters, numbers and underscores
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

// check if the email is valid
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// check if the password is valid
// 8-20 characters, at least one letter and one number
func IsValidPassword(password string) bool {
	if len(password) < 8 || len(password) > 20 {
		return false
	}

	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)

	return hasLetter && hasNumber
}

// generate a random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
