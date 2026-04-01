package folders

import "testing"

func TestNormalizeListResponseInitializesEmptySlices(t *testing.T) {
	resp := normalizeListResponse(listResponse{})

	if resp.Personal == nil {
		t.Fatal("expected personal slice to be initialized")
	}
	if resp.Team == nil {
		t.Fatal("expected team slice to be initialized")
	}
	if len(resp.Personal) != 0 || len(resp.Team) != 0 {
		t.Fatal("expected normalized slices to be empty")
	}
}
