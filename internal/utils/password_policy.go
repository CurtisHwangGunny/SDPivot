package utils

import (
	"errors"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"
)

type PasswordPolicy struct {
	MinLength        int
	RequireComplexity bool
	RotationDays     int
}

func ValidatePasswordPolicy(password string, policy PasswordPolicy) error {
	if utf8.RuneCountInString(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters", policy.MinLength)
	}
	if !policy.RequireComplexity {
		return nil
	}
	var upper, lower, digit, special bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			special = true
		}
	}
	if !upper || !lower || !digit || !special {
		return errors.New("password must include uppercase, lowercase, number, and special characters")
	}
	return nil
}

func PasswordExpiry(changedAt time.Time, rotationDays int) *time.Time {
	if rotationDays <= 0 {
		return nil
	}
	expiresAt := changedAt.AddDate(0, 0, rotationDays)
	return &expiresAt
}

func IsPasswordExpired(now time.Time, expiresAt *time.Time) bool {
	return expiresAt != nil && !now.Before(*expiresAt)
}
