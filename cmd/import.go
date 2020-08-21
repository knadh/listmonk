package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/labstack/echo"
)

// reqImport represents file upload import params.
type reqImport struct {
	Mode      string `json:"mode"`
	Overwrite bool   `json:"overwrite"`
	Delim     string `json:"delim"`
	ListIDs   []int  `json:"lists"`
}

// handleImportSubscribers handles the uploading and bulk importing of
// a ZIP file of one or more CSV files.
func handleImportSubscribers(c echo.Context) error {
	app, _ := c.Get("app").(*App)

	// Is an import already running?
	if app.importer.GetStats().Status == subimporter.StatusImporting {
		return echo.NewHTTPError(http.StatusBadRequest,
			"An import is already running. Wait for it to finish or stop it before trying again.")
	}

	// Unmarsal the JSON params.
	var r reqImport
	if err := json.Unmarshal([]byte(c.FormValue("params")), &r); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Invalid `params` field: %v", err))
	}

	if r.Mode != subimporter.ModeSubscribe && r.Mode != subimporter.ModeBlocklist {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `mode`")
	}

	if len(r.Delim) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest,
			"`delim` should be a single character")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Invalid `file`: %v", err))
	}

	src, err := file.Open()
	if err != nil {
		return err
	}

	defer func() { _ = src.Close() }()

	out, err := ioutil.TempFile("", "listmonk")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error copying uploaded file: %v", err))
	}

	defer func() { _ = out.Close() }()

	if _, err = io.Copy(out, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error copying uploaded file: %v", err))
	}

	// Start the importer session.
	impSess, err := app.importer.NewSession(file.Filename, r.Mode, r.Overwrite, r.ListIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error starting import session: %v", err))
	}

	go impSess.Start()

	if strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		go func() {
			_ = impSess.LoadCSV(out.Name(), rune(r.Delim[0]))
		}()
	} else {
		// Only 1 CSV from the ZIP is considered. If multiple files have
		// to be processed, counting the net number of lines (to track progress),
		// keeping the global import state (failed / successful) etc. across
		// multiple files becomes complex. Instead, it's just easier for the
		// end user to concat multiple CSVs (if there are multiple in the first)
		// place and uploada as one in the first place.
		dir, files, err := impSess.ExtractZIP(out.Name(), 1)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				fmt.Sprintf("Error processing ZIP file: %v", err))
		}

		go func() {
			_ = impSess.LoadCSV(dir+"/"+files[0], rune(r.Delim[0]))
		}()
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
	app, _ := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{string(app.importer.GetLogs())})
}

// handleStopImportSubscribers sends a stop signal to the importer.
// If there's an ongoing import, it'll be stopped, and if an import
// is finished, it's state is cleared.
func handleStopImportSubscribers(c echo.Context) error {
	app, _ := c.Get("app").(*App)

	app.importer.Stop()

	return c.JSON(http.StatusOK, okResp{app.importer.GetStats()})
}
