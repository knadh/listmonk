package main

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

var avatarColors = []struct {
	light string
	mid   string
	dark  string
}{
	{"#e4f0ff", "#83b8fa", "#203657"},
	{"#e8edff", "#95adfa", "#283457"},
	{"#eeebff", "#ad9ff5", "#303254"},
	{"#f2eaff", "#c29cf2", "#3c3152"},
	{"#f8e8ff", "#d69beb", "#46314f"},
	{"#ffe8f6", "#eb9cd4", "#50314a"},
	{"#ffe8ef", "#f59fbc", "#573443"},
	{"#ffeae7", "#faa99e", "#5b373a"},
	{"#fff0e3", "#f9bb8b", "#593d30"},
	{"#fff4df", "#f5ca7c", "#544328"},
	{"#fff7df", "#ecd77c", "#4d4828"},
	{"#f5f9e1", "#d1df86", "#434c2a"},
	{"#eaf9e6", "#ace099", "#384d2f"},
	{"#e2f9ec", "#8cddb0", "#2e4d37"},
	{"#dff9f2", "#7adbc2", "#274d41"},
	{"#dff8f7", "#76d8d1", "#234c49"},
	{"#e0f7fc", "#7cd1e5", "#244850"},
	{"#e2f3ff", "#85c5f5", "#264356"},
	{"#e7efff", "#99b4fa", "#2a3d59"},
	{"#ececff", "#aaa8f6", "#323858"},
	{"#e9eeff", "#a1aef8", "#2f3956"},
	{"#e5f0ff", "#90b7fa", "#283d57"},
	{"#e1f3ff", "#7ec3f7", "#234158"},
	{"#e1f2ff", "#7ebff9", "#204459"},
}

// avatar is the CSS style and the initials of a generated avatar.
type avatar struct {
	Style template.CSS
	Text  string
}

// makeAvatar generates a "glass" style avatar with a CSS gradient for a given seed.
// It returns the CSS style + initials of the name that can be printed in HTML.
func makeAvatar(seed, name string) avatar {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(seed))))

	// Pick colours.
	n := len(avatarColors)
	i := int(sum[0]) % n
	one := avatarColors[i]
	two := avatarColors[(i+1+int(sum[1])%3)%n]

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
