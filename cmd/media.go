package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const (
	thumbPrefix   = "thumb_"
	thumbnailSize = 250
)

var (
	vectorExts = []string{"svg"}
	imageExts  = []string{"gif", "png", "jpg", "jpeg"}
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

	// Read file contents in memory
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorReadingFile", "error", err.Error()))
	}
	defer src.Close()

	var (
		// Naive check for content type and extension.
		ext         = strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
		contentType = file.Header.Get("Content-Type")
	)

	// Validate file extension.
	if !inArray("*", app.constants.MediaUpload.Extensions) {
		if ok := inArray(ext, app.constants.MediaUpload.Extensions); !ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("media.unsupportedFileType", "type", ext))
		}
	}

	// Upload the file.
	fName := makeFilename(file.Filename)
	fName, err = app.media.Put(fName, contentType, src)
	if err != nil {
		app.log.Printf("error uploading file: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("media.errorUploading", "error", err.Error()))
	}

	var (
		thumbfName = ""
		width      = 0
		height     = 0
	)
	defer func() {
		// If any of the subroutines in this function fail,
		// the uploaded image should be removed.
		if cleanUp {
			app.media.Delete(fName)

			if thumbfName != "" {
				app.media.Delete(thumbfName)
			}
		}
	}()

	// Create thumbnail from file for non-vector formats.
	isImage := inArray(ext, imageExts)
	if isImage {
		thumbFile, w, h, err := processImage(file)
		if err != nil {
			cleanUp = true
			app.log.Printf("error resizing image: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("media.errorResizing", "error", err.Error()))
		}
		width = w
		height = h

		// Upload thumbnail.
		tf, err := app.media.Put(thumbPrefix+fName, contentType, thumbFile)
		if err != nil {
			cleanUp = true
			app.log.Printf("error saving thumbnail: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("media.errorSavingThumbnail", "error", err.Error()))
		}
		thumbfName = tf
	}
	if inArray(ext, vectorExts) {
		thumbfName = fName
	}

	// Write to the DB.
	meta := models.JSON{}
	if isImage {
		meta = models.JSON{
			"width":  width,
			"height": height,
		}
	}
	m, err := app.core.InsertMedia(fName, thumbfName, contentType, meta, app.constants.MediaUpload.Provider, app.media)
	if err != nil {
		cleanUp = true
		return err
	}
	return c.JSON(http.StatusOK, okResp{m})
}

// handleGetMedia handles retrieval of uploaded media.
func handleGetMedia(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		pg    = app.paginator.NewFromURL(c.Request().URL.Query())
		query = c.FormValue("query")
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

	res, total, err := app.core.QueryMedia(app.constants.MediaUpload.Provider, app.media, query, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
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

// processImage reads the image file and returns thumbnail bytes and
// the original image's width, and height.
func processImage(file *multipart.FileHeader) (*bytes.Reader, int, int, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, 0, err
	}
	defer src.Close()

	img, err := imaging.Decode(src)
	if err != nil {
		return nil, 0, 0, err
	}

	// Encode the image into a byte slice as PNG.
	var (
		thumb = imaging.Resize(img, thumbnailSize, 0, imaging.Lanczos)
		out   bytes.Buffer
	)
	if err := imaging.Encode(&out, thumb, imaging.PNG); err != nil {
		return nil, 0, 0, err
	}

	b := img.Bounds().Max
	return bytes.NewReader(out.Bytes()), b.X, b.Y, nil
}
