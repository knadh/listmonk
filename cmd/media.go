package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/labstack/echo/v4"
)

const (
	thumbPrefix   = "thumb_"
	thumbnailSize = 90
)

// validMimes is the list of image types allowed to be uploaded.
var (
	validMimes = []string{"image/jpg", "image/jpeg", "image/png", "image/gif"}
	validExts  = []string{".jpg", ".jpeg", ".png", ".gif"}
)

// handleUploadMedia handles media file uploads.
func handleUploadMedia(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		cleanUp = false
	)
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("media.invalidFile", "error", err.Error()))
	}

	// Validate file extension.
	ext := filepath.Ext(file.Filename)
	if ok := inArray(ext, validExts); !ok {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("media.unsupportedFileType", "type", ext))
	}

	// Validate file's mime.
	typ := file.Header.Get("Content-type")
	if ok := inArray(typ, validMimes); !ok {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("media.unsupportedFileType", "type", typ))
	}

	// Generate filename
	fName := makeFilename(file.Filename)

	// Read file contents in memory
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorReadingFile", "error", err.Error()))
	}
	defer src.Close()

	// Upload the file.
	fName, err = app.media.Put(fName, typ, src)
	if err != nil {
		app.log.Printf("error uploading file: %v", err)
		cleanUp = true
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorUploading", "error", err.Error()))
	}

	defer func() {
		// If any of the subroutines in this function fail,
		// the uploaded image should be removed.
		if cleanUp {
			app.media.Delete(fName)
			app.media.Delete(thumbPrefix + fName)
		}
	}()

	// Create thumbnail from file.
	thumbFile, err := createThumbnail(file)
	if err != nil {
		cleanUp = true
		app.log.Printf("error resizing image: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorResizing", "error", err.Error()))
	}

	// Upload thumbnail.
	thumbfName, err := app.media.Put(thumbPrefix+fName, typ, thumbFile)
	if err != nil {
		cleanUp = true
		app.log.Printf("error saving thumbnail: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorSavingThumbnail", "error", err.Error()))
	}

	// Write to the DB.
	// TODO: cleanup
	if _, err := app.core.InsertMedia(fName, thumbfName, app.constants.MediaProvider, app.media); err != nil {
		cleanUp = true
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetMedia handles retrieval of uploaded media.
func handleGetMedia(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	// Fetch one list.
	if id > 0 {
		out, err := app.core.GetMedia(id, "", app.media)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	out, err := app.core.GetAllMedia(app.constants.MediaProvider, app.media)
	if err != nil {
		return err
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
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	fname, err := app.core.DeleteMedia(id)
	if err != nil {
		return err
	}

	app.media.Delete(fname)
	app.media.Delete(thumbPrefix + fname)

	return c.JSON(http.StatusOK, okResp{true})
}

// createThumbnail reads the file object and returns a smaller image
func createThumbnail(file *multipart.FileHeader) (*bytes.Reader, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	img, err := imaging.Decode(src)
	if err != nil {
		return nil, err
	}

	// Encode the image into a byte slice as PNG.
	var (
		thumb = imaging.Resize(img, thumbnailSize, 0, imaging.Lanczos)
		out   bytes.Buffer
	)
	if err := imaging.Encode(&out, thumb, imaging.PNG); err != nil {
		return nil, err
	}
	return bytes.NewReader(out.Bytes()), nil
}
