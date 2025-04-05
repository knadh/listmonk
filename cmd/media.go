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

// UploadMedia handles media file uploads.
func (h *Handlers) UploadMedia(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.Ts("media.invalidFile", "error", err.Error()))
	}

	// Read the file from the HTTP form.
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			h.app.i18n.Ts("media.errorReadingFile", "error", err.Error()))
	}
	defer src.Close()

	var (
		// Naive check for content type and extension.
		ext         = strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
		contentType = file.Header.Get("Content-Type")
	)

	// Validate file extension.
	if !inArray("*", h.app.constants.MediaUpload.Extensions) {
		if ok := inArray(ext, h.app.constants.MediaUpload.Extensions); !ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("media.unsupportedFileType", "type", ext))
		}
	}

	// Sanitize the filename.
	fName := makeFilename(file.Filename)

	// If the filename already exists in the DB, make it unique by adding a random suffix.
	if _, err := h.app.core.GetMedia(0, "", fName, h.app.media); err == nil {
		suffix, err := generateRandomString(6)
		if err != nil {
			h.app.log.Printf("error generating random string: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("globals.messages.internalError"))
		}

		fName = appendSuffixToFilename(fName, suffix)
	}

	// Upload the file to the media store.
	fName, err = h.app.media.Put(fName, contentType, src)
	if err != nil {
		h.app.log.Printf("error uploading file: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			h.app.i18n.Ts("media.errorUploading", "error", err.Error()))
	}

	// This keeps track of whether the file has to be deleted from the DB and the store
	// if any of the subsequent steps fail.
	var (
		cleanUp    = false
		thumbfName = ""
	)
	defer func() {
		if cleanUp {
			h.app.media.Delete(fName)

			if thumbfName != "" {
				h.app.media.Delete(thumbfName)
			}
		}
	}()

	// Thumbnail width and height.
	var width, height int

	// Create thumbnail from file for non-vector formats.
	isImage := inArray(ext, imageExts)
	if isImage {
		thumbFile, wi, he, err := processImage(file)
		if err != nil {
			cleanUp = true
			h.app.log.Printf("error resizing image: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				h.app.i18n.Ts("media.errorResizing", "error", err.Error()))
		}
		width = wi
		height = he

		// Upload thumbnail.
		tf, err := h.app.media.Put(thumbPrefix+fName, contentType, thumbFile)
		if err != nil {
			cleanUp = true
			h.app.log.Printf("error saving thumbnail: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				h.app.i18n.Ts("media.errorSavingThumbnail", "error", err.Error()))
		}
		thumbfName = tf
	}
	if inArray(ext, vectorExts) {
		thumbfName = fName
	}

	// Images have metadata.
	meta := models.JSON{}
	if isImage {
		meta = models.JSON{
			"width":  width,
			"height": height,
		}
	}

	// Insert the media into the DB.
	m, err := h.app.core.InsertMedia(fName, thumbfName, contentType, meta, h.app.constants.MediaUpload.Provider, h.app.media)
	if err != nil {
		cleanUp = true
		return err
	}

	return c.JSON(http.StatusOK, okResp{m})
}

// GetMedia handles retrieval of uploaded media.
func (h *Handlers) GetMedia(c echo.Context) error {
	// Fetch one media item from the DB.
	id, _ := strconv.Atoi(c.Param("id"))
	if id > 0 {
		out, err := h.app.core.GetMedia(id, "", "", h.app.media)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get the media from the DB.
	var (
		pg    = h.app.paginator.NewFromURL(c.Request().URL.Query())
		query = c.FormValue("query")
	)
	res, total, err := h.app.core.QueryMedia(h.app.constants.MediaUpload.Provider, h.app.media, query, pg.Offset, pg.Limit)
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

// DeleteMedia handles deletion of uploaded media.
func (h *Handlers) DeleteMedia(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Delete the media from the DB. The query returns the filename.
	fname, err := h.app.core.DeleteMedia(id)
	if err != nil {
		return err
	}

	// Delete the files from the media store.
	h.app.media.Delete(fname)
	h.app.media.Delete(thumbPrefix + fname)

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
