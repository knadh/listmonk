// i18n is a simple package that translates strings using a language map.
// It mimics some functionality of the vue-i18n library so that the same JSON
// language map may be used in the JS frontend and the Go backend.
package i18n

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// I18n offers translation functions over a language map.
type I18n struct {
	code    string `json:"code"`
	name    string `json:"name"`
	langMap map[string]string
}

var reParam = regexp.MustCompile(`(?i)\{([a-z0-9-.]+)\}`)

// New returns an I18n instance.
func New(b []byte) (*I18n, error) {
	var l map[string]string
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}

	code, ok := l["_.code"]
	if !ok {
		return nil, errors.New("missing _.code field in language file")
	}

	name, ok := l["_.name"]
	if !ok {
		return nil, errors.New("missing _.name field in language file")
	}

	return &I18n{
		langMap: l,
		code:    code,
		name:    name,
	}, nil
}

// Load loads a JSON language map into the instance overwriting
// existing keys that conflict.
func (i *I18n) Load(b []byte) error {
	var l map[string]string
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}

	for k, v := range l {
		i.langMap[k] = v
	}

	return nil
}

// Name returns the canonical name of the language.
func (i *I18n) Name() string {
	return i.name
}

// Code returns the ISO code of the language.
func (i *I18n) Code() string {
	return i.code
}

// JSON returns the languagemap as raw JSON.
func (i *I18n) JSON() []byte {
	b, _ := json.Marshal(i.langMap)
	return b
}

// T returns the translation for the given key similar to vue i18n's t().
func (i *I18n) T(key string) string {
	s, ok := i.langMap[key]
	if !ok {
		return key
	}

	return i.getSingular(s)
}

// Ts returns the translation for the given key similar to vue i18n's t()
// and substitutes the params in the given map in the translated value.
// In the language values, the substitutions are represented as: {key}
// The params and values are received as a pairs of succeeding strings.
// That is, the number of these arguments should be an even number.
// eg: Ts("globals.message.notFound",
//         "name", "campaigns",
//         "error", err)
func (i *I18n) Ts(key string, params ...string) string {
	if len(params)%2 != 0 {
		return key + `: Invalid arguments`
	}

	s, ok := i.langMap[key]
	if !ok {
		return key
	}

	s = i.getSingular(s)
	for n := 0; n < len(params); n += 2 {
		// If there are {params} in the param values, substitute them.
		val := i.subAllParams(params[n+1])
		s = strings.ReplaceAll(s, `{`+params[n]+`}`, val)
	}

	return s
}

// Tc returns the translation for the given key similar to vue i18n's tc().
// It expects the language string in the map to be of the form `Singular | Plural` and
// returns `Plural` if n > 1, or `Singular` otherwise.
func (i *I18n) Tc(key string, n int) string {
	s, ok := i.langMap[key]
	if !ok {
		return key
	}

	// Plural.
	if n > 1 {
		return i.getPlural(s)
	}

	return i.getSingular(s)
}

// getSingular returns the singular term from the vuei18n pipe separated value.
// singular term | plural term
func (i *I18n) getSingular(s string) string {
	if !strings.Contains(s, "|") {
		return s
	}

	return strings.TrimSpace(strings.Split(s, "|")[0])
}

// getSingular returns the plural term from the vuei18n pipe separated value.
// singular term | plural term
func (i *I18n) getPlural(s string) string {
	if !strings.Contains(s, "|") {
		return s
	}

	chunks := strings.Split(s, "|")
	if len(chunks) == 2 {
		return strings.TrimSpace(chunks[1])
	}

	return strings.TrimSpace(chunks[0])
}

// subAllParams recursively resolves and replaces all {params} in a string.
func (i *I18n) subAllParams(s string) string {
	if !strings.Contains(s, `{`) {
		return s
	}

	parts := reParam.FindAllStringSubmatch(s, -1)
	if len(parts) < 1 {
		return s
	}

	for _, p := range parts {
		s = strings.ReplaceAll(s, p[0], i.T(p[1]))
	}

	return i.subAllParams(s)
}
