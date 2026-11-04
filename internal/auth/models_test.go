package auth

import (
	"reflect"
	"testing"
)

func TestGetPermittedLists(t *testing.T) {
	// Super admin short-circuit
	superAdmin := User{
		UserRoleID: SuperAdminRoleID,
	}
	all, ids := superAdmin.GetPermittedLists(PermTypeGet)
	if !all || ids != nil {
		t.Fatalf("expected (true, nil) for super admin, got (%v, %v)", all, ids)
	}

	// Normal user with blanket get_all permission
	userGetAll := User{
		PermissionsMap: map[string]struct{}{
			PermListGetAll: {},
		},
	}
	all, ids = userGetAll.GetPermittedLists(PermTypeGet)
	if !all || ids != nil {
		t.Fatalf("expected (true, nil) for PermListGetAll, got (%v, %v)", all, ids)
	}

	// Normal user with specific list permissions
	userLists := User{
		GetListIDs:    []int{1, 2, 3},
		ManageListIDs: []int{2, 3, 4, 5},
	}

	// Get only
	all, ids = userLists.GetPermittedLists(PermTypeGet)
	if all || !reflect.DeepEqual(ids, []int{1, 2, 3}) {
		t.Errorf("Get only: expected (false, [1 2 3]), got (%v, %v)", all, ids)
	}

	// Manage only
	all, ids = userLists.GetPermittedLists(PermTypeManage)
	if all || !reflect.DeepEqual(ids, []int{2, 3, 4, 5}) {
		t.Errorf("Manage only: expected (false, [2 3 4 5]), got (%v, %v)", all, ids)
	}

	// Both Get and Manage: should combine without duplicates
	all, ids = userLists.GetPermittedLists(PermTypeGet | PermTypeManage)
	expected := []int{1, 2, 3, 4, 5}
	if all || !reflect.DeepEqual(ids, expected) {
		t.Errorf("Get | Manage: expected (false, %v), got (%v, %v)", expected, all, ids)
	}

	// Zero type
	all, ids = userLists.GetPermittedLists(0)
	if all || ids != nil {
		t.Errorf("Zero PermType: expected (false, nil), got (%v, %v)", all, ids)
	}
}

func BenchmarkGetPermittedLists(b *testing.B) {
	u := User{
		GetListIDs:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		ManageListIDs: []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = u.GetPermittedLists(PermTypeGet | PermTypeManage)
	}
}
