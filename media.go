package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/media"
	"github.com/labstack/echo"
)

const (
	thumbPrefix   = "thumb_"
	thumbnailSize = 90
)

// imageMimes is the list of image types allowed to be uploaded.
var imageMimes = []string{
	"image/jpg",
	"image/jpeg",
	"image/png",
	"image/svg",
	"image/gif"}

// handleUploadMedia handles media file uploads.
func handleUploadMedia(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		cleanUp = false
	)
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Invalid file uploaded: %v", err))
	}

	// Validate MIME type with the list of allowed types.
	var typ = file.Header.Get("Content-type")
	if ok := validateMIME(typ, imageMimes); !ok {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Unsupported file type (%s) uploaded.", typ))
	}

	// Generate filename
	fName := generateFileName(file.Filename)

	// Read file contents in memory
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error reading file: %s", err))
	}
	defer src.Close()

	// Upload the file.
	fName, err = app.Media.Put(fName, typ, src)
	if err != nil {
		app.Logger.Printf("error uploading file: %v", err)
		cleanUp = true
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error uploading file: %s", err))
	}

	defer func() {
		// If any of the subroutines in this function fail,
		// the uploaded image should be removed.
		if cleanUp {
			app.Media.Delete(fName)
			app.Media.Delete(thumbPrefix + fName)
		}
	}()

	// Create thumbnail from file.
	thumbFile, err := createThumbnail(file)
	if err != nil {
		cleanUp = true
		app.Logger.Printf("error resizing image: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error resizing image: %s", err))
	}

	// Upload thumbnail.
	thumbfName, err := app.Media.Put(thumbPrefix+fName, typ, thumbFile)
	if err != nil {
		cleanUp = true
		app.Logger.Printf("error saving thumbnail: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error saving thumbnail: %s", err))
	}

	uu, err := uuid.NewV4()
	if err != nil {
		app.Logger.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error generating UUID")
	}

	// Write to the DB.
	if _, err := app.Queries.InsertMedia.Exec(uu, fName, thumbfName, 0, 0); err != nil {
		cleanUp = true
		app.Logger.Printf("error inserting uploaded file to db: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error saving uploaded file to db: %s", pqErrMsg(err)))
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetMedia handles retrieval of uploaded media.
func handleGetMedia(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []media.Media
	)

	if err := app.Queries.GetMedia.Select(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching media list: %s", pqErrMsg(err)))
	}

	for i := 0; i < len(out); i++ {
		out[i].URI = app.Media.Get(out[i].Filename)
		out[i].ThumbURI = app.Media.Get(thumbPrefix + out[i].Filename)
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

	var m media.Media
	if err := app.Queries.DeleteMedia.Get(&m, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error deleting media: %s", pqErrMsg(err)))
	}

	app.Media.Delete(m.Filename)
	app.Media.Delete(thumbPrefix + m.Filename)
	return c.JSON(http.StatusOK, okResp{true})
}
