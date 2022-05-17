package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo/v4"
)

type i18nLang struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type i18nLangRaw struct {
	Code string `json:"_.code"`
	Name string `json:"_.name"`
}

// handleGetI18nLang returns the JSON language pack given the language code.
func handleGetI18nLang(c echo.Context) error {
	app := c.Get("app").(*App)

	lang := c.Param("lang")
	if len(lang) > 6 || reLangCode.MatchString(lang) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid language code.")
	}

	i, ok, err := getI18nLang(lang, app.fs)
	if err != nil && !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Unknown language.")
	}

	return c.JSON(http.StatusOK, okResp{json.RawMessage(i.JSON())})
}

// getI18nLangList returns the list of available i18n languages.
func getI18nLangList(lang string, app *App) ([]i18nLang, error) {
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

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Code < out[j].Code
	})

	return out, nil
}

// The bool indicates whether the specified language could be loaded. If it couldn't
// be, the app shouldn't halt but throw a warning.
func getI18nLang(lang string, fs stuffbin.FileSystem) (*i18n.I18n, bool, error) {
	const def = "en"

	b, err := fs.Read(fmt.Sprintf("/i18n/%s.json", def))
	if err != nil {
		return nil, false, fmt.Errorf("error reading default i18n language file: %s: %v", def, err)
	}

	// Initialize with the default language.
	i, err := i18n.New(b)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling i18n language: %s: %v", lang, err)
	}

	// Load the selected language on top of it.
	b, err = fs.Read(fmt.Sprintf("/i18n/%s.json", lang))
	if err != nil {
		return i, true, fmt.Errorf("error reading i18n language file: %s: %v", lang, err)
	}

	if err := i.Load(b); err != nil {
		return i, true, fmt.Errorf("error loading i18n language file: %s: %v", lang, err)
	}

	return i, true, nil
}
