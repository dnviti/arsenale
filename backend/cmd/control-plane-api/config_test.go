package main

import "testing"

func TestParseCSV(t *testing.T) {
	t.Parallel()

	got := parseCSV("192.168.254.1, 192.168.254.3, ,192.168.254.1")
	if len(got) != 2 || got[0] != "192.168.254.1" || got[1] != "192.168.254.3" {
		t.Fatalf("parseCSV returned %#v", got)
	}
}
