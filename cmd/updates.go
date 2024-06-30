package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/mod/semver"
)

const updateCheckURL = "https://api.github.com/repos/knadh/listmonk/releases/latest"

type remoteUpdateResp struct {
	Version string `json:"tag_name"`
	URL     string `json:"html_url"`
}

// AppUpdate contains information of a new update available to the app that
// is sent to the frontend.
type AppUpdate struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

var reSemver = regexp.MustCompile(`-(.*)`)

// checkUpdates is a blocking function that checks for updates to the app
// at the given intervals. It uses a HEAD request to check the latest release
// on GitHub without downloading the entire payload. On detecting a new update
// (new semver), it sets the global update status that renders a prompt on the UI.

// checkUpdates checks for updates every 24 hours.
// curVersion (the current version of the listmonk)
func checkUpdates(curVersion string, interval time.Duration, app *App) {
	app.log.Printf("checkUpdates started with current version: %s", curVersion)

	// Strip -* suffix.
	curVersion = reSemver.ReplaceAllString(curVersion, "")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		app.log.Println("checking for remote update")

		resp, err := http.Head(updateCheckURL)
		if err != nil {
			app.log.Printf("error checking for remote update: %v", err)
			continue
		}

		app.log.Printf("response status code: %d", resp.StatusCode)

		if resp.StatusCode != 200 {
			app.log.Printf("non 200 response on remote update check: %d", resp.StatusCode)
			continue
		}

		etag := resp.Header.Get("Etag")
		if etag == "" {
			app.log.Println("no Etag header in remote update response")
			continue
		}

		// There is an update. Set it on the global app state.
		if semver.IsValid(etag) {
			v := reSemver.ReplaceAllString(etag, "")
			if semver.Compare(v, curVersion) > 0 {
				app.Lock()
				app.update = &AppUpdate{
					Version: etag,
					URL:     fmt.Sprintf("%s/releases/latest", updateCheckURL[:len(updateCheckURL)-1]),
				}
				app.Unlock()

				app.log.Printf("new update %s found", etag)
			}
		}
	}
}
