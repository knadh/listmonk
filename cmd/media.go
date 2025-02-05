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
func (h *Handler) handleUploadMedia(c echo.Context) error {
	cleanUp := false
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.Ts("media.invalidFile", "error", err.Error()))
	}

	// Read file contents in memory
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
	if !isASCII(file.Filename) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity,
			h.app.i18n.Ts("media.invalidFileName", "name", file.Filename))
	}

	// Validate file extension.
	if !inArray("*", h.app.constants.MediaUpload.Extensions) {
		if ok := inArray(ext, h.app.constants.MediaUpload.Extensions); !ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("media.unsupportedFileType", "type", ext))
		}
	}

	// Sanitize filename.
	fName := makeFilename(file.Filename)

	// Add a random suffix to the filename to ensure uniqueness.
	suffix, _ := generateRandomString(6)
	fName = appendSuffixToFilename(fName, suffix)

	// Upload the file.
	fName, err = h.app.media.Put(fName, contentType, src)
	if err != nil {
		h.app.log.Printf("error uploading file: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			h.app.i18n.Ts("media.errorUploading", "error", err.Error()))
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
			h.app.media.Delete(fName)

			if thumbfName != "" {
				h.app.media.Delete(thumbfName)
			}
		}
	}()

	// Create thumbnail from file for non-vector formats.
	isImage := inArray(ext, imageExts)
	if isImage {
		var thumbFile *bytes.Reader
		thumbFile, width, height, err = processImage(file)
		if err != nil {
			cleanUp = true
			h.app.log.Printf("error resizing image: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				h.app.i18n.Ts("media.errorResizing", "error", err.Error()))
		}
		width = width
		height = height

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

	// Write to the DB.
	meta := models.JSON{}
	if isImage {
		meta = models.JSON{
			"width":  width,
			"height": height,
		}
	}
	m, err := h.app.core.InsertMedia(fName, thumbfName, contentType, meta, h.app.constants.MediaUpload.Provider, h.app.media)
	if err != nil {
		cleanUp = true
		return err
	}
	return c.JSON(http.StatusOK, okResp{m})
}

// handleGetMedia handles retrieval of uploaded media.
func (h *Handler) handleGetMedia(c echo.Context) error {
	var (
		pg    = h.app.paginator.NewFromURL(c.Request().URL.Query())
		query = c.FormValue("query")
		id, _ = strconv.Atoi(c.Param("id"))
	)

	// Fetch one list.
	if id > 0 {
		out, err := h.app.core.GetMedia(id, "", h.app.media)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

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

// deleteMedia handles deletion of uploaded media.
func (h *Handler) handleDeleteMedia(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	fname, err := h.app.core.DeleteMedia(id)
	if err != nil {
		return err
	}

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
