package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"github.com/labstack/echo/v4"
)

// handleGetLists retrieves lists with additional metadata like subscriber counts. This may be slow.
func handleGetListsByUserId(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out listsWrap

		pg        = getPagination(c.QueryParams(), 20)
		query     = strings.TrimSpace(c.FormValue("query"))
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		listID, _ = strconv.Atoi(c.Param("id"))
		userId    = c.FormValue("userid")
	)

	// Fetch one list.

	single := false
	if listID > 0 {
		single = true
	}

	println(listID)

	queryStr, stmt := makeSearchQuery(query, orderBy, order, app.queries.QueryListsByUserId)

	if err := db.Select(&out.Results,
		stmt,
		listID,
		queryStr,
		pg.Offset,
		pg.Limit, userId); err != nil {
		app.log.Printf("error fetching lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	if single && len(out.Results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}
	if len(out.Results) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Replace null tags.
	for i, v := range out.Results {
		if v.Tags == nil {
			out.Results[i].Tags = make(pq.StringArray, 0)
		}
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out.Results[0]})
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	if out.PerPage == 0 {
		out.PerPage = out.Total
	}

	return c.JSON(http.StatusOK, okResp{out})
}
