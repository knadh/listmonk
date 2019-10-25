package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
)

var imageMimes = []string{"image/jpg", "image/jpeg", "image/png", "image/svg", "image/gif"}

const (
	thumbPrefix   = "thumb_"
	thumbnailSize = 90
)

// handleUploadMedia handles media file uploads.
func handleUploadMedia(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		cleanUp = false
	)

	// Upload the file.
	fName, err := uploadFile("file", app.Constants.UploadPath, "", imageMimes, c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error uploading file: %s", err))
	}
	path := filepath.Join(app.Constants.UploadPath, fName)

	defer func() {
		// If any of the subroutines in this function fail,
		// the uploaded image should be removed.
		if cleanUp {
			os.Remove(path)
		}
	}()

	// Create a thumbnail.
	src, err := imaging.Open(path)
	if err != nil {
		cleanUp = true
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error opening image for resizing: %s", err))
	}

	t := imaging.Resize(src, thumbnailSize, 0, imaging.Lanczos)
	if err := imaging.Save(t, fmt.Sprintf("%s/%s%s", app.Constants.UploadPath, thumbPrefix, fName)); err != nil {
		cleanUp = true
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error saving thumbnail: %s", err))
	}

	// Write to the DB.
	if _, err := app.Queries.InsertMedia.Exec(uuid.NewV4(), fName, fmt.Sprintf("%s%s", thumbPrefix, fName), 0, 0); err != nil {
		cleanUp = true
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error saving uploaded file: %s", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetMedia handles retrieval of uploaded media.
func handleGetMedia(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []models.Media
	)

	if err := app.Queries.GetMedia.Select(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching media list: %s", pqErrMsg(err)))
	}

	for i := 0; i < len(out); i++ {
		out[i].URI = fmt.Sprintf("%s/%s", app.Constants.UploadURI, out[i].Filename)
		out[i].ThumbURI = fmt.Sprintf("%s/%s%s", app.Constants.UploadURI, thumbPrefix, out[i].Filename)
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// deleteMedia handles deletion of uploaded media.
func handleDeleteMedia(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	var m models.Media
	if err := app.Queries.DeleteMedia.Get(&m, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error deleting media: %s", pqErrMsg(err)))
	}
	os.Remove(filepath.Join(app.Constants.UploadPath, m.Filename))
	os.Remove(filepath.Join(app.Constants.UploadPath, fmt.Sprintf("%s%s", thumbPrefix, m.Filename)))

	return c.JSON(http.StatusOK, okResp{true})
}
