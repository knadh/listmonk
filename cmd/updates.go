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
// at the given intervals. On detecting a new update (new semver), it
// sets the global update status that renders a prompt on the UI.
func checkUpdates(curVersion string, interval time.Duration, app *App) {
	// Strip -* suffix.
	curVersion = reSemver.ReplaceAllString(curVersion, "")
	time.Sleep(time.Second * 1)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := http.Get(updateCheckURL)
		if err != nil {
			app.log.Printf("error checking for remote update: %v", err)
			continue
		}

		if resp.StatusCode != 200 {
			app.log.Printf("non 200 response on remote update check: %d", resp.StatusCode)
			continue
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			app.log.Printf("error reading remote update payload: %v", err)
			continue
		}
		resp.Body.Close()

		var up remoteUpdateResp
		if err := json.Unmarshal(b, &up); err != nil {
			app.log.Printf("error unmarshalling remote update payload: %v", err)
			continue
		}

		// There is an update. Set it on the global app state.
		if semver.IsValid(up.Version) {
			v := reSemver.ReplaceAllString(up.Version, "")
			if semver.Compare(v, curVersion) > 0 {
				app.Lock()
				app.update = &AppUpdate{
					Version: up.Version,
					URL:     up.URL,
				}
				app.Unlock()

				app.log.Printf("new update %s found", up.Version)
			}
		}
	}
}
