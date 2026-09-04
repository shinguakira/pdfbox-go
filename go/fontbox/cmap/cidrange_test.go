package cmap

import "testing"

// Port of org.apache.fontbox.cmap.CIDRangeTest.

func TestCIDRangeOneByte(t *testing.T) {
	cidRange := newCIDRange(0, 20, 65, 1)
	if got := cidRange.CodeLength(); got != 1 {
		t.Errorf("CodeLength() = %d, want 1", got)
	}
	if got := cidRange.MapBytes([]byte{0}); got != 65 {
		t.Errorf("MapBytes({0}) = %d, want 65", got)
	}
	if got := cidRange.MapBytes([]byte{10}); got != 75 {
		t.Errorf("MapBytes({10}) = %d, want 75", got)
	}
	// out of range
	if got := cidRange.MapBytes([]byte{30}); got != -1 {
		t.Errorf("MapBytes({30}) = %d, want -1", got)
	}
	// wrong code length
	if got := cidRange.MapBytes([]byte{0, 10}); got != -1 {
		t.Errorf("MapBytes({0, 10}) = %d, want -1", got)
	}

	if got := cidRange.Map(0, 1); got != 65 {
		t.Errorf("Map(0, 1) = %d, want 65", got)
	}
	if got := cidRange.Map(10, 1); got != 75 {
		t.Errorf("Map(10, 1) = %d, want 75", got)
	}
	// out of range
	if got := cidRange.Map(30, 1); got != -1 {
		t.Errorf("Map(30, 1) = %d, want -1", got)
	}
	// wrong code length
	if got := cidRange.Map(10, 2); got != -1 {
		t.Errorf("Map(10, 2) = %d, want -1", got)
	}

	if got := cidRange.Unmap(65); got != 0 {
		t.Errorf("Unmap(65) = %d, want 0", got)
	}
	if got := cidRange.Unmap(75); got != 10 {
		t.Errorf("Unmap(75) = %d, want 10", got)
	}
	// out of range
	if got := cidRange.Unmap(100); got != -1 {
		t.Errorf("Unmap(100) = %d, want -1", got)
	}
}

func TestCIDRangeTwoByte(t *testing.T) {
	cidRange := newCIDRange(256, 280, 65, 2)
	if got := cidRange.CodeLength(); got != 2 {
		t.Errorf("CodeLength() = %d, want 2", got)
	}
	if got := cidRange.MapBytes([]byte{1, 0}); got != 65 {
		t.Errorf("MapBytes({1, 0}) = %d, want 65", got)
	}
	if got := cidRange.MapBytes([]byte{1, 10}); got != 75 {
		t.Errorf("MapBytes({1, 10}) = %d, want 75", got)
	}
	// out of range
	if got := cidRange.MapBytes([]byte{1, 30}); got != -1 {
		t.Errorf("MapBytes({1, 30}) = %d, want -1", got)
	}
	// wrong code length
	if got := cidRange.MapBytes([]byte{10}); got != -1 {
		t.Errorf("MapBytes({10}) = %d, want -1", got)
	}

	if got := cidRange.Map(256, 2); got != 65 {
		t.Errorf("Map(256, 2) = %d, want 65", got)
	}
	if got := cidRange.Map(266, 2); got != 75 {
		t.Errorf("Map(266, 2) = %d, want 75", got)
	}
	// out of range
	if got := cidRange.Map(290, 2); got != -1 {
		t.Errorf("Map(290, 2) = %d, want -1", got)
	}
	// wrong code length
	if got := cidRange.Map(256, 1); got != -1 {
		t.Errorf("Map(256, 1) = %d, want -1", got)
	}

	if got := cidRange.Unmap(65); got != 256 {
		t.Errorf("Unmap(65) = %d, want 256", got)
	}
	if got := cidRange.Unmap(75); got != 266 {
		t.Errorf("Unmap(75) = %d, want 266", got)
	}
	// out of range
	if got := cidRange.Unmap(100); got != -1 {
		t.Errorf("Unmap(100) = %d, want -1", got)
	}
}
