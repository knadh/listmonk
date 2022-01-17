package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"net/http"
)

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySubscribersByUserId(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 30)

		// The "WHERE ?" bit.
		query   = sanitizeSQLExp(c.FormValue("query"))
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
		userId  = c.Param("userid")
		out     = subsWrapDet{Results: make([]models.SubscriberWithListDetails, 0, 1)}
	)

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "subscribers.id"
	}
	if order != sortAsc && order != sortDesc {
		order = sortDesc
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	stmt := fmt.Sprintf(app.queries.QuerySubscribersByUserIdCount)
	tx, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Execute the readonly query and get the count of results.
	var total = 0
	if err := tx.Get(&total, stmt, userId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// No results.
	if total == 0 {
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Run the query again and fetch the actual data. stmt is the raw SQL query.
	stmt = fmt.Sprintf(app.queries.QuerySubscribersByUserid, cond, orderBy, order)
	if err := tx.Select(&out.Results, stmt, pg.Offset, pg.Limit, userId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	out.Query = query
	if len(out.Results) == 0 {
		out.Results = make(models.SubscribersWithDetails, 0)
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Total = total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}
