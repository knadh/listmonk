package main

import (
	"encoding/json"
	"testing"

	"github.com/knadh/listmonk/models"
)

func intPtr(v int) *int {
	return &v
}

func TestOIDCClaimMatches(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
		match    bool
	}{
		{
			name:     "matching string",
			raw:      `"DSI"`,
			expected: "DSI",
			match:    true,
		},
		{
			name:     "non matching string",
			raw:      `"RH"`,
			expected: "DSI",
			match:    false,
		},
		{
			name:     "case sensitive",
			raw:      `"dsi"`,
			expected: "DSI",
			match:    false,
		},
		{
			name:     "no substring matching",
			raw:      `"DSI-PARIS"`,
			expected: "DSI",
			match:    false,
		},
		{
			name:     "no implicit trim",
			raw:      `" DSI "`,
			expected: "DSI",
			match:    false,
		},
		{
			name:     "matching array",
			raw:      `["users","marketing","admins"]`,
			expected: "marketing",
			match:    true,
		},
		{
			name:     "non matching array",
			raw:      `["users","admins"]`,
			expected: "marketing",
			match:    false,
		},
		{
			name:     "empty array",
			raw:      `[]`,
			expected: "marketing",
			match:    false,
		},
		{
			name:     "null",
			raw:      `null`,
			expected: "",
			match:    false,
		},
		{
			name:     "number",
			raw:      `123`,
			expected: "123",
			match:    false,
		},
		{
			name:     "boolean",
			raw:      `true`,
			expected: "true",
			match:    false,
		},
		{
			name:     "object",
			raw:      `{"name":"DSI"}`,
			expected: "DSI",
			match:    false,
		},
		{
			name:     "mixed array",
			raw:      `["users",123,"marketing"]`,
			expected: "marketing",
			match:    false,
		},
		{
			name:     "invalid JSON",
			raw:      `[`,
			expected: "marketing",
			match:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oidcClaimMatches(json.RawMessage(tt.raw), tt.expected)
			if got != tt.match {
				t.Fatalf("oidcClaimMatches(%s, %q) = %v, want %v",
					tt.raw, tt.expected, got, tt.match)
			}
		})
	}
}

func TestResolveOIDCRoles(t *testing.T) {
	const (
		defaultUserRoleID = 2
		defaultListRoleID = 1
	)

	tests := []struct {
		name       string
		claims     map[string]json.RawMessage
		mappings   []models.OIDCRoleMapping
		wantUserID int
		wantListID int
	}{
		{
			name:       "no mappings uses defaults",
			claims:     map[string]json.RawMessage{},
			mappings:   nil,
			wantUserID: defaultUserRoleID,
			wantListID: defaultListRoleID,
		},
		{
			name: "missing claim uses defaults",
			claims: map[string]json.RawMessage{
				"email": json.RawMessage(`"alice@example.org"`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "departement",
					Match:      "DSI",
					UserRoleID: intPtr(3),
					ListRoleID: intPtr(10),
				},
			},
			wantUserID: defaultUserRoleID,
			wantListID: defaultListRoleID,
		},
		{
			name: "string claim overrides both roles",
			claims: map[string]json.RawMessage{
				"departement": json.RawMessage(`"DSI"`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "departement",
					Match:      "DSI",
					UserRoleID: intPtr(3),
					ListRoleID: intPtr(10),
				},
			},
			wantUserID: 3,
			wantListID: 10,
		},
		{
			name: "array claim matches contained value",
			claims: map[string]json.RawMessage{
				"groups": json.RawMessage(`["users","marketing","admins"]`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "groups",
					Match:      "marketing",
					ListRoleID: intPtr(12),
				},
			},
			wantUserID: defaultUserRoleID,
			wantListID: 12,
		},
		{
			name: "user role only preserves default list role",
			claims: map[string]json.RawMessage{
				"listmonk_role": json.RawMessage(`"editor"`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "listmonk_role",
					Match:      "editor",
					UserRoleID: intPtr(3),
				},
			},
			wantUserID: 3,
			wantListID: defaultListRoleID,
		},
		{
			name: "list role only preserves default user role",
			claims: map[string]json.RawMessage{
				"departement": json.RawMessage(`"RH"`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "departement",
					Match:      "RH",
					ListRoleID: intPtr(11),
				},
			},
			wantUserID: defaultUserRoleID,
			wantListID: 11,
		},
		{
			name: "first matching mapping wins",
			claims: map[string]json.RawMessage{
				"departement": json.RawMessage(`"DSI"`),
				"groups":      json.RawMessage(`["users","listmonk-admin"]`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "groups",
					Match:      "listmonk-admin",
					UserRoleID: intPtr(1),
					ListRoleID: intPtr(1),
				},
				{
					Claim:      "departement",
					Match:      "DSI",
					UserRoleID: intPtr(3),
					ListRoleID: intPtr(10),
				},
			},
			wantUserID: 1,
			wantListID: 1,
		},
		{
			name: "first non match continues to second mapping",
			claims: map[string]json.RawMessage{
				"departement": json.RawMessage(`"DSI"`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "departement",
					Match:      "RH",
					ListRoleID: intPtr(11),
				},
				{
					Claim:      "departement",
					Match:      "DSI",
					ListRoleID: intPtr(10),
				},
			},
			wantUserID: defaultUserRoleID,
			wantListID: 10,
		},
		{
			name: "unsupported claim type is ignored",
			claims: map[string]json.RawMessage{
				"departement": json.RawMessage(`{"name":"DSI"}`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "departement",
					Match:      "DSI",
					UserRoleID: intPtr(3),
					ListRoleID: intPtr(10),
				},
			},
			wantUserID: defaultUserRoleID,
			wantListID: defaultListRoleID,
		},
		{
			name: "array value order does not affect mapping priority",
			claims: map[string]json.RawMessage{
				"groups": json.RawMessage(`["marketing","listmonk-admin"]`),
			},
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "groups",
					Match:      "listmonk-admin",
					UserRoleID: intPtr(1),
					ListRoleID: intPtr(1),
				},
				{
					Claim:      "groups",
					Match:      "marketing",
					ListRoleID: intPtr(12),
				},
			},
			wantUserID: 1,
			wantListID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, gotListID := resolveOIDCRoles(
				tt.claims,
				tt.mappings,
				defaultUserRoleID,
				defaultListRoleID,
			)

			if gotUserID != tt.wantUserID {
				t.Errorf("user role = %d, want %d", gotUserID, tt.wantUserID)
			}

			if gotListID != tt.wantListID {
				t.Errorf("list role = %d, want %d", gotListID, tt.wantListID)
			}
		})
	}
}

func TestValidateOIDCRoleMappings(t *testing.T) {
	userRoleIDs := map[int]struct{}{
		1: {},
		5: {},
	}

	listRoleIDs := map[int]struct{}{
		6: {},
		7: {},
	}

	tests := []struct {
		name     string
		mappings []models.OIDCRoleMapping
		wantErr  bool
	}{
		{
			name: "valid mapping with both roles",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "department",
					Match:      "AN",
					UserRoleID: intPtr(5),
					ListRoleID: intPtr(6),
				},
			},
		},
		{
			name: "valid user role only",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "groups",
					Match:      "listmonk-admin",
					UserRoleID: intPtr(1),
				},
			},
		},
		{
			name: "valid list role only",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "department",
					Match:      "AN",
					ListRoleID: intPtr(6),
				},
			},
		},
		{
			name: "missing claim",
			mappings: []models.OIDCRoleMapping{
				{
					Match:      "AN",
					UserRoleID: intPtr(5),
				},
			},
			wantErr: true,
		},
		{
			name: "missing match",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "department",
					UserRoleID: intPtr(5),
				},
			},
			wantErr: true,
		},
		{
			name: "no target role",
			mappings: []models.OIDCRoleMapping{
				{
					Claim: "department",
					Match: "AN",
				},
			},
			wantErr: true,
		},
		{
			name: "unknown user role",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "department",
					Match:      "AN",
					UserRoleID: intPtr(999),
				},
			},
			wantErr: true,
		},
		{
			name: "unknown list role",
			mappings: []models.OIDCRoleMapping{
				{
					Claim:      "department",
					Match:      "AN",
					ListRoleID: intPtr(999),
				},
			},
			wantErr: true,
		},
		{
			name:     "empty mappings are valid",
			mappings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOIDCRoleMappings(
				tt.mappings,
				userRoleIDs,
				listRoleIDs,
			)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
