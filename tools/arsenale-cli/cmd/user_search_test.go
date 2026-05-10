package cmd

import "testing"

func TestBuildUserSearchParamsDefaultsToTenantScope(t *testing.T) {
	params, err := buildUserSearchParams("admin", "", "")
	if err != nil {
		t.Fatalf("buildUserSearchParams() error = %v", err)
	}

	if got := params.Get("q"); got != "admin" {
		t.Fatalf("q = %q, want admin", got)
	}
	if got := params.Get("scope"); got != "tenant" {
		t.Fatalf("scope = %q, want tenant", got)
	}
	if got := params.Get("teamId"); got != "" {
		t.Fatalf("teamId = %q, want empty", got)
	}
}

func TestBuildUserSearchParamsRequiresTeamIDForTeamScope(t *testing.T) {
	if _, err := buildUserSearchParams("admin", "team", ""); err == nil {
		t.Fatal("buildUserSearchParams() error = nil, want error")
	}

	params, err := buildUserSearchParams("admin", "team", "team-1")
	if err != nil {
		t.Fatalf("buildUserSearchParams() error = %v", err)
	}
	if got := params.Get("scope"); got != "team" {
		t.Fatalf("scope = %q, want team", got)
	}
	if got := params.Get("teamId"); got != "team-1" {
		t.Fatalf("teamId = %q, want team-1", got)
	}
}
