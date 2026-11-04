package core

import (
	"strings"
	"testing"

	"github.com/knadh/listmonk/models"
)

// resolveBounceQueryOrder resolves and sanitizes order and orderBy strings for bounce queries.
func resolveBounceQueryOrder(orderBy, order string) (string, string) {
	cleanOrderBy := models.FieldCreatedAt
	for _, f := range bounceQuerySortFields {
		if f == orderBy {
			cleanOrderBy = f
			break
		}
	}
	cleanOrder := models.OrderDesc
	if order == models.OrderAsc {
		cleanOrder = models.OrderAsc
	}

	if cleanOrderBy == "campaign_name" {
		cleanOrderBy = "campaigns.name"
	}

	return cleanOrderBy, cleanOrder
}

func TestResolveBounceQueryOrder(t *testing.T) {
	tests := []struct {
		orderBy, order       string
		wantOrderBy, wantDir string
	}{
		{"", "", models.FieldCreatedAt, models.OrderDesc},
		{"campaign_name", "asc", "campaigns.name", models.OrderAsc},
		{"email", "asc", "email", models.OrderAsc},
		{"type", "desc", "type", models.OrderDesc},
		{"invalid; DROP TABLE", "desc", models.FieldCreatedAt, models.OrderDesc},
		{"created_at", "invalid", models.FieldCreatedAt, models.OrderDesc},
	}

	for _, tt := range tests {
		gotOrder, gotDir := resolveBounceQueryOrder(tt.orderBy, tt.order)
		if gotOrder != tt.wantOrderBy || gotDir != tt.wantDir {
			t.Errorf("resolveBounceQueryOrder(%q, %q) = (%q, %q), want (%q, %q)",
				tt.orderBy, tt.order, gotOrder, gotDir, tt.wantOrderBy, tt.wantDir)
		}
	}
}

func BenchmarkResolveBounceQueryOrder(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = resolveBounceQueryOrder("campaign_name", "asc")
	}
}

func BenchmarkBounceQueryOrderStmt(b *testing.B) {
	q := "SELECT * FROM bounces ORDER BY %order% OFFSET $1 LIMIT $2"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cleanOrderBy, cleanOrder := resolveBounceQueryOrder("campaign_name", "asc")
		_ = strings.ReplaceAll(q, "%order%", cleanOrderBy+" "+cleanOrder+", bounces.id "+cleanOrder)
	}
}
