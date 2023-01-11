package core

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetAllMedia returns all uploaded media.
func (c *Core) GetAllMedia(ctx context.Context, provider string, s media.Store) ([]media.Media, error) {
	out := []media.Media{}
	if err := c.q.GetAllMedia.SelectContext(ctx, &out, provider); err != nil {
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
func (c *Core) GetMedia(ctx context.Context, id int, uuid string, s media.Store) (media.Media, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out media.Media
	if err := c.q.GetMedia.GetContext(ctx, &out, id, uu); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	out.URL = s.Get(out.Filename)
	out.ThumbURL = s.Get(out.Thumb)

	return out, nil
}

// InsertMedia inserts a new media file into the DB.
func (c *Core) InsertMedia(ctx context.Context, fileName, thumbName string, meta models.JSON, provider string, s media.Store) (media.Media, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Write to the DB.
	var newID int
	if err := c.q.InsertMedia.GetContext(ctx, &newID, uu, fileName, thumbName, provider, meta); err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return c.GetMedia(ctx, newID, "", s)
}

// DeleteMedia deletes a given media item and returns the filename of the deleted item.
func (c *Core) DeleteMedia(ctx context.Context, id int) (string, error) {
	var fname string
	if err := c.q.DeleteMedia.GetContext(ctx, &fname, id); err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return fname, nil
}
