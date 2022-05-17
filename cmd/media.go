package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/media"
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

	uu, err := uuid.NewV4()
	if err != nil {
		app.log.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Write to the DB.
	if _, err := app.queries.InsertMedia.Exec(uu, fName, thumbfName, app.constants.MediaProvider); err != nil {
		cleanUp = true
		app.log.Printf("error inserting uploaded file to db: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorCreating",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetMedia handles retrieval of uploaded media.
func handleGetMedia(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = []media.Media{}
	)

	if err := app.queries.GetMedia.Select(&out, app.constants.MediaProvider); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	for i := 0; i < len(out); i++ {
		out[i].URL = app.media.Get(out[i].Filename)
		out[i].ThumbURL = app.media.Get(out[i].Thumb)
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

	var m media.Media
	if err := app.queries.DeleteMedia.Get(&m, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	app.media.Delete(m.Filename)
	app.media.Delete(thumbPrefix + m.Filename)
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
