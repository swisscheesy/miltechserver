package shared

import "testing"

func TestFormerCreatorCannotDeleteList(t *testing.T) {
	if CanDeleteList(false, false, false, true) {
		t.Fatal("authorship must not outlive membership")
	}
}

func TestListAuthorizationPolicy(t *testing.T) {
	for _, member := range []bool{false, true} {
		for _, admin := range []bool{false, true} {
			for _, only := range []bool{false, true} {
				wantManage := member && (admin || !only)
				if got := CanManageListItems(member, admin, only); got != wantManage {
					t.Errorf("manage member=%v admin=%v only=%v: %v", member, admin, only, got)
				}
				for _, creator := range []bool{false, true} {
					wantDelete := member && (admin || (!only && creator))
					if got := CanDeleteList(member, admin, only, creator); got != wantDelete {
						t.Errorf("delete member=%v admin=%v only=%v creator=%v: %v", member, admin, only, creator, got)
					}
				}
			}
		}
	}
}
