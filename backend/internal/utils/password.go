package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword securely hashes a plain text password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain text password with a stored hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // Returns true if the password matches the hash
}

// ValidatePasswordPolicy enforces a basic strong password policy.
// Rules:
// - >= 10 characters
// - at least one lowercase, one uppercase, one digit, one special
// - not contain common weak substrings
func ValidatePasswordPolicy(pw string, disallowContains ...string) (ok bool, reason string) {
	if len(pw) < 10 {
		return false, "password must be at least 10 characters"
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range pw {
		switch {
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			// Consider visible ASCII specials; anything non-letter/digit counts
			hasSpecial = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
		return false, "password must include lowercase, uppercase, digit, and special character"
	}
	// Common weak patterns
	weak := []string{"password", "123456", "qwerty", "letmein", "admin"}
	for _, w := range weak {
		if len(pw) >= len(w) && containsFold(pw, w) {
			return false, "password is too common/guessable"
		}
	}
	for _, dis := range disallowContains {
		if dis == "" {
			continue
		}
		if containsFold(pw, dis) {
			return false, "password must not contain personal information"
		}
	}
	return true, ""
}

func containsFold(s, substr string) bool {
	// case-insensitive substring check; simple ASCII fold
	lower := func(b byte) byte {
		if b >= 'A' && b <= 'Z' {
			return b + 32
		}
		return b
	}
	n, m := len(s), len(substr)
	if m == 0 || m > n {
		return false
	}
	for i := 0; i <= n-m; i++ {
		match := true
		for j := 0; j < m; j++ {
			if lower(s[i+j]) != lower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
