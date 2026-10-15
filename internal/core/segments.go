package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// segmentPreviewMaxLimit is the max number of sample subscribers returned in a segment preview.
const segmentPreviewMaxLimit = 50

// SegmentSubscriber is a minimal representation of a subscriber matching a campaign segment.
type SegmentSubscriber struct {
	ID      int         `json:"id"`
	UUID    string      `json:"uuid"`
	Email   string      `json:"email"`
	Name    string      `json:"name"`
	Status  string      `json:"status"`
	Attribs models.JSON `json:"attribs"`
}

// SegmentPreview is the result of a campaign segment dry-run.
type SegmentPreview struct {
	// Matched is the number of subscribers in the lists matching the segment.
	Matched int `json:"matched"`

	// Total is the number of subscribers in the lists the campaign would be sent to without the segment.
	Total int `json:"total"`

	Results []SegmentSubscriber `json:"results"`
}

// checkSegmentExpr checks a user-entered segment expression, which is an SQL expression on the
// subscribers table, same as the advanced subscriber query. eg: subscribers.attribs->>'lang' = 'tr'
//
// As the expression is embedded into a larger query as `AND (expr)`, it is lexically
// checked to ensure that it cannot "break out" of the enclosing parentheses (eg: `1=1) OR (1=1`)
// to bypass the campaign's list, subscription status, and blocklist conditions.
func checkSegmentExpr(q string) error {
	depth := 0
	for i := 0; i < len(q); i++ {
		ch := q[i]
		switch {
		case ch == '\'' || ch == '"':
			// Skip over the quoted string literal or identifier. E'..' strings support backslash escapes.
			escapes := ch == '\'' && i > 0 && (q[i-1] == 'e' || q[i-1] == 'E') &&
				(i == 1 || !isIdentChar(q[i-2]))

			j := i + 1
			for ; j < len(q); j++ {
				if escapes && q[j] == '\\' {
					j++
					continue
				}
				if q[j] == ch {
					// A doubled quote is an escaped quote.
					if j+1 < len(q) && q[j+1] == ch {
						j++
						continue
					}
					break
				}
			}
			if j >= len(q) {
				return errors.New("unterminated quoted string")
			}
			i = j

		case ch == '(':
			depth++

		case ch == ')':
			depth--
			if depth < 0 {
				return errors.New("unbalanced parentheses")
			}

		case ch == ';' || ch == '$':
			return fmt.Errorf("'%c' is not allowed", ch)

		case strings.HasPrefix(q[i:], "--") || strings.HasPrefix(q[i:], "/*"):
			return errors.New("comments are not allowed")
		}
	}

	if depth != 0 {
		return errors.New("unbalanced parentheses")
	}

	return nil
}

func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// ValidateCampaignSegment checks whether the given segment expression is a valid,
// read-only SQL expression that only accesses the allowed tables.
func (c *Core) ValidateCampaignSegment(query string) error {
	if strings.TrimSpace(query) == "" {
		return nil
	}

	_, err := c.prepareSegmentStmt(query, true)
	return err
}

// PreviewCampaignSegment does a dry-run of a segment expression on the given lists and returns
// the number of matching subscribers, the total number of subscribers in the lists (without the segment),
// and a sample of matching subscribers.
func (c *Core) PreviewCampaignSegment(listIDs []int, campType, query string, limit int) (SegmentPreview, error) {
	if limit < 1 || limit > segmentPreviewMaxLimit {
		limit = segmentPreviewMaxLimit
	}

	stmt, err := c.prepareSegmentStmt(query, true)
	if err != nil {
		return SegmentPreview{}, err
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.log.Printf("error preparing segment query: %v", err)
		return SegmentPreview{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	out := SegmentPreview{Results: []SegmentSubscriber{}}

	// Total subscribers in the lists without the segment.
	totalStmt := makeSegmentCountStmt(strings.ReplaceAll(c.q.QueryCampaignSegment, "%query%", "TRUE"))
	if err := tx.Get(&out.Total, totalStmt, pq.Array(listIDs), campType, 0, 0, 0); err != nil {
		return SegmentPreview{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", pqErrMsg(err)))
	}

	// Subscribers matching the segment.
	if err := tx.Get(&out.Matched, makeSegmentCountStmt(stmt), pq.Array(listIDs), campType, 0, 0, 0); err != nil {
		return SegmentPreview{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", pqErrMsg(err)))
	}

	if out.Matched == 0 {
		return out, nil
	}

	// Sample of matching subscribers.
	var subs models.Subscribers
	if err := tx.Select(&subs, stmt, pq.Array(listIDs), campType, 0, 0, limit); err != nil {
		return SegmentPreview{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", pqErrMsg(err)))
	}

	for _, s := range subs {
		out.Results = append(out.Results, SegmentSubscriber{
			ID:      s.ID,
			UUID:    s.UUID,
			Email:   s.Email,
			Name:    s.Name,
			Status:  s.Status,
			Attribs: s.Attribs,
		})
	}

	return out, nil
}

// CountCampaignSegment returns the number of subscribers in the given lists that match the
// segment expression with IDs up to maxSubID (0 for no limit).
func (c *Core) CountCampaignSegment(listIDs []int, campType, query string, maxSubID int) (int, error) {
	stmt, err := c.prepareSegmentStmt(query, true)
	if err != nil {
		return 0, err
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	total := 0
	if err := tx.Get(&total, makeSegmentCountStmt(stmt), pq.Array(listIDs), campType, 0, maxSubID, 0); err != nil {
		return 0, err
	}

	return total, nil
}

// GetCampaignSegmentSubscribers returns the next batch of subscribers in the given lists that match the segment
// expression with IDs in the range (lastSubID, maxSubID]. The query is run in a read-only transaction.
func (c *Core) GetCampaignSegmentSubscribers(listIDs []int, campType, query string, lastSubID, maxSubID, limit int) (models.Subscribers, error) {
	// The query's tables are validated when the campaign starts (CountCampaignSegment)
	// and doesn't have to be repeated for every batch.
	stmt, err := c.prepareSegmentStmt(query, false)
	if err != nil {
		return nil, err
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var out models.Subscribers
	if err := tx.Select(&out, stmt, pq.Array(listIDs), campType, lastSubID, maxSubID, limit); err != nil {
		return nil, err
	}

	return out, nil
}

// prepareSegmentStmt compiles a segment expression into the raw campaign segment query
// and optionally validates the tables accessed by it.
func (c *Core) prepareSegmentStmt(query string, validate bool) (string, error) {
	cond := strings.TrimSpace(query)
	if err := checkSegmentExpr(cond); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", err.Error()))
	}
	if cond == "" {
		cond = "TRUE"
	}

	stmt := strings.ReplaceAll(c.q.QueryCampaignSegment, "%query%", cond)
	if !validate {
		return stmt, nil
	}

	if err := validateQueryTables(c.db, stmt, allowedSubQueryTables, pq.Array([]int{}), "", 0, 0, 0); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("campaigns.errorSegmentQuery", "error", pqErrMsg(err)))
	}

	return stmt, nil
}

// makeSegmentCountStmt wraps a campaign segment query to return the count of results.
func makeSegmentCountStmt(stmt string) string {
	return "SELECT COUNT(*) FROM (" + stmt + ") AS segment"
}
