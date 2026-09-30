package main

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

var avatarColors = map[string][]struct {
	light string
	mid   string
	dark  string
}{
	"list": {
		{"#e1f7f3", "#78cbbb", "#234a43"},
		{"#e3f7f5", "#80cec6", "#234a47"},
		{"#e4f6f2", "#88cdbb", "#284b42"},
		{"#e0f5f4", "#72c5c2", "#214947"},
	},
	"campaign": {
		{"#eeebff", "#ad9ff5", "#303254"},
		{"#f0eaff", "#b29bf0", "#383052"},
		{"#edeaff", "#a89aee", "#323052"},
		{"#f2ecff", "#ba9feb", "#3c3152"},
	},
	"subscriber": {
		{"#e4f0ff", "#83b8fa", "#203657"},
		{"#e2f3ff", "#85c5f5", "#264356"},
		{"#e5f0ff", "#90b7fa", "#283d57"},
		{"#e1f2ff", "#7ebff9", "#204459"},
	},
	"user": {
		{"#faedf2", "#dfa4bb", "#50313e"},
		{"#f9edf3", "#d9a5bf", "#4b3143"},
		{"#fbedf0", "#e2a7b5", "#52343e"},
		{"#f8ecf2", "#d49eb5", "#4c3140"},
	},
	"role": {
		{"#edf0f4", "#a6b4c5", "#303c4c"},
		{"#eef1f5", "#afbac9", "#343f4d"},
		{"#ebf0f3", "#a1b5c1", "#2f414a"},
		{"#eef0f5", "#adb3c8", "#373d4e"},
	},
}

// avatar is the CSS style and the initials of a generated avatar.
type avatar struct {
	Style template.CSS
	Text  string
}

// makeAvatar generates a "glass" style avatar with a CSS gradient for an object type and seed.
// It returns the CSS style + initials of the name that can be printed in HTML.
func makeAvatar(kind, seed, name string) avatar {
	colors, ok := avatarColors[kind]
	if !ok {
		// Greyish default.
		colors = avatarColors["role"]
	}
	sum := sha256.Sum256([]byte(kind + ":" + strings.ToLower(strings.TrimSpace(seed))))

	// Pick colours.
	var (
		n   = len(colors)
		i   = int(sum[0]) % n
		one = colors[i]
		two = colors[(i+1+int(sum[1])%3)%n]
	)

	// Two hue blobs, a white shimmer on top, inspired by DiceBear's glass style.
	style := fmt.Sprintf("background-color:%s;color:%s;background-image:radial-gradient(circle at %d%% %d%%,#ffffff73,transparent 55%%),radial-gradient(circle at %d%% %d%%,%sa6,transparent %d%%),radial-gradient(circle at %d%% %d%%,%sa6,transparent %d%%)",
		one.light, one.dark,
		20+int(sum[8])%15, 10+int(sum[9])%15,
		15+int(sum[2])%30, 15+int(sum[3])%30, one.mid, 85+int(sum[4])%20,
		55+int(sum[5])%30, 55+int(sum[6])%30, two.mid, 80+int(sum[7])%20)

	return avatar{Style: template.CSS(style), Text: makeInitials(name)}
}

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
