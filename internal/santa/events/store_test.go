package events

import (
	"strings"
	"testing"
)

func TestExecutionEventWhereUserFilter(t *testing.T) {
	t.Parallel()

	whereSQL, args, err := executionEventWhere(ExecutionEventListParams{User: "root"})
	if err != nil {
		t.Fatalf("executionEventWhere: %v", err)
	}
	if !strings.Contains(whereSQL, "ee.executing_user = $1") {
		t.Fatalf("where SQL = %q, want exact user predicate", whereSQL)
	}
	if len(args) != 1 || args[0] != "root" {
		t.Fatalf("args = %#v, want root as the only predicate arg", args)
	}
}
