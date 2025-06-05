package models

import (
	"testing"
)

func TestBaseQueryValidate(t *testing.T) {
	tests := []struct {
		name     string
		query    BaseQuery
		expected BaseQuery
	}{
		{
			name:  "empty query sets defaults",
			query: BaseQuery{},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 20, Offset: 0},
				Sort:       SortQuery{Order: SortAsc},
			},
		},
		{
			name: "negative page sets to 1",
			query: BaseQuery{
				Pagination: PaginationQuery{Page: -1, PerPage: 10},
			},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 10, Offset: 0},
				Sort:       SortQuery{Order: SortAsc},
			},
		},
		{
			name: "zero per_page sets to 20",
			query: BaseQuery{
				Pagination: PaginationQuery{Page: 2, PerPage: 0},
			},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 2, PerPage: 20, Offset: 20},
				Sort:       SortQuery{Order: SortAsc},
			},
		},
		{
			name: "per_page over 1000 sets to 20",
			query: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 1500},
			},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 20, Offset: 0},
				Sort:       SortQuery{Order: SortAsc},
			},
		},
		{
			name: "valid values preserved",
			query: BaseQuery{
				Pagination: PaginationQuery{Page: 3, PerPage: 50},
				Sort:       SortQuery{Field: "name", Order: SortDesc},
			},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 3, PerPage: 50, Offset: 100},
				Sort:       SortQuery{Field: "name", Order: SortDesc},
			},
		},
		{
			name: "invalid sort order defaults to ASC",
			query: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 10},
				Sort:       SortQuery{Field: "name", Order: "INVALID"},
			},
			expected: BaseQuery{
				Pagination: PaginationQuery{Page: 1, PerPage: 10, Offset: 0},
				Sort:       SortQuery{Field: "name", Order: SortAsc},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.query.Validate()
			
			if tt.query.Pagination.Page != tt.expected.Pagination.Page {
				t.Errorf("Expected page %d, got %d", tt.expected.Pagination.Page, tt.query.Pagination.Page)
			}
			
			if tt.query.Pagination.PerPage != tt.expected.Pagination.PerPage {
				t.Errorf("Expected per_page %d, got %d", tt.expected.Pagination.PerPage, tt.query.Pagination.PerPage)
			}
			
			if tt.query.Pagination.Offset != tt.expected.Pagination.Offset {
				t.Errorf("Expected offset %d, got %d", tt.expected.Pagination.Offset, tt.query.Pagination.Offset)
			}
			
			if tt.query.Sort.Order != tt.expected.Sort.Order {
				t.Errorf("Expected sort order %s, got %s", tt.expected.Sort.Order, tt.query.Sort.Order)
			}
			
			if tt.query.Sort.Field != tt.expected.Sort.Field {
				t.Errorf("Expected sort field %s, got %s", tt.expected.Sort.Field, tt.query.Sort.Field)
			}
		})
	}
}

func TestSubscriberQueryValidate(t *testing.T) {
	query := SubscriberQuery{}
	query.Validate()
	
	// Should set default sort field to "id"
	if query.Sort.Field != "id" {
		t.Errorf("Expected default sort field 'id', got %q", query.Sort.Field)
	}
	
	// Should validate base query
	if query.Pagination.Page != 1 {
		t.Errorf("Expected default page 1, got %d", query.Pagination.Page)
	}
	
	if query.Pagination.PerPage != 20 {
		t.Errorf("Expected default per_page 20, got %d", query.Pagination.PerPage)
	}
}

func TestCampaignQueryValidate(t *testing.T) {
	query := CampaignQuery{}
	query.Validate()
	
	// Should set default sort field to "created_at" and order to DESC
	if query.Sort.Field != "created_at" {
		t.Errorf("Expected default sort field 'created_at', got %q", query.Sort.Field)
	}
	
	if query.Sort.Order != SortDesc {
		t.Errorf("Expected default sort order 'DESC', got %q", query.Sort.Order)
	}
}

func TestListQueryValidate(t *testing.T) {
	query := ListQuery{}
	query.Validate()
	
	// Should set default sort field to "name"
	if query.Sort.Field != "name" {
		t.Errorf("Expected default sort field 'name', got %q", query.Sort.Field)
	}
}

func TestTemplateQueryValidate(t *testing.T) {
	query := TemplateQuery{}
	query.Validate()
	
	// Should set default sort field to "name"
	if query.Sort.Field != "name" {
		t.Errorf("Expected default sort field 'name', got %q", query.Sort.Field)
	}
}

func TestUserQueryValidate(t *testing.T) {
	query := UserQuery{}
	query.Validate()
	
	// Should set default sort field to "email"
	if query.Sort.Field != "email" {
		t.Errorf("Expected default sort field 'email', got %q", query.Sort.Field)
	}
}

func TestMediaQueryValidate(t *testing.T) {
	query := MediaQuery{}
	query.Validate()
	
	// Should set default sort field to "filename"
	if query.Sort.Field != "filename" {
		t.Errorf("Expected default sort field 'filename', got %q", query.Sort.Field)
	}
}

func TestBounceQueryValidate(t *testing.T) {
	query := BounceQuery{}
	query.Validate()
	
	// Should set default sort field to "created_at" and order to DESC
	if query.Sort.Field != "created_at" {
		t.Errorf("Expected default sort field 'created_at', got %q", query.Sort.Field)
	}
	
	if query.Sort.Order != SortDesc {
		t.Errorf("Expected default sort order 'DESC', got %q", query.Sort.Order)
	}
}

func TestSortOrder(t *testing.T) {
	tests := []struct {
		name  string
		order SortOrder
		valid bool
	}{
		{"ASC is valid", SortAsc, true},
		{"DESC is valid", SortDesc, true},
		{"lowercase asc is invalid", SortOrder("asc"), false},
		{"lowercase desc is invalid", SortOrder("desc"), false},
		{"empty is invalid", SortOrder(""), false},
		{"random string is invalid", SortOrder("random"), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.order == SortAsc || tt.order == SortDesc
			if valid != tt.valid {
				t.Errorf("Expected valid=%v for order %q, got %v", tt.valid, tt.order, valid)
			}
		})
	}
}