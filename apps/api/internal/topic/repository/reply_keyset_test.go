package repository

import "testing"

func TestFloorBound(t *testing.T) {
	if _, _, ok := FloorBound("asc", 0); ok {
		t.Fatal("from_floor 0 is absent")
	}
	if _, _, ok := FloorBound("desc", -1); ok {
		t.Fatal("negative from_floor is absent")
	}
	sql, arg, ok := FloorBound("asc", 7)
	if !ok || sql != "floor >= ?" || arg != 7 {
		t.Fatalf("asc = %q %d %v", sql, arg, ok)
	}
	sql, arg, ok = FloorBound("desc", 7)
	if !ok || sql != "floor <= ?" || arg != 7 {
		t.Fatalf("desc = %q %d %v", sql, arg, ok)
	}
	sql, arg, ok = FloorBound("asc", 1)
	if !ok || sql != "floor >= ?" || arg != 1 {
		t.Fatalf("asc floor 1 = %q %d %v", sql, arg, ok)
	}
}
