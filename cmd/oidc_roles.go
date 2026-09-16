package main

import (
	"bytes"
	"encoding/json"

	"github.com/knadh/listmonk/models"
)

// resolveOIDCRoles resolves the user and list roles for a newly auto-created
// OIDC user. Mappings are evaluated in declaration order and the first
// matching mapping wins.
func resolveOIDCRoles(
	claims map[string]json.RawMessage,
	mappings []models.OIDCRoleMapping,
	defaultUserRoleID int,
	defaultListRoleID int,
) (int, int) {
	userRoleID := defaultUserRoleID
	listRoleID := defaultListRoleID

	for _, mapping := range mappings {
		raw, ok := claims[mapping.Claim]
		if !ok {
			continue
		}

		if !oidcClaimMatches(raw, mapping.Match) {
			continue
		}

		if mapping.UserRoleID != nil {
			userRoleID = *mapping.UserRoleID
		}

		if mapping.ListRoleID != nil {
			listRoleID = *mapping.ListRoleID
		}

		break
	}

	return userRoleID, listRoleID
}

// oidcClaimMatches performs an exact match against a supported top-level OIDC
// claim. Supported claim values are strings and arrays containing only strings.
func oidcClaimMatches(raw json.RawMessage, expected string) bool {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value == expected
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return false
	}

	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
