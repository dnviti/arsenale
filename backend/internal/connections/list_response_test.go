package connections

import "testing"

func TestNormalizeListResponseInitializesEmptySlices(t *testing.T) {
	resp := normalizeListResponse(listResponse{})

	if resp.Own == nil {
		t.Fatal("expected own slice to be initialized")
	}
	if resp.Shared == nil {
		t.Fatal("expected shared slice to be initialized")
	}
	if resp.Team == nil {
		t.Fatal("expected team slice to be initialized")
	}
	if len(resp.Own) != 0 || len(resp.Shared) != 0 || len(resp.Team) != 0 {
		t.Fatal("expected normalized slices to be empty")
	}
}
