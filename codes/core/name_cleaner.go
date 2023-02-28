package core

import (
	"strings"
	"unicode"
)

// CleanCharactersFromString should remove characters not suitable for file names, or Mp3 device displays
func CleanCharactersFromString(str string) (cleanedString string) {
	cleanedString = cleanCharacters(str)

	cleanedString = strings.ReplaceAll(cleanedString, " ", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "(", "_")
	cleanedString = strings.ReplaceAll(cleanedString, ")", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "[", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "&", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "%", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "*", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "!", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "|", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "]", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "/", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "`", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "@", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "#", "_")
	cleanedString = strings.ReplaceAll(cleanedString, ":", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "◆", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "'", "_")
	cleanedString = strings.ReplaceAll(cleanedString, ",", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "~", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "`", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "💀", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "\"", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "\\", "_")

	cleanedString = strings.ReplaceAll(cleanedString, "___", "_")
	cleanedString = strings.ReplaceAll(cleanedString, "__", "_")
	cleanedString = strings.TrimRight(cleanedString, "_")
	cleanedString = strings.TrimLeft(cleanedString, "_")

	return cleanedString
}

func cleanCharacters(str string) string {
	invisibleChars := str
	//fmt.Printf("%q\n", invisibleChars)
	//fmt.Println(len(invisibleChars))

	clean := strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, invisibleChars)

	//fmt.Printf("%q\n", clean)
	//fmt.Println(len(clean))

	clean = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, invisibleChars)

	//fmt.Printf("%q\n", clean)
	//fmt.Println(len(clean))

	return clean
}
