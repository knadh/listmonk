package main

import (
	"encoding/json"
	"io"
	"os"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleImportSubscribers handles the uploading and bulk importing of
// a ZIP file of one or more CSV files.
func handleImportSubscribers(c echo.Context) error {
	app := c.Get("app").(*App)

	// Is an import already running?
	if app.importer.GetStats().Status == subimporter.StatusImporting {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("import.alreadyRunning"))
	}

	// Unmarshal the JSON params.
	var opt subimporter.SessionOpt
	if err := json.Unmarshal([]byte(c.FormValue("params")), &opt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("import.invalidParams", "error", err.Error()))
	}

	// Validate mode.
	if opt.Mode != subimporter.ModeSubscribe && opt.Mode != subimporter.ModeBlocklist {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("import.invalidMode"))
	}

	// If no status is specified, pick a default one.
	if opt.SubStatus == "" {
		switch opt.Mode {
		case subimporter.ModeSubscribe:
			opt.SubStatus = models.SubscriptionStatusUnconfirmed
		case subimporter.ModeBlocklist:
			opt.SubStatus = models.SubscriptionStatusUnsubscribed
		}
	}

	if opt.SubStatus != models.SubscriptionStatusUnconfirmed &&
		opt.SubStatus != models.SubscriptionStatusConfirmed &&
		opt.SubStatus != models.SubscriptionStatusUnsubscribed {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("import.invalidSubStatus"))
	}

	if len(opt.Delim) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("import.invalidDelim"))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("import.invalidFile", "error", err.Error()))
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.CreateTemp("", "listmonk")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}
	defer out.Close()

	if _, err = io.Copy(out, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}

	// Start the importer session.
	opt.Filename = file.Filename
	impSess, err := app.importer.NewSession(opt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("import.errorStarting", "error", err.Error()))
	}
	go impSess.Start()

	if strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		go impSess.LoadCSV(out.Name(), rune(opt.Delim[0]))
	} else {
		// Only 1 CSV from the ZIP is considered. If multiple files have
		// to be processed, counting the net number of lines (to track progress),
		// keeping the global import state (failed / successful) etc. across
		// multiple files becomes complex. Instead, it's just easier for the
		// end user to concat multiple CSVs (if there are multiple in the first)
		// place and upload as one in the first place.
		dir, files, err := impSess.ExtractZIP(out.Name(), 1)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("import.errorProcessingZIP", "error", err.Error()))
		}
		go impSess.LoadCSV(dir+"/"+files[0], rune(opt.Delim[0]))
	}

	return c.JSON(http.StatusOK, okResp{app.importer.GetStats()})
}

// handleGetImportSubscribers returns import statistics.
func handleGetImportSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		s   = app.importer.GetStats()
	)
	return c.JSON(http.StatusOK, okResp{s})
}

// handleGetImportSubscriberStats returns import statistics.
func handleGetImportSubscriberStats(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{string(app.importer.GetLogs())})
}

// handleStopImportSubscribers sends a stop signal to the importer.
// If there's an ongoing import, it'll be stopped, and if an import
// is finished, it's state is cleared.
func handleStopImportSubscribers(c echo.Context) error {
	app := c.Get("app").(*App)
	app.importer.Stop()
	return c.JSON(http.StatusOK, okResp{app.importer.GetStats()})
}
