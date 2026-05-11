package agentapi

import "testing"

func TestResourceBulkDeleteInputIDs(t *testing.T) {
	checks, err := (checkBulkDeleteInput{Body: bulkIDsBody{IDs: []int64{9, 8, 9}}}).ids()
	if err != nil {
		t.Fatalf("check ids returned error: %v", err)
	}
	if len(checks) != 2 || checks[0] != 8 || checks[1] != 9 {
		t.Fatalf("check ids = %#v, want [8 9]", checks)
	}

	reports, err := (queryBulkDeleteInput{Body: bulkIDsBody{IDs: []int64{4, 2, 4}}}).ids()
	if err != nil {
		t.Fatalf("report ids returned error: %v", err)
	}
	if len(reports) != 2 || reports[0] != 2 || reports[1] != 4 {
		t.Fatalf("report ids = %#v, want [2 4]", reports)
	}
}
