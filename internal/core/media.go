package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// QueryMedia returns media entries optionally filtered by a query string.
func (c *Core) QueryMedia(provider string, s media.Store, query string, offset, limit int) ([]media.Media, int, error) {
	out := []media.Media{}

	if query != "" {
		query = strings.ToLower(query)
	}

	if err := c.q.QueryMedia.Select(&out, fmt.Sprintf("%%%s%%", query), provider, offset, limit); err != nil {
		return out, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total

		for i := 0; i < len(out); i++ {
			out[i].URL = s.GetURL(out[i].Filename)

			if out[i].Thumb != "" {
				out[i].ThumbURL = null.String{Valid: true, String: s.GetURL(out[i].Thumb)}
			}
		}
	}

	return out, total, nil
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

	out.URL = s.GetURL(out.Filename)
	if out.Thumb != "" {
		out.ThumbURL = null.String{Valid: true, String: s.GetURL(out.Thumb)}
	}

	return out, nil
}

// InsertMedia inserts a new media file into the DB.
func (c *Core) InsertMedia(fileName, thumbName, contentType string, meta models.JSON, provider string, s media.Store) (media.Media, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Write to the DB.
	var newID int
	if err := c.q.InsertMedia.Get(&newID, uu, fileName, thumbName, contentType, provider, meta); err != nil {
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
