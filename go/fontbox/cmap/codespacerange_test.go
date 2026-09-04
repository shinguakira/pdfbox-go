package cmap

import "testing"

// Port of org.apache.fontbox.cmap.TestCodespaceRange.

func TestCodespaceRangeCodeLength(t *testing.T) {
	range1, err := NewCodespaceRange([]byte{0x00}, []byte{0x20})
	if err != nil {
		t.Fatalf("NewCodespaceRange: %v", err)
	}
	if got := range1.CodeLength(); got != 1 {
		t.Errorf("CodeLength() = %d, want 1", got)
	}

	range2, err := NewCodespaceRange([]byte{0x00, 0x00}, []byte{0x01, 0x20})
	if err != nil {
		t.Fatalf("NewCodespaceRange: %v", err)
	}
	if got := range2.CodeLength(); got != 2 {
		t.Errorf("CodeLength() = %d, want 2", got)
	}
}

func TestCodespaceRangeConstructor(t *testing.T) {
	// PDFBOX-4923 "1 begincodespacerange <00> <ffff> endcodespacerange" case is
	// accepted
	if _, err := NewCodespaceRange([]byte{0x00}, []byte{0xff, 0xff}); err != nil {
		t.Errorf("a one-byte zero start with a two-byte end was rejected: %v", err)
	}
	// other cases of different lengths are not
	if _, err := NewCodespaceRange([]byte{0x01}, []byte{0x01, 0x20}); err == nil {
		t.Error("a one-byte non-zero start with a two-byte end was accepted")
	}
}

func TestCodespaceRangeMatches(t *testing.T) {
	range1, err := NewCodespaceRange([]byte{0x00}, []byte{0xA0})
	if err != nil {
		t.Fatalf("NewCodespaceRange: %v", err)
	}
	oneByte := []struct {
		code []byte
		want bool
		why  string
	}{
		{[]byte{0x00}, true, "the start value"},
		{[]byte{0xA0}, true, "the end value"},
		{[]byte{0x10}, true, "a value within range"},
		{[]byte{0xA1}, false, "the first value out of range"},
		{[]byte{0xD0}, false, "a value out of range"},
		{[]byte{0x00, 0x10}, false, "a value with a different code length"},
	}
	for _, c := range oneByte {
		if got := range1.Matches(c.code); got != c.want {
			t.Errorf("range1.Matches(%v) = %v, want %v (%s)", c.code, got, c.want, c.why)
		}
	}

	range2, err := NewCodespaceRange([]byte{0x81, 0x40}, []byte{0x9F, 0xFC})
	if err != nil {
		t.Fatalf("NewCodespaceRange: %v", err)
	}
	twoByte := []struct {
		code []byte
		want bool
		why  string
	}{
		{[]byte{0x81, 0x40}, true, "the lower start value"},
		{[]byte{0x81, 0xFC}, true, "the lower end value"},
		{[]byte{0x9F, 0x40}, true, "the higher start value"},
		{[]byte{0x81, 0x65}, true, "a value within the lower range"},
		{[]byte{0x90, 0x40}, true, "a value within the higher range"},
		{[]byte{0x81, 0xFD}, false, "the first value out of the lower range"},
		{[]byte{0xA0, 0x40}, false, "the first value out of the higher range"},
		{[]byte{0x81, 0x20}, false, "a value out of the lower range"},
		{[]byte{0x10, 0x40}, false, "a value out of the higher range"},
		{[]byte{0x82, 0x20}, false, "a value between start and end but not within the rectangle"},
		{[]byte{0x00}, false, "a value with a different code length"},
	}
	for _, c := range twoByte {
		if got := range2.Matches(c.code); got != c.want {
			t.Errorf("range2.Matches(%v) = %v, want %v (%s)", c.code, got, c.want, c.why)
		}
	}
}
