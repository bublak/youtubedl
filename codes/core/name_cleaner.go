package core

import (
	"strings"
	"unicode"
)

// CleanCharactersFromString keeps only Czech alphabet characters, numbers, and word separators
func CleanCharactersFromString(str string) (cleanedString string) {
	// Use a more efficient approach with unicode filtering
	cleanedString = strings.Map(func(r rune) rune {
		// Keep Czech alphabet (including diacritics)
		if isCzechLetter(r) {
			return r
		}
		// Keep numbers
		if unicode.IsDigit(r) {
			return r
		}
		// Keep word separators
		if r == '_' || r == '-' {
			return r
		}
		// Convert spaces and other characters to underscores
		if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return '_'
		}
		// Replace any other character with underscore
		return '_'
	}, str)

	// Clean up multiple consecutive underscores and trim
	cleanedString = strings.ReplaceAll(cleanedString, "___", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "__", "_")
	cleanedString = strings.Trim(cleanedString, "_")

	return cleanedString
}

// isCzechLetter checks if a rune is a Czech alphabet letter (including diacritics)
func isCzechLetter(r rune) bool {
	// Basic Latin letters (a-z, A-Z)
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}

	// Czech diacritical marks
	czechDiacritics := "áčďéěíňóřšťúůýžÁČĎÉĚÍŇÓŘŠŤÚŮÝŽ"
	for _, czech := range czechDiacritics {
		if r == czech {
			return true
		}
	}

	return false
}
