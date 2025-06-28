package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetSQLSnippets retrieves SQL snippets based on the given filters.
func (c *Core) GetSQLSnippets(id int, name string, isActive *bool, offset, limit int) ([]models.SQLSnippet, error) {
	var out []models.SQLSnippet
	if err := c.q.GetSQLSnippets.Select(&out, id, name, isActive, offset, limit); err != nil {
		c.log.Printf("error fetching SQL snippets: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "SQL snippets", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetSQLSnippet retrieves a single SQL snippet by ID or name.
func (c *Core) GetSQLSnippet(id int, name string) (models.SQLSnippet, error) {
	var out models.SQLSnippet
	if err := c.q.GetSQLSnippet.Get(&out, id, name); err != nil {
		c.log.Printf("error fetching SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateSQLSnippet creates a new SQL snippet.
func (c *Core) CreateSQLSnippet(s models.SQLSnippet, createdBy int) (models.SQLSnippet, error) {
	var createdByVal *int
	if createdBy > 0 {
		createdByVal = &createdBy
	}

	var newID int
	if err := c.q.CreateSQLSnippet.Get(&newID, s.Name, s.Description, s.QuerySQL, createdByVal); err != nil {
		c.log.Printf("error creating SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return c.GetSQLSnippet(newID, "")
}

// UpdateSQLSnippet updates a SQL snippet.
func (c *Core) UpdateSQLSnippet(id int, s models.SQLSnippet) (models.SQLSnippet, error) {
	if err := c.q.UpdateSQLSnippet.Exec(s.Name, s.Description, s.QuerySQL, s.IsActive, id); err != nil {
		c.log.Printf("error updating SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return c.GetSQLSnippet(id, "")
}

// DeleteSQLSnippet deletes a SQL snippet.
func (c *Core) DeleteSQLSnippet(id int) error {
	if err := c.q.DeleteSQLSnippet.Exec(id); err != nil {
		c.log.Printf("error deleting SQL snippet: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return nil
}

// ValidateSQLSnippet validates a SQL snippet by attempting to execute it.
func (c *Core) ValidateSQLSnippet(querySQL string) error {
	// Create a read-only transaction for validation
	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "database connection", "error", err.Error()))
	}
	defer tx.Rollback()

	// Construct a test query to validate the SQL snippet
	stmt := fmt.Sprintf(`
		SELECT COUNT(*) FROM subscribers
		LEFT JOIN subscriber_lists ON (
			subscriber_lists.subscriber_id = subscribers.id
		)
		WHERE %s
		LIMIT 1
	`, querySQL)

	// Try to execute the query to validate syntax and permissions
	var count int
	if err := tx.Get(&count, stmt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", err.Error()))
	}

	return nil
}