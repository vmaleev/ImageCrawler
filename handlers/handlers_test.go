package handlers

import "testing"

func TestDeterministicGUIDConsistency(t *testing.T) {
	host := "example.com"
	id1 := deterministicGUID(host)
	id2 := deterministicGUID(host)
	if id1 != id2 {
		t.Fatalf("expected identical GUIDs, got %s and %s", id1, id2)
	}
	if len(id1) == 0 {
		t.Fatalf("GUID should not be empty")
	}
}
