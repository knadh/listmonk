package core

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetAllMedia returns all uploaded media.
func (c *Core) GetAllMedia(provider string, s media.Store) ([]media.Media, error) {
	out := []media.Media{}
	if err := c.q.GetAllMedia.Select(&out, provider); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	for i := 0; i < len(out); i++ {
		out[i].URL = s.Get(out[i].Filename)
		out[i].ThumbURL = s.Get(out[i].Thumb)
	}

	return out, nil
}

// GetMedia returns a media item.
func (c *Core) GetMedia(id int, uuid string, s media.Store) (media.Media, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out media.Media
	if err := c.q.GetMedia.Get(&out, id, uu); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	out.URL = s.Get(out.Filename)
	out.ThumbURL = s.Get(out.Thumb)

	return out, nil
}

// InsertMedia inserts a new media file into the DB.
func (c *Core) InsertMedia(fileName, thumbName string, meta models.JSON, provider string, s media.Store) (media.Media, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Write to the DB.
	var newID int
	if err := c.q.InsertMedia.Get(&newID, uu, fileName, thumbName, provider, meta); err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return c.GetMedia(newID, "", s)
}

// DeleteMedia deletes a given media item and returns the filename of the deleted item.
func (c *Core) DeleteMedia(id int) (string, error) {
	var fname string
	if err := c.q.DeleteMedia.Get(&fname, id); err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return fname, nil
}
