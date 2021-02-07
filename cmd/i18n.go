package main

import (
	"encoding/json"
	"fmt"
)

type i18nLang struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type i18nLangRaw struct {
	Code string `json:"_.code"`
	Name string `json:"_.name"`
}

// geti18nLangList returns the list of available i18n languages.
func geti18nLangList(lang string, app *App) ([]i18nLang, error) {
	list, err := app.fs.Glob("/i18n/*.json")
	if err != nil {
		return nil, err
	}

	var out []i18nLang
	for _, l := range list {
		b, err := app.fs.Get(l)
		if err != nil {
			return out, fmt.Errorf("error reading lang file: %s: %v", l, err)
		}

		var lang i18nLangRaw
		if err := json.Unmarshal(b.ReadBytes(), &lang); err != nil {
			return out, fmt.Errorf("error parsing lang file: %s: %v", l, err)
		}

		out = append(out, i18nLang{
			Code: lang.Code,
			Name: lang.Name,
		})
	}

	return out, nil
}
