package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
    "os"

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
func (a *App) UploadMedia(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("media.invalidFile", "error", err.Error()))
	}

	// Read the file from the HTTP form.
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("media.errorReadingFile", "error", err.Error()))
	}
	defer src.Close()

	var (
		// Naive check for content type and extension.
		ext         = strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
		contentType = file.Header.Get("Content-Type")
	)

	// Validate file extension.
	if !inArray("*", a.cfg.MediaUpload.Extensions) {
		if ok := inArray(ext, a.cfg.MediaUpload.Extensions); !ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("media.unsupportedFileType", "type", ext))
		}
	}

	// Sanitize the filename.
	fName := makeFilename(file.Filename)

	// Optional folder - client may send a folder path and we should store files under it.
	// ...existing code...
    // Optional folder - client may send a folder path and we should store files under it.
    folder := strings.TrimSpace(c.FormValue("folder"))
    if folder != "" {
        // normalize folder: remove leading/trailing slashes
        folder = strings.Trim(folder, "/\\")
+		// Reject traversal and absolute paths
+		if strings.Contains(folder, "..") || filepath.IsAbs(folder) {
+			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("media.invalidFolder"))
+		}
        fName = filepath.ToSlash(filepath.Join(folder, fName))
    }
// ...existing code...

	// If the filename already exists in the DB, make it unique by adding a random suffix.
	if _, err := a.core.GetMedia(0, "", fName, a.media); err == nil {
		suffix, err := generateRandomString(6)
		if err != nil {
			a.log.Printf("error generating random string: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}

		fName = appendSuffixToFilename(fName, suffix)
	}

	// Upload the file to the media store.
	fName, err = a.media.Put(fName, contentType, src)
	if err != nil {
		a.log.Printf("error uploading file: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("media.errorUploading", "error", err.Error()))
	}

	// This keeps track of whether the file has to be deleted from the DB and the store
	// if any of the subsequent steps fail.
	var (
		cleanUp    = false
		thumbfName = ""
	)
	defer func() {
		if cleanUp {
			a.media.Delete(fName)

			if thumbfName != "" {
				a.media.Delete(thumbfName)
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
			a.log.Printf("error resizing image: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("media.errorResizing", "error", err.Error()))
		}
		width = wi
		height = he

		// Upload thumbnail.
		tf, err := a.media.Put(thumbPrefix+fName, contentType, thumbFile)
		if err != nil {
			cleanUp = true
			a.log.Printf("error saving thumbnail: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("media.errorSavingThumbnail", "error", err.Error()))
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
	m, err := a.core.InsertMedia(fName, thumbfName, contentType, meta, a.cfg.MediaUpload.Provider, a.media)
	if err != nil {
		cleanUp = true
		return err
	}

	return c.JSON(http.StatusOK, okResp{m})
}

// GetAllMedia handles retrieval of uploaded media.
func (a *App) GetAllMedia(c echo.Context) error {
	var (
		query = c.FormValue("query")
		folder = c.FormValue("folder")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)
	// Fetch the media items from the DB.
	res, total, err := a.core.QueryMedia(a.cfg.MediaUpload.Provider, a.media, query, pg.Offset, pg.Limit, folder)
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

// GetMedia handles retrieval of a media item by ID.
func (a *App) GetMedia(c echo.Context) error {
	// Fetch the media item from the DB.
	id := getID(c)
	out, err := a.core.GetMedia(id, "", "", a.media)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteMedia handles deletion of uploaded media.
func (a *App) DeleteMedia(c echo.Context) error {

	// Delete the media from the DB. The query returns the filename.
	id := getID(c)
	fname, err := a.core.DeleteMedia(id)
	if err != nil {
		return err
	}

	// Delete the files from the media store.
	a.media.Delete(fname)
	a.media.Delete(thumbPrefix + fname)

	return c.JSON(http.StatusOK, okResp{true})
}

// CreateMediaFolder creates a folder in the media store. For filesystem it ensures
// the directory exists. For S3, it creates a zero-byte object ending with '/'.
func (a *App) CreateMediaFolder(c echo.Context) error {
	folder := strings.TrimSpace(c.FormValue("folder"))
	if folder == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("media.invalidFolder"))
	}

	// Normalize and ensure no leading slash.
	folder = strings.Trim(folder, "/\\")
	if strings.Contains(folder, "..") || filepath.IsAbs(folder) {
+		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("media.invalidFolder"))
+	}
    
	// For filesystem, create the directory under upload path.
	switch prov := a.cfg.MediaUpload.Provider; prov {
	case "filesystem":
		// getDir + join
		dir := getDir(a.cfg.MediaUpload.UploadPath)
		full := filepath.Join(dir, filepath.FromSlash(folder))
		if err := os.MkdirAll(full, 0755); err != nil {
			a.log.Printf("error creating media folder: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("media.errorCreatingFolder", "error", err.Error()))
		}
		return c.JSON(http.StatusOK, okResp{true})
	case "s3":
		// Create a zero-byte object with trailing slash to represent folder.
		// Use media.Store interface: Put accepts a name; create an empty reader.
		name := strings.TrimSuffix(folder, "/") + "/"
		r := bytes.NewReader([]byte{})
		if _, err := a.media.Put(name, "application/octet-stream", r); err != nil {
			a.log.Printf("error creating s3 folder object: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("media.errorCreatingFolder", "error", err.Error()))
		}
		return c.JSON(http.StatusOK, okResp{true})
	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("media.unsupportedProvider"))
	}
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
