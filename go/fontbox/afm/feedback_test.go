package afm

import "testing"

// TestParseIntRadixRejectsValuesJavaRejects pins a defect the slice 3 review
// feedback found: parseIntRadix parsed with a 64-bit size, so an AFM token that
// Java's Integer.parseInt throws on was accepted here. An oversized Characters
// or StartCharMetrics count then reached a loop the reference implementation
// never enters.
func TestParseIntRadixRejectsValuesJavaRejects(t *testing.T) {
	// Integer.MAX_VALUE + 1
	if _, err := parseIntRadix("2147483648", 10); err == nil {
		t.Error("parseIntRadix accepted 2147483648, which Integer.parseInt rejects")
	}
	// Integer.MIN_VALUE - 1
	if _, err := parseIntRadix("-2147483649", 10); err == nil {
		t.Error("parseIntRadix accepted -2147483649, which Integer.parseInt rejects")
	}

	// the two ends Java does accept
	if got, err := parseIntRadix("2147483647", 10); err != nil || got != 2147483647 {
		t.Errorf("parseIntRadix(\"2147483647\") = %d, %v, want 2147483647, nil", got, err)
	}
	if got, err := parseIntRadix("-2147483648", 10); err != nil || got != -2147483648 {
		t.Errorf("parseIntRadix(\"-2147483648\") = %d, %v, want -2147483648, nil", got, err)
	}

	// and the hexadecimal form the character metrics use
	if got, err := parseIntRadix("ff", 16); err != nil || got != 255 {
		t.Errorf("parseIntRadix(\"ff\", 16) = %d, %v, want 255, nil", got, err)
	}
}
