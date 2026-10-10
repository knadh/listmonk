package main

import (
	"strings"
	"unicode"
)

// makeInitials returns a two letter abbreviation of the given name.
func makeInitials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) == 0 {
		return "?"
	}

	out := []rune{[]rune(words[0])[0]}
	if len(words) > 1 {
		out = append(out, []rune(words[1])[0])
	} else if r := []rune(words[0]); len(r) > 1 {
		out = append(out, r[1])
	}

	return strings.ToUpper(string(out))
}
