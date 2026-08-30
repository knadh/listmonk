package main

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

// avatarColors is the pool of colours to use in generated avatars. Based on
// (CC0, https://www.dicebear.com/styles/glass).
var avatarColors = []struct {
	light string
	mid   string
	dark  string
}{
	{"#fde2e2", "#f08c8c", "#411010"},
	{"#fde6e2", "#f09d8c", "#411810"},
	{"#fdebe2", "#f0ae8c", "#412110"},
	{"#fdf0e2", "#f0be8c", "#412910"},
	{"#fdf4e2", "#f0cf8c", "#413110"},
	{"#fdf9e2", "#f0e08c", "#413910"},
	{"#fdfde2", "#f0f08c", "#414110"},
	{"#f9fde2", "#e0f08c", "#394110"},
	{"#f4fde2", "#cff08c", "#314110"},
	{"#f0fde2", "#bef08c", "#294110"},
	{"#ebfde2", "#aef08c", "#214110"},
	{"#e6fde2", "#9df08c", "#184110"},
	{"#e2fde2", "#8cf08c", "#104110"},
	{"#e2fde6", "#8cf09d", "#104118"},
	{"#e2fdeb", "#8cf0ae", "#104121"},
	{"#e2fdf0", "#8cf0be", "#104129"},
	{"#e2fdf4", "#8cf0cf", "#104131"},
	{"#e2fdf9", "#8cf0e0", "#104139"},
	{"#e2fdfd", "#8cf0f0", "#104141"},
	{"#e2f9fd", "#8ce0f0", "#103941"},
	{"#e2f4fd", "#8ccff0", "#103141"},
	{"#e2f0fd", "#8cbef0", "#102941"},
	{"#e2ebfd", "#8caef0", "#102141"},
	{"#e2e6fd", "#8c9df0", "#101841"},
	{"#e2e2fd", "#8c8cf0", "#101041"},
	{"#e6e2fd", "#9d8cf0", "#181041"},
	{"#ebe2fd", "#ae8cf0", "#211041"},
	{"#f0e2fd", "#be8cf0", "#291041"},
	{"#f4e2fd", "#cf8cf0", "#311041"},
	{"#f9e2fd", "#e08cf0", "#391041"},
	{"#fde2fd", "#f08cf0", "#411041"},
	{"#fde2f9", "#f08ce0", "#411039"},
	{"#fde2f0", "#f08cbe", "#411029"},
	{"#fde2eb", "#f08cae", "#411021"},
	{"#fde2e6", "#f08c9d", "#411018"},
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
