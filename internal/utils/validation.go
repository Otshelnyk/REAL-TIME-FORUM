package utils

import "strings"

// IsValidEmail performs basic email validation
func IsValidEmail(email string) bool {
	if strings.Count(email, "@") != 1 {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) < 1 || len(parts[1]) < 3 {
		return false
	}
	domain := parts[1]
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || !strings.Contains(domain, ".") {
		return false
	}
	return true
}

// IsValidUsername validates username format
func IsValidUsername(username string) bool {
	r := []rune(username)
	if len(r) < 3 || len(r) > 20 {
		return false
	}
	for _, ch := range r {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			return false
		}
		if !(isLetter(ch) || isDigit(ch)) {
			return false
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= 0x410 && r <= 0x44F) || (r >= 0xC0 && r <= 0xFF) || (r >= 0x0410 && r <= 0x042F) || (r >= 0x0430 && r <= 0x044F) || (r > 127 && r != ' ')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
